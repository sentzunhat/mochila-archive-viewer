package archive

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mochila-archive-viewer/src/internal/providers/snapchat"

	_ "modernc.org/sqlite"
)

const sqliteDriver = "sqlite"

type Snapshot struct {
	Selected      []ArchiveFile
	Summary       *IndexSummary
	Media         []snapchat.MediaItem
	JsonFiles     []snapchat.JsonFileRef
	Conversations []snapchat.Conversation
}

type Profile struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
	LoggedIn bool   `json:"loggedIn"`
}

type Store struct {
	db   *sql.DB
	path string
}

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

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *Store) RootDir() string {
	if s == nil {
		return ""
	}
	return filepath.Dir(s.path)
}

func (s *Store) ProvidersRoot() string {
	return filepath.Join(s.RootDir(), "indexed", "providers")
}

func (s *Store) ProviderRoot(platform string) string {
	return filepath.Join(s.ProvidersRoot(), platform)
}

func (s *Store) ProviderMediaRoot(platform string) string {
	return filepath.Join(s.ProviderRoot(platform), "media")
}

func (s *Store) ProviderSnapshotPath(platform string) string {
	return filepath.Join(s.ProviderRoot(platform), "snapshot.json")
}

func (s *Store) SaveSnapshot(platform string, userId int64, selected []ArchiveFile, idx *snapchat.Index, conversations []snapchat.Conversation) error {
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
		ON CONFLICT(platform) DO UPDATE SET media_count = excluded.media_count, zip_count = excluded.zip_count
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
			INSERT INTO archive_files(platform, user_id, ordinal, path, name)
			VALUES (?, ?, ?, ?, ?)
		`, platform, userId, i, file.Path, file.Name); err != nil {
			return err
		}
	}

	for _, item := range idx.Media {
		if _, err := tx.Exec(`
			INSERT INTO media_items(platform, user_id, media_id, zip_index, zip, entry, category, date, year, type, ext, local_path)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, platform, userId, item.ID, item.ZipIndex, item.Zip, item.Entry, item.Category, item.Date, item.Year, item.Type, item.Ext, item.LocalPath); err != nil {
			return err
		}
	}

	for ordinal, item := range idx.JsonFiles {
		if _, err := tx.Exec(`
			INSERT INTO json_files(platform, user_id, ordinal, zip_index, zip, entry)
			VALUES (?, ?, ?, ?, ?, ?)
		`, platform, userId, ordinal, item.ZipIndex, item.Zip, item.Entry); err != nil {
			return err
		}
	}

	for _, convo := range conversations {
		if _, err := tx.Exec(`
			INSERT INTO conversations(platform, user_id, conversation_id, title, message_count, saved_count, media_count, last_created)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, platform, userId, convo.ID, convo.Title, convo.MessageCount, convo.SavedCount, convo.MediaCount, convo.LastCreated); err != nil {
			return err
		}

		for ordinal, msg := range convo.Messages {
			if _, err := tx.Exec(`
				INSERT INTO messages(platform, user_id, conversation_id, ordinal, from_name, content, media_type, created, is_sender, is_saved, media_ids)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, platform, userId, convo.ID, ordinal, msg.From, msg.Content, msg.MediaType, msg.Created, boolToInt(msg.IsSender), boolToInt(msg.IsSaved), msg.MediaIDs); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *Store) SaveSelection(platform string, userId int64, selected []ArchiveFile) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM archive_files WHERE platform = ? AND user_id = ?`, platform, userId); err != nil {
		return err
	}

	for i, file := range selected {
		if _, err := tx.Exec(`
			INSERT INTO archive_files(platform, user_id, ordinal, path, name)
			VALUES (?, ?, ?, ?, ?)
		`, platform, userId, i, file.Path, file.Name); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) LoadSnapshot(platform string, userId int64) (*Snapshot, error) {
	selected, err := s.loadSelected(platform, userId)
	if err != nil {
		return nil, err
	}

	summary, err := s.loadSummary(platform, userId)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return &Snapshot{Selected: selected}, nil
	}

	media, err := s.loadMedia(platform, "all", userId)
	if err != nil {
		return nil, err
	}
	jsonFiles, err := s.loadJSONFiles(platform, userId)
	if err != nil {
		return nil, err
	}
	conversations, err := s.loadConversations(platform, true, userId)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		Selected:      selected,
		Summary:       summary,
		Media:         media,
		JsonFiles:     jsonFiles,
		Conversations: conversations,
	}, nil
}

func (s *Store) SelectedArchives(platform string, userId int64) ([]ArchiveFile, error) {
	return s.loadSelected(platform, userId)
}

func (s *Store) Summary(platform string, userId int64) (*IndexSummary, error) {
	return s.loadSummary(platform, userId)
}

func (s *Store) Media(platform, year string, userId int64) ([]snapchat.MediaItem, error) {
	return s.loadMedia(platform, year, userId)
}

func (s *Store) Conversations(platform string, userId int64) ([]snapchat.Conversation, error) {
	return s.loadConversations(platform, false, userId)
}

func (s *Store) JSONFiles(platform string, userId int64) ([]snapchat.JsonFileRef, error) {
	return s.loadJSONFiles(platform, userId)
}

func (s *Store) Conversation(platform, id string, userId int64) (*snapchat.Conversation, error) {
	convos, err := s.loadConversations(platform, true, userId)
	if err != nil {
		return nil, err
	}
	for i := range convos {
		if convos[i].ID == id {
			return &convos[i], nil
		}
	}
	return nil, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS platform_snapshots (
			platform TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL DEFAULT 1,
			media_count INTEGER NOT NULL,
			zip_count INTEGER NOT NULL
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
			local_path TEXT NOT NULL DEFAULT '',
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

	// Backwards-compatible column additions for existing databases
	cols := map[string]string{
		"platform_snapshots": "user_id INTEGER NOT NULL DEFAULT 1",
		"archive_files":      "user_id INTEGER NOT NULL DEFAULT 1",
		"media_items":        "user_id INTEGER NOT NULL DEFAULT 1",
		"json_files":         "user_id INTEGER NOT NULL DEFAULT 1",
		"conversations":      "user_id INTEGER NOT NULL DEFAULT 1",
		"messages":           "user_id INTEGER NOT NULL DEFAULT 1",
	}
	for tbl, col := range cols {
		if _, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tbl, col)); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("migrate table %s: %w", tbl, err)
		}
	}

	// Migrate profile: recreate with multi-user support if it still has CHECK constraint
	// Count columns to detect old single-user schema
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
		// Old schema (profile_id, username, full_name, logged_in) — recreate with new schema
		if _, err := s.db.Exec(`CREATE TABLE profile_new (
			profile_id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			full_name TEXT NOT NULL DEFAULT '',
			logged_in INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("migrate profile new: %w", err)
		}
		s.db.Exec(`INSERT OR IGNORE INTO profile_new (username, full_name, logged_in, created_at, updated_at)
			SELECT username, full_name, logged_in, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP FROM profile WHERE profile_id = 1`)
		s.db.Exec(`DROP TABLE IF EXISTS profile`)
		s.db.Exec(`ALTER TABLE profile_new RENAME TO profile`)
	}

	return nil
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

func (s *Store) loadGroupedCounts(platform, field string, userId int64) (map[string]int, error) {
	switch field {
	case "year", "type", "category":
	default:
		return nil, fmt.Errorf("unsupported group field %q", field)
	}

	rows, err := s.db.Query(`
		SELECT `+field+`, COUNT(*)
		FROM media_items
		WHERE platform = ? AND user_id = ?
		GROUP BY `+field, platform, userId)
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

func (s *Store) loadMedia(platform, year string, userId int64) ([]snapchat.MediaItem, error) {
	query := `
		SELECT media_id, zip_index, zip, entry, category, date, year, type, ext, local_path
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

	out := []snapchat.MediaItem{}
	for rows.Next() {
		var item snapchat.MediaItem
		if err := rows.Scan(&item.ID, &item.ZipIndex, &item.Zip, &item.Entry, &item.Category, &item.Date, &item.Year, &item.Type, &item.Ext, &item.LocalPath); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) loadJSONFiles(platform string, userId int64) ([]snapchat.JsonFileRef, error) {
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

	out := []snapchat.JsonFileRef{}
	for rows.Next() {
		var item snapchat.JsonFileRef
		if err := rows.Scan(&item.ZipIndex, &item.Zip, &item.Entry); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) loadConversations(platform string, includeMessages bool, userId int64) ([]snapchat.Conversation, error) {
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

	out := []snapchat.Conversation{}
	for rows.Next() {
		var convo snapchat.Conversation
		if err := rows.Scan(&convo.ID, &convo.Title, &convo.MessageCount, &convo.SavedCount, &convo.MediaCount, &convo.LastCreated); err != nil {
			return nil, err
		}
		if includeMessages {
			convo.Messages, err = s.loadMessages(platform, convo.ID, userId)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, convo)
	}
	return out, rows.Err()
}

func (s *Store) loadMessages(platform, conversationID string, userId int64) ([]snapchat.ChatMessage, error) {
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

	out := []snapchat.ChatMessage{}
	for rows.Next() {
		var msg snapchat.ChatMessage
		var isSender, isSaved int
		if err := rows.Scan(&msg.From, &msg.Content, &msg.MediaType, &msg.Created, &isSender, &isSaved, &msg.MediaIDs); err != nil {
			return nil, err
		}
		msg.IsSender = isSender == 1
		msg.IsSaved = isSaved == 1
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

func (s *Store) SaveProfile(profile Profile) error {
	// Check if profile already exists by username
	var existingID int64
	err := s.db.QueryRow("SELECT profile_id FROM profile WHERE username = ?", profile.Username).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		// New user - insert
		result, err := s.db.Exec(`
			INSERT INTO profile (username, full_name, logged_in)
			VALUES (?, ?, ?)
		`, profile.Username, profile.FullName, boolToInt(profile.LoggedIn))
		if err != nil {
			return err
		}
		id, _ := result.LastInsertId()
		profile.ID = id
		return nil
	} else if err != nil {
		return err
	}
	// Existing user - update
	_, err = s.db.Exec(`
		UPDATE profile SET full_name = ?, logged_in = ?, updated_at = CURRENT_TIMESTAMP
		WHERE profile_id = ?
	`, profile.FullName, boolToInt(profile.LoggedIn), existingID)
	return err
}


func (s *Store) Logout() error {
	_, err := s.db.Exec("UPDATE profile SET logged_in = 0")
	return err
}

func (s *Store) ActiveUser() (*Profile, error) {
	var profile Profile
	row := s.db.QueryRow("SELECT profile_id, username, full_name, logged_in FROM profile WHERE logged_in = 1 LIMIT 1")
	err := row.Scan(&profile.ID, &profile.Username, &profile.FullName, &profile.LoggedIn)
	if err != nil {
		return &Profile{}, nil
	}
	return &profile, nil
}

func (s *Store) AvailableUsers() ([]UserEntry, error) {
	rows, err := s.db.Query("SELECT profile_id, username, full_name FROM profile ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []UserEntry
	for rows.Next() {
		var u UserEntry
		if err := rows.Scan(&u.ID, &u.Username, &u.FullName); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

type UserEntry struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
}

func (s *Store) DebugJSON(platform string, userId int64) (string, error) {
	snapshot, err := s.LoadSnapshot(platform, userId)
	if err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
