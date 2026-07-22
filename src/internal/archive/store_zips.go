// store_zips.go — zip file and JSON file reference queries.
package archive

import (
	"database/sql"
	"errors"

	"mochila-archive-viewer/src/internal/types"
)

// ZipPathForUser looks up a single archive zip path by ordinal (= MediaItem.ZipIndex),
// scoped to (platform, user). Used by the HTTP media handler to resolve a media
// item's zip without depending on any in-memory "active session" state — the
// request's userId comes from the URL, not the process-global service state,
// so correctness must be self-contained in the DB lookup.
func (s *Store) ZipPathForUser(platform string, userId int64, zipIndex int) (string, error) {
	var path string
	err := s.db.QueryRow(`
		SELECT path FROM archive_files WHERE platform = ? AND user_id = ? AND ordinal = ?
	`, platform, userId, zipIndex).Scan(&path)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return path, err
}

// SelectedArchives returns the currently selected zip files for a (platform, user) pair.
func (s *Store) SelectedArchives(platform string, userId int64) ([]ArchiveFile, error) {
	return s.loadSelected(platform, userId)
}

func (s *Store) loadSelected(platform string, userId int64) ([]ArchiveFile, error) {
	rows, err := s.db.Query(`
		SELECT path, name
		FROM archive_files
		WHERE platform = ? AND user_id = ?
		ORDER BY ordinal
	`, platform, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ArchiveFile{}
	for rows.Next() {
		var item ArchiveFile
		if err := rows.Scan(&item.Path, &item.Name); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// JSONFiles returns the JSON file references for a (platform, user) pair.
func (s *Store) JSONFiles(platform string, userId int64) ([]types.JsonFileRef, error) {
	return s.loadJSONFiles(platform, userId)
}

func (s *Store) loadJSONFiles(platform string, userId int64) ([]types.JsonFileRef, error) {
	rows, err := s.db.Query(`
		SELECT zip_index, zip, entry
		FROM json_files
		WHERE platform = ? AND user_id = ?
		ORDER BY ordinal
	`, platform, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []types.JsonFileRef{}
	for rows.Next() {
		var item types.JsonFileRef
		if err := rows.Scan(&item.ZipIndex, &item.Zip, &item.Entry); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
