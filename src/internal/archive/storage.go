// Package archive provides the SQLite storage layer and service orchestration
// for indexing and querying social-media export archives.
package archive

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mochila-archive-viewer/src/internal/types"

	_ "modernc.org/sqlite"
)

const sqliteDriver = "sqlite"

// Snapshot holds the startup state restored from the database for one
// (platform, user) pair. Only Selected and Summary are restored into
// PlatformState; heavy data (media, conversations, JSON files) is fetched
// on demand via Store methods.
type Snapshot struct {
	Selected []ArchiveFile
	Summary  *IndexSummary
}

// Profile represents a local user account (one per Snapchat/IG/FB username).
type Profile struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
	LoggedIn bool   `json:"loggedIn"`
}

// Store wraps the SQLite database and exposes repository-level operations.
// All methods are scoped to (platform, user_id) — media_id is only unique
// within that pair, never globally.
type Store struct {
	db   *sql.DB
	path string
}

// OpenStore opens (and migrates) the SQLite database at ~/.mochila/database.sqlite.
func OpenStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home: %w", err)
	}

	dir := filepath.Join(home, ".mochila")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create mochila data dir: %w", err)
	}

	path := filepath.Join(dir, "database.sqlite")
	db, err := sql.Open(sqliteDriver, path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	store := &Store{db: db, path: path}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Path returns the absolute path of the SQLite database file.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// RootDir returns the directory that contains the database file (~/.mochila).
func (s *Store) RootDir() string {
	if s == nil {
		return ""
	}
	return filepath.Dir(s.path)
}

// ProvidersRoot returns the root directory for all provider-specific artifacts.
func (s *Store) ProvidersRoot() string {
	return filepath.Join(s.RootDir(), "indexed", "providers")
}

// ProviderRoot returns the artifact directory for a single provider.
func (s *Store) ProviderRoot(platform string) string {
	return filepath.Join(s.ProvidersRoot(), platform)
}

// ProviderMediaRoot returns the cached-media directory for a provider.
// This directory is populated by older index runs but is no longer read by the
// HTTP media handler (which reads from zips directly). It is kept for
// backwards compatibility with existing indexed archives.
func (s *Store) ProviderMediaRoot(platform string) string {
	return filepath.Join(s.ProviderRoot(platform), "media")
}

// ProviderSnapshotPath returns the JSON snapshot file path for a provider.
func (s *Store) ProviderSnapshotPath(platform string) string {
	return filepath.Join(s.ProviderRoot(platform), "snapshot.json")
}

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

// SelectedArchives returns the currently selected zip files for a (platform, user) pair.
func (s *Store) SelectedArchives(platform string, userId int64) ([]ArchiveFile, error) {
	return s.loadSelected(platform, userId)
}

// Summary returns the aggregate counts snapshot for a (platform, user) pair.
func (s *Store) Summary(platform string, userId int64) (*IndexSummary, error) {
	return s.loadSummary(platform, userId)
}

// Media returns all media items for a platform, optionally filtered by year.
func (s *Store) Media(platform, year string, userId int64) ([]types.MediaItem, error) {
	return s.loadMedia(platform, year, userId)
}

// Conversations returns the conversation list (without message bodies) for a (platform, user) pair.
func (s *Store) Conversations(platform string, userId int64) ([]types.Conversation, error) {
	return s.loadConversations(platform, false, userId)
}

// JSONFiles returns the JSON file references for a (platform, user) pair.
func (s *Store) JSONFiles(platform string, userId int64) ([]types.JsonFileRef, error) {
	return s.loadJSONFiles(platform, userId)
}

// Conversation returns a single conversation with its full message list by ID.
// Returns nil, nil when the conversation ID is not found.
func (s *Store) Conversation(platform, id string, userId int64) (*types.Conversation, error) {
	return s.ConversationByID(platform, id, userId)
}

// ConversationByID returns a single conversation with its full message list,
// fetching only the one matching row instead of scanning all conversations.
// Returns nil, nil when not found.
func (s *Store) ConversationByID(platform, id string, userId int64) (*types.Conversation, error) {
	var convo types.Conversation
	err := s.db.QueryRow(`
		SELECT conversation_id, title, message_count, saved_count, media_count, last_created
		FROM conversations
		WHERE platform = ? AND user_id = ? AND conversation_id = ?
	`, platform, userId, id).Scan(&convo.ID, &convo.Title, &convo.MessageCount, &convo.SavedCount, &convo.MediaCount, &convo.LastCreated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	mediaIndex, err := s.mediaTokenIndex(platform, userId)
	if err != nil {
		return nil, err
	}
	convo.Messages, err = s.loadMessages(platform, id, userId, mediaIndex)
	if err != nil {
		return nil, err
	}
	return &convo, nil
}

func (s *Store) migrate() error {
	// Phase 1: create tables (idempotent — CREATE TABLE IF NOT EXISTS).
	// platform_snapshots uses a compound PK so each (platform, user) has its
	// own row; a single-column PK would cause the second user's index to
	// silently overwrite the first's counts.
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS platform_snapshots (
			platform TEXT NOT NULL,
			user_id INTEGER NOT NULL DEFAULT 1,
			media_count INTEGER NOT NULL,
			zip_count INTEGER NOT NULL,
			PRIMARY KEY(platform, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS archive_files (
			platform TEXT NOT NULL,
			user_id INTEGER NOT NULL DEFAULT 1,
			ordinal INTEGER NOT NULL,
			path TEXT NOT NULL,
			name TEXT NOT NULL,
			PRIMARY KEY(platform, user_id, ordinal)
		)`,
		`CREATE TABLE IF NOT EXISTS media_items (
			platform TEXT NOT NULL,
			user_id INTEGER NOT NULL DEFAULT 1,
			media_id INTEGER NOT NULL,
			zip_index INTEGER NOT NULL,
			zip TEXT NOT NULL,
			entry TEXT NOT NULL,
			category TEXT NOT NULL,
			date TEXT NOT NULL,
			year TEXT NOT NULL,
			type TEXT NOT NULL,
			ext TEXT NOT NULL,
			PRIMARY KEY(platform, user_id, media_id)
		)`,
		`CREATE TABLE IF NOT EXISTS json_files (
			platform TEXT NOT NULL,
			user_id INTEGER NOT NULL DEFAULT 1,
			ordinal INTEGER NOT NULL,
			zip_index INTEGER NOT NULL,
			zip TEXT NOT NULL,
			entry TEXT NOT NULL,
			PRIMARY KEY(platform, user_id, ordinal)
		)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			platform TEXT NOT NULL,
			user_id INTEGER NOT NULL DEFAULT 1,
			conversation_id TEXT NOT NULL,
			title TEXT NOT NULL,
			message_count INTEGER NOT NULL,
			saved_count INTEGER NOT NULL,
			media_count INTEGER NOT NULL,
			last_created TEXT NOT NULL,
			PRIMARY KEY(platform, user_id, conversation_id)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			platform TEXT NOT NULL,
			user_id INTEGER NOT NULL DEFAULT 1,
			conversation_id TEXT NOT NULL,
			ordinal INTEGER NOT NULL,
			from_name TEXT NOT NULL,
			content TEXT NOT NULL,
			media_type TEXT NOT NULL,
			created TEXT NOT NULL,
			is_sender INTEGER NOT NULL,
			is_saved INTEGER NOT NULL,
			media_ids TEXT NOT NULL,
			PRIMARY KEY(platform, user_id, conversation_id, ordinal)
		)`,
		`CREATE TABLE IF NOT EXISTS profile (
			profile_id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			full_name TEXT NOT NULL DEFAULT '',
			logged_in INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate sqlite store: %w", err)
		}
	}

	// Phase 2: backwards-compatible column additions for existing databases.
	cols := map[string]string{
		"archive_files":  "user_id INTEGER NOT NULL DEFAULT 1",
		"media_items":    "user_id INTEGER NOT NULL DEFAULT 1",
		"json_files":     "user_id INTEGER NOT NULL DEFAULT 1",
		"conversations":  "user_id INTEGER NOT NULL DEFAULT 1",
		"messages":       "user_id INTEGER NOT NULL DEFAULT 1",
	}
	for tbl, col := range cols {
		if _, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tbl, col)); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("migrate table %s: %w", tbl, err)
		}
	}
	// Backwards-compat for media_items: local_path column may exist in older DBs.
	s.db.Exec("ALTER TABLE media_items ADD COLUMN local_path TEXT NOT NULL DEFAULT ''")

	// Phase 3: migrate platform_snapshots to compound PK when the existing table
	// has a single-column PK (created before the multi-user schema).
	var pkColCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('platform_snapshots') WHERE pk > 0`).Scan(&pkColCount); err == nil && pkColCount < 2 {
		if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS platform_snapshots_new (
			platform TEXT NOT NULL,
			user_id INTEGER NOT NULL DEFAULT 1,
			media_count INTEGER NOT NULL,
			zip_count INTEGER NOT NULL,
			PRIMARY KEY(platform, user_id)
		)`); err != nil {
			return fmt.Errorf("migrate platform_snapshots_new create: %w", err)
		}
		if _, err := s.db.Exec(`INSERT OR IGNORE INTO platform_snapshots_new SELECT platform, COALESCE(user_id, 1), media_count, zip_count FROM platform_snapshots`); err != nil {
			return fmt.Errorf("migrate platform_snapshots_new copy: %w", err)
		}
		if _, err := s.db.Exec(`DROP TABLE platform_snapshots`); err != nil {
			return fmt.Errorf("migrate platform_snapshots drop: %w", err)
		}
		if _, err := s.db.Exec(`ALTER TABLE platform_snapshots_new RENAME TO platform_snapshots`); err != nil {
			return fmt.Errorf("migrate platform_snapshots rename: %w", err)
		}
	}

	// Phase 4: profile table recreation for multi-user schema (removes single-user CHECK constraint).
	rows, err := s.db.Query(`PRAGMA table_info(profile)`)
	if err != nil {
		return fmt.Errorf("pragma profile: %w", err)
	}
	count := 0
	for rows.Next() {
		count++
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
	}
	rows.Close()

	if count <= 4 {
		if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS profile_new (
			profile_id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			full_name TEXT NOT NULL DEFAULT '',
			logged_in INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`); err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("migrate profile_new create: %w", err)
		}
		if _, err := s.db.Exec(`INSERT OR IGNORE INTO profile_new (username, full_name, logged_in, created_at, updated_at)
			SELECT username, full_name, logged_in, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP FROM profile WHERE profile_id = 1`); err != nil {
			return fmt.Errorf("migrate profile_new copy: %w", err)
		}
		if _, err := s.db.Exec(`DROP TABLE IF EXISTS profile`); err != nil {
			return fmt.Errorf("migrate profile drop: %w", err)
		}
		if _, err := s.db.Exec(`ALTER TABLE profile_new RENAME TO profile`); err != nil && !strings.Contains(err.Error(), "no such table: profile_new") {
			return fmt.Errorf("migrate profile rename: %w", err)
		}
	}

	// Phase 5: indexes on frequently-filtered columns not already covered by PKs.
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_media_items_platform_user_year     ON media_items(platform, user_id, year)`,
		`CREATE INDEX IF NOT EXISTS idx_media_items_platform_user_type     ON media_items(platform, user_id, type)`,
		`CREATE INDEX IF NOT EXISTS idx_media_items_platform_user_category ON media_items(platform, user_id, category)`,
		`CREATE INDEX IF NOT EXISTS idx_profile_logged_in                  ON profile(logged_in)`,
	}
	for _, idx := range indexes {
		if _, err := s.db.Exec(idx); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

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

func (s *Store) loadConversations(platform string, includeMessages bool, userId int64) ([]types.Conversation, error) {
	// Build the media token index once (not per-conversation) so that opening a
	// conversation with hundreds of media messages doesn't scan media_items
	// hundreds of times.
	var mediaIndex map[string]mediaRef
	if includeMessages {
		var err error
		mediaIndex, err = s.mediaTokenIndex(platform, userId)
		if err != nil {
			return nil, err
		}
	}

	rows, err := s.db.Query(`
		SELECT conversation_id, title, message_count, saved_count, media_count, last_created
		FROM conversations
		WHERE platform = ? AND user_id = ?
		ORDER BY last_created DESC
	`, platform, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []types.Conversation{}
	for rows.Next() {
		var convo types.Conversation
		if err := rows.Scan(&convo.ID, &convo.Title, &convo.MessageCount, &convo.SavedCount, &convo.MediaCount, &convo.LastCreated); err != nil {
			return nil, err
		}
		if includeMessages {
			var err error
			convo.Messages, err = s.loadMessages(platform, convo.ID, userId, mediaIndex)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, convo)
	}
	return out, rows.Err()
}

// chatMediaTokenPattern strips the "YYYY-MM-DD_" date prefix used in Snapchat
// export filenames (see snapchat.extractDate).
var chatMediaTokenPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}_(.+)$`)

// mediaEntryToken extracts the opaque blob id from a media_items.entry path so
// it can be matched against a chat message's media_ids token.
// e.g. "chat_media/2019-04-01_b~EiQSF....jpg" → "b~EiQSF..."
func mediaEntryToken(entry string) string {
	base := filepath.Base(entry)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if m := chatMediaTokenPattern.FindStringSubmatch(base); m != nil {
		return m[1]
	}
	return base
}

type mediaRef struct {
	ID   int
	Type string
}

// mediaTokenIndex maps every media item's blob-id token to its (media_id, type)
// for resolving chat message media references. Built once per conversation-load
// so per-message lookups are O(1) map accesses instead of LIKE scans.
func (s *Store) mediaTokenIndex(platform string, userId int64) (map[string]mediaRef, error) {
	rows, err := s.db.Query(`SELECT media_id, entry, type FROM media_items WHERE platform = ? AND user_id = ?`, platform, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idx := make(map[string]mediaRef)
	for rows.Next() {
		var ref mediaRef
		var entry string
		if err := rows.Scan(&ref.ID, &entry, &ref.Type); err != nil {
			return nil, err
		}
		idx[mediaEntryToken(entry)] = ref
	}
	return idx, rows.Err()
}

func (s *Store) loadMessages(platform, conversationID string, userId int64, mediaIndex map[string]mediaRef) ([]types.ChatMessage, error) {
	rows, err := s.db.Query(`
		SELECT from_name, content, media_type, created, is_sender, is_saved, media_ids
		FROM messages
		WHERE platform = ? AND user_id = ? AND conversation_id = ?
		ORDER BY ordinal
	`, platform, userId, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []types.ChatMessage{}
	for rows.Next() {
		var msg types.ChatMessage
		var isSender, isSaved int
		if err := rows.Scan(&msg.From, &msg.Content, &msg.MediaType, &msg.Created, &isSender, &isSaved, &msg.MediaIDs); err != nil {
			return nil, err
		}
		msg.IsSender = isSender == 1
		msg.IsSaved = isSaved == 1
		if msg.MediaIDs != "" {
			// media_ids is " | "-delimited; only the first token has been
			// confirmed to match a media_items filename (see 017).
			token := strings.TrimSpace(strings.SplitN(msg.MediaIDs, "|", 2)[0])
			if ref, ok := mediaIndex[token]; ok {
				mediaID := ref.ID
				msg.MediaID = &mediaID
				msg.LinkedMediaType = ref.Type
			}
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
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

// LoadProfile returns the first profile in the database (legacy single-user path).
// For multi-user flows, prefer ActiveUser or GetProfileByID.
func (s *Store) LoadProfile() (*Profile, error) {
	var profile Profile
	var loggedIn int
	err := s.db.QueryRow(`
		SELECT profile_id, username, full_name, logged_in
		FROM profile
		ORDER BY profile_id
		LIMIT 1
	`).Scan(&profile.ID, &profile.Username, &profile.FullName, &loggedIn)
	if errors.Is(err, sql.ErrNoRows) {
		return &Profile{}, nil
	}
	if err != nil {
		return nil, err
	}
	profile.LoggedIn = loggedIn == 1
	return &profile, nil
}

// GetProfileByID returns the profile with the given id, or an empty Profile when not found.
func (s *Store) GetProfileByID(id int64) (*Profile, error) {
	var profile Profile
	var loggedIn int
	err := s.db.QueryRow(`
		SELECT profile_id, username, full_name, logged_in
		FROM profile
		WHERE profile_id = ?
		LIMIT 1
	`, id).Scan(&profile.ID, &profile.Username, &profile.FullName, &loggedIn)
	if errors.Is(err, sql.ErrNoRows) {
		return &Profile{}, nil
	}
	if err != nil {
		return nil, err
	}
	profile.LoggedIn = loggedIn == 1
	return &profile, nil
}

// SaveProfile inserts or updates a profile and returns its id.
// Logging a profile in is exclusive: all other profiles are logged out first,
// and both operations run in a single transaction.
func (s *Store) SaveProfile(profile Profile) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if profile.LoggedIn {
		if _, err := tx.Exec("UPDATE profile SET logged_in = 0"); err != nil {
			return 0, err
		}
	}
	var existingID int64
	err = tx.QueryRow("SELECT profile_id FROM profile WHERE username = ?", profile.Username).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		result, err := tx.Exec(`
			INSERT INTO profile (username, full_name, logged_in)
			VALUES (?, ?, ?)
		`, profile.Username, profile.FullName, boolToInt(profile.LoggedIn))
		if err != nil {
			return 0, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}
		return id, tx.Commit()
	} else if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`
		UPDATE profile SET full_name = ?, logged_in = ?, updated_at = CURRENT_TIMESTAMP
		WHERE profile_id = ?
	`, profile.FullName, boolToInt(profile.LoggedIn), existingID); err != nil {
		return 0, err
	}
	return existingID, tx.Commit()
}

// Logout sets all profiles to logged_in = 0.
func (s *Store) Logout() error {
	_, err := s.db.Exec("UPDATE profile SET logged_in = 0")
	return err
}

// ActiveUser returns the currently logged-in profile, or an empty Profile when none.
func (s *Store) ActiveUser() (*Profile, error) {
	var profile Profile
	var loggedIn int
	err := s.db.QueryRow("SELECT profile_id, username, full_name, logged_in FROM profile WHERE logged_in = 1 LIMIT 1").
		Scan(&profile.ID, &profile.Username, &profile.FullName, &loggedIn)
	if err != nil {
		return &Profile{}, nil
	}
	profile.LoggedIn = loggedIn == 1
	return &profile, nil
}

// AvailableUsers returns all known profiles ordered by username.
func (s *Store) AvailableUsers() ([]Profile, error) {
	rows, err := s.db.Query("SELECT profile_id, username, full_name, logged_in FROM profile ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Profile
	for rows.Next() {
		var p Profile
		var loggedIn int
		if err := rows.Scan(&p.ID, &p.Username, &p.FullName, &loggedIn); err != nil {
			return nil, err
		}
		p.LoggedIn = loggedIn == 1
		result = append(result, p)
	}
	return result, rows.Err()
}
