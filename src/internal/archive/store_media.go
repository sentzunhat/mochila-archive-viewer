// store_media.go — media item queries: paginated gallery, single-item lookup, filter, stats.
package archive

import (
	"database/sql"
	"errors"
	"strings"

	"mochila-archive-viewer/src/internal/types"
)

// MediaFilter narrows a media_items query. Empty/("all") fields are ignored.
// Search is a case-insensitive substring match against entry, category, and date.
type MediaFilter struct {
	Year     string
	Category string
	Type     string
	Search   string
}

func (f MediaFilter) apply(query string, args []any) (string, []any) {
	if f.Year != "" && f.Year != "all" {
		query += " AND year = ?"
		args = append(args, f.Year)
	}
	if f.Category != "" && f.Category != "all" {
		query += " AND category = ?"
		args = append(args, f.Category)
	}
	if f.Type != "" && f.Type != "all" {
		query += " AND type = ?"
		args = append(args, f.Type)
	}
	if f.Search != "" {
		query += " AND (entry LIKE ? ESCAPE '\\' OR category LIKE ? ESCAPE '\\' OR date LIKE ? ESCAPE '\\')"
		needle := "%" + escapeLike(f.Search) + "%"
		args = append(args, needle, needle, needle)
	}
	return query, args
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// Media returns all media items for a platform, optionally filtered by year.
func (s *Store) Media(platform, year string, userId int64) ([]types.MediaItem, error) {
	return s.loadMedia(platform, year, userId)
}

func (s *Store) loadMedia(platform, year string, userId int64) ([]types.MediaItem, error) {
	query := `
		SELECT media_id, zip_index, zip, entry, category, date, year, type, ext
		FROM media_items
		WHERE platform = ? AND user_id = ?
	`
	args := []any{platform, userId}
	if year != "" && year != "all" {
		query += " AND year = ?"
		args = append(args, year)
	}
	query += " ORDER BY media_id"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []types.MediaItem{}
	for rows.Next() {
		var item types.MediaItem
		if err := rows.Scan(&item.ID, &item.ZipIndex, &item.Zip, &item.Entry, &item.Category, &item.Date, &item.Year, &item.Type, &item.Ext); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// MediaItemByID looks up a single media item by its numeric id, scoped to
// (platform, user). Used to resolve a chat message's linked media for display.
// Returns nil, nil when the item does not exist for that user.
func (s *Store) MediaItemByID(platform string, id int, userId int64) (*types.MediaItem, error) {
	var item types.MediaItem
	err := s.db.QueryRow(`
		SELECT media_id, zip_index, zip, entry, category, date, year, type, ext
		FROM media_items
		WHERE platform = ? AND user_id = ? AND media_id = ?
	`, platform, userId, id).Scan(&item.ID, &item.ZipIndex, &item.Zip, &item.Entry, &item.Category, &item.Date, &item.Year, &item.Type, &item.Ext)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// MediaPaginated returns one page of media items for a (platform, user) pair,
// narrowed by the supplied filter.
func (s *Store) MediaPaginated(platform string, filter MediaFilter, userId, offset, limit int64) ([]types.MediaItem, error) {
	query := `
		SELECT media_id, zip_index, zip, entry, category, date, year, type, ext
		FROM media_items
		WHERE platform = ? AND user_id = ?
	`
	args := []any{platform, userId}
	query, args = filter.apply(query, args)
	query += " ORDER BY media_id LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []types.MediaItem{}
	for rows.Next() {
		var item types.MediaItem
		if err := rows.Scan(&item.ID, &item.ZipIndex, &item.Zip, &item.Entry, &item.Category, &item.Date, &item.Year, &item.Type, &item.Ext); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// MediaCount returns the total count of media items for a (platform, user) pair
// matching the supplied filter.
func (s *Store) MediaCount(platform string, filter MediaFilter, userId int64) (int64, error) {
	query := `SELECT COUNT(*) FROM media_items WHERE platform = ? AND user_id = ?`
	args := []any{platform, userId}
	query, args = filter.apply(query, args)

	var count int64
	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// PlatformStats returns aggregate row counts across all tables for a (platform, user) pair.
// All counts are returned together; any scan failure returns an error rather than
// silently defaulting to zero.
type PlatformStats struct {
	Platform          string `json:"platform"`
	MediaCount        int64  `json:"mediaCount"`
	ZipCount          int64  `json:"zipCount"`
	ConversationCount int64  `json:"conversationCount"`
	JsonFileCount     int64  `json:"jsonFileCount"`
	ImageCount        int64  `json:"imageCount"`
	VideoCount        int64  `json:"videoCount"`
	YearsFound        int    `json:"yearsFound"`
}

func (s *Store) PlatformStats(platform string, userId int64) (*PlatformStats, error) {
	scan := func(dest any, q string, args ...any) error {
		return s.db.QueryRow(q, args...).Scan(dest)
	}

	var mediaCount, zipCount, convCount, jsonCount, imageCount, videoCount int64
	var yearsFound int

	if err := scan(&mediaCount, `SELECT COUNT(*) FROM media_items WHERE platform=? AND user_id=?`, platform, userId); err != nil {
		return nil, err
	}
	if err := scan(&imageCount, `SELECT COUNT(*) FROM media_items WHERE platform=? AND user_id=? AND type='image'`, platform, userId); err != nil {
		return nil, err
	}
	if err := scan(&videoCount, `SELECT COUNT(*) FROM media_items WHERE platform=? AND user_id=? AND type='video'`, platform, userId); err != nil {
		return nil, err
	}
	if err := scan(&zipCount, `SELECT COUNT(*) FROM archive_files WHERE platform=? AND user_id=?`, platform, userId); err != nil {
		return nil, err
	}
	if err := scan(&convCount, `SELECT COUNT(DISTINCT conversation_id) FROM conversations WHERE platform=? AND user_id=?`, platform, userId); err != nil {
		return nil, err
	}
	if err := scan(&jsonCount, `SELECT COUNT(*) FROM json_files WHERE platform=? AND user_id=?`, platform, userId); err != nil {
		return nil, err
	}
	if err := scan(&yearsFound, `SELECT COUNT(DISTINCT year) FROM media_items WHERE platform=? AND user_id=? AND year!='unknown'`, platform, userId); err != nil {
		return nil, err
	}

	return &PlatformStats{
		Platform:          platform,
		MediaCount:        mediaCount,
		ZipCount:          zipCount,
		ConversationCount: convCount,
		JsonFileCount:     jsonCount,
		ImageCount:        imageCount,
		VideoCount:        videoCount,
		YearsFound:        yearsFound,
	}, nil
}
