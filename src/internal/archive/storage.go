// Package archive provides the SQLite storage layer and service orchestration
// for indexing and querying social-media export archives.
package archive

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		"archive_files": "user_id INTEGER NOT NULL DEFAULT 1",
		"media_items":   "user_id INTEGER NOT NULL DEFAULT 1",
		"json_files":    "user_id INTEGER NOT NULL DEFAULT 1",
		"conversations": "user_id INTEGER NOT NULL DEFAULT 1",
		"messages":      "user_id INTEGER NOT NULL DEFAULT 1",
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

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
