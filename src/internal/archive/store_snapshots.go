// store_snapshots.go — snapshot persistence: save/load index state, zip selection, summary counts.
package archive

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"mochila-archive-viewer/src/internal/types"
)

// SaveSnapshot persists a full archive index (media, conversations, messages,
// zip references) for a (platform, user) pair inside a single transaction.
// Existing rows for that pair are deleted and replaced atomically.
func (s *Store) SaveSnapshot(platform string, userId int64, selected []ArchiveFile, idx *types.Index, conversations []types.Conversation) error {
	if idx == nil {
		return errors.New("cannot save empty archive index")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO platform_snapshots(platform, user_id, media_count, zip_count)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(platform, user_id) DO UPDATE SET media_count = excluded.media_count, zip_count = excluded.zip_count
	`, platform, userId, len(idx.Media), len(idx.Zips)); err != nil {
		return err
	}

	for _, table := range []string{"archive_files", "media_items", "json_files", "conversations", "messages"} {
		if _, err := tx.Exec("DELETE FROM "+table+" WHERE platform = ? AND user_id = ?", platform, userId); err != nil {
			return err
		}
	}

	for i, file := range selected {
		if _, err := tx.Exec(`
			INSERT OR REPLACE INTO archive_files(platform, user_id, ordinal, path, name)
			VALUES (?, ?, ?, ?, ?)
		`, platform, userId, i, file.Path, file.Name); err != nil {
			return err
		}
	}

	for _, item := range idx.Media {
		if _, err := tx.Exec(`
			INSERT OR REPLACE INTO media_items(platform, user_id, media_id, zip_index, zip, entry, category, date, year, type, ext)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, platform, userId, item.ID, item.ZipIndex, item.Zip, item.Entry, item.Category, item.Date, item.Year, item.Type, item.Ext); err != nil {
			return err
		}
	}

	for ordinal, item := range idx.JsonFiles {
		if _, err := tx.Exec(`
			INSERT OR REPLACE INTO json_files(platform, user_id, ordinal, zip_index, zip, entry)
			VALUES (?, ?, ?, ?, ?, ?)
		`, platform, userId, ordinal, item.ZipIndex, item.Zip, item.Entry); err != nil {
			return err
		}
	}

	for _, convo := range conversations {
		if _, err := tx.Exec(`
			INSERT OR REPLACE INTO conversations(platform, user_id, conversation_id, title, message_count, saved_count, media_count, last_created)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, platform, userId, convo.ID, convo.Title, convo.MessageCount, convo.SavedCount, convo.MediaCount, convo.LastCreated); err != nil {
			return err
		}

		for ordinal, msg := range convo.Messages {
			if _, err := tx.Exec(`
				INSERT OR REPLACE INTO messages(platform, user_id, conversation_id, ordinal, from_name, content, media_type, created, is_sender, is_saved, media_ids)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, platform, userId, convo.ID, ordinal, msg.From, msg.Content, msg.MediaType, msg.Created, boolToInt(msg.IsSender), boolToInt(msg.IsSaved), msg.MediaIDs); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// SaveSelection persists the chosen zip file paths for a (platform, user) pair.
func (s *Store) SaveSelection(platform string, userId int64, selected []ArchiveFile) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, file := range selected {
		if _, err := tx.Exec(`
			INSERT OR REPLACE INTO archive_files(platform, user_id, ordinal, path, name)
			VALUES (?, ?, ?, ?, ?)
		`, platform, userId, i, file.Path, file.Name); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// LoadSnapshot restores the startup state (selected zips + summary) for a
// (platform, user) pair. Heavy data is fetched on demand via the individual
// Store methods.
func (s *Store) LoadSnapshot(platform string, userId int64) (*Snapshot, error) {
	selected, err := s.loadSelected(platform, userId)
	if err != nil {
		return nil, err
	}
	summary, err := s.loadSummary(platform, userId)
	if err != nil {
		return nil, err
	}
	return &Snapshot{Selected: selected, Summary: summary}, nil
}

// Summary returns the aggregate counts snapshot for a (platform, user) pair.
func (s *Store) Summary(platform string, userId int64) (*IndexSummary, error) {
	return s.loadSummary(platform, userId)
}

func (s *Store) loadSummary(platform string, userId int64) (*IndexSummary, error) {
	var mediaCount, zipCount int
	err := s.db.QueryRow(`
		SELECT media_count, zip_count
		FROM platform_snapshots
		WHERE platform = ? AND user_id = ?
	`, platform, userId).Scan(&mediaCount, &zipCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	years, err := s.loadGroupedCounts(platform, "year", userId)
	if err != nil {
		return nil, err
	}
	types, err := s.loadGroupedCounts(platform, "type", userId)
	if err != nil {
		return nil, err
	}
	categories, err := s.loadGroupedCounts(platform, "category", userId)
	if err != nil {
		return nil, err
	}

	return &IndexSummary{
		Platform:   platform,
		MediaCount: mediaCount,
		ZipCount:   zipCount,
		Years:      years,
		Types:      types,
		Categories: categories,
	}, nil
}

// loadGroupedCounts uses explicit query constants to avoid string interpolation
// for column names — see sql.md standard.
func (s *Store) loadGroupedCounts(platform, field string, userId int64) (map[string]int, error) {
	const qYear     = `SELECT year,     COUNT(*) FROM media_items WHERE platform = ? AND user_id = ? GROUP BY year`
	const qType     = `SELECT type,     COUNT(*) FROM media_items WHERE platform = ? AND user_id = ? GROUP BY type`
	const qCategory = `SELECT category, COUNT(*) FROM media_items WHERE platform = ? AND user_id = ? GROUP BY category`

	var q string
	switch field {
	case "year":
		q = qYear
	case "type":
		q = qType
	case "category":
		q = qCategory
	default:
		return nil, fmt.Errorf("unsupported group field %q", field)
	}

	rows, err := s.db.Query(q, platform, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int)
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		out[key] = count
	}
	return out, rows.Err()
}

// WritePlatformSnapshotFile writes a JSON summary of the index state to the
// provider snapshot file for offline inspection.
func (s *Store) WritePlatformSnapshotFile(platform string, snapshot any) error {
	root := s.ProviderRoot(platform)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.ProviderSnapshotPath(platform), raw, 0o644)
}
