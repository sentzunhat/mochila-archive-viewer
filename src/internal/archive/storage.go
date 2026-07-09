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

func (s *Store) SaveSnapshot(platform string, selected []ArchiveFile, idx *snapchat.Index, conversations []snapchat.Conversation) error {
	if idx == nil {
		return errors.New("cannot save empty archive index")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO platform_snapshots(platform, media_count, zip_count)
		VALUES (?, ?, ?)
		ON CONFLICT(platform) DO UPDATE SET media_count = excluded.media_count, zip_count = excluded.zip_count
	`, platform, len(idx.Media), len(idx.Zips)); err != nil {
		return err
	}

	for _, table := range []string{"archive_files", "media_items", "json_files", "conversations", "messages"} {
		if _, err := tx.Exec("DELETE FROM "+table+" WHERE platform = ?", platform); err != nil {
			return err
		}
	}

	for i, file := range selected {
		if _, err := tx.Exec(`
			INSERT INTO archive_files(platform, ordinal, path, name)
			VALUES (?, ?, ?, ?)
		`, platform, i, file.Path, file.Name); err != nil {
			return err
		}
	}

	for _, item := range idx.Media {
		if _, err := tx.Exec(`
			INSERT INTO media_items(platform, media_id, zip_index, zip, entry, category, date, year, type, ext, local_path)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, platform, item.ID, item.ZipIndex, item.Zip, item.Entry, item.Category, item.Date, item.Year, item.Type, item.Ext, item.LocalPath); err != nil {
			return err
		}
	}

	for ordinal, item := range idx.JsonFiles {
		if _, err := tx.Exec(`
			INSERT INTO json_files(platform, ordinal, zip_index, zip, entry)
			VALUES (?, ?, ?, ?, ?)
		`, platform, ordinal, item.ZipIndex, item.Zip, item.Entry); err != nil {
			return err
		}
	}

	for _, convo := range conversations {
		if _, err := tx.Exec(`
			INSERT INTO conversations(platform, conversation_id, title, message_count, saved_count, media_count, last_created)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, platform, convo.ID, convo.Title, convo.MessageCount, convo.SavedCount, convo.MediaCount, convo.LastCreated); err != nil {
			return err
		}

		for ordinal, msg := range convo.Messages {
			if _, err := tx.Exec(`
				INSERT INTO messages(platform, conversation_id, ordinal, from_name, content, media_type, created, is_sender, is_saved, media_ids)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, platform, convo.ID, ordinal, msg.From, msg.Content, msg.MediaType, msg.Created, boolToInt(msg.IsSender), boolToInt(msg.IsSaved), msg.MediaIDs); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *Store) SaveSelection(platform string, selected []ArchiveFile) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM archive_files WHERE platform = ?`, platform); err != nil {
		return err
	}

	for i, file := range selected {
		if _, err := tx.Exec(`
			INSERT INTO archive_files(platform, ordinal, path, name)
			VALUES (?, ?, ?, ?)
		`, platform, i, file.Path, file.Name); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) LoadSnapshot(platform string) (*Snapshot, error) {
	selected, err := s.loadSelected(platform)
	if err != nil {
		return nil, err
	}

	summary, err := s.loadSummary(platform)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return &Snapshot{Selected: selected}, nil
	}

	media, err := s.loadMedia(platform, "all")
	if err != nil {
		return nil, err
	}
	jsonFiles, err := s.loadJSONFiles(platform)
	if err != nil {
		return nil, err
	}
	conversations, err := s.loadConversations(platform, true)
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

func (s *Store) SelectedArchives(platform string) ([]ArchiveFile, error) {
	return s.loadSelected(platform)
}

func (s *Store) Summary(platform string) (*IndexSummary, error) {
	return s.loadSummary(platform)
}

func (s *Store) Media(platform, year string) ([]snapchat.MediaItem, error) {
	return s.loadMedia(platform, year)
}

func (s *Store) Conversations(platform string) ([]snapchat.Conversation, error) {
	return s.loadConversations(platform, false)
}

func (s *Store) JSONFiles(platform string) ([]snapchat.JsonFileRef, error) {
	return s.loadJSONFiles(platform)
}

func (s *Store) Conversation(platform, id string) (*snapchat.Conversation, error) {
	convos, err := s.loadConversations(platform, true)
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
			media_count INTEGER NOT NULL,
			zip_count INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS archive_files (
			platform TEXT NOT NULL,
			ordinal INTEGER NOT NULL,
			path TEXT NOT NULL,
			name TEXT NOT NULL,
			PRIMARY KEY(platform, ordinal)
		)`,
		`CREATE TABLE IF NOT EXISTS media_items (
			platform TEXT NOT NULL,
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
			PRIMARY KEY(platform, media_id)
		)`,
		`CREATE TABLE IF NOT EXISTS json_files (
			platform TEXT NOT NULL,
			ordinal INTEGER NOT NULL,
			zip_index INTEGER NOT NULL,
			zip TEXT NOT NULL,
			entry TEXT NOT NULL,
			PRIMARY KEY(platform, ordinal)
		)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			platform TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			title TEXT NOT NULL,
			message_count INTEGER NOT NULL,
			saved_count INTEGER NOT NULL,
			media_count INTEGER NOT NULL,
			last_created TEXT NOT NULL,
			PRIMARY KEY(platform, conversation_id)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			platform TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			ordinal INTEGER NOT NULL,
			from_name TEXT NOT NULL,
			content TEXT NOT NULL,
			media_type TEXT NOT NULL,
			created TEXT NOT NULL,
			is_sender INTEGER NOT NULL,
			is_saved INTEGER NOT NULL,
			media_ids TEXT NOT NULL,
			PRIMARY KEY(platform, conversation_id, ordinal)
		)`,
		`CREATE TABLE IF NOT EXISTS profile (
			profile_id INTEGER PRIMARY KEY CHECK (profile_id = 1),
			username TEXT NOT NULL,
			full_name TEXT NOT NULL,
			logged_in INTEGER NOT NULL
		)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate sqlite store: %w", err)
		}
	}
	if _, err := s.db.Exec(`ALTER TABLE media_items ADD COLUMN local_path TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("migrate sqlite media local path: %w", err)
	}
	return nil
}

func (s *Store) loadSelected(platform string) ([]ArchiveFile, error) {
	rows, err := s.db.Query(`
		SELECT path, name
		FROM archive_files
		WHERE platform = ?
		ORDER BY ordinal
	`, platform)
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

func (s *Store) loadSummary(platform string) (*IndexSummary, error) {
	var mediaCount, zipCount int
	err := s.db.QueryRow(`
		SELECT media_count, zip_count
		FROM platform_snapshots
		WHERE platform = ?
	`, platform).Scan(&mediaCount, &zipCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	years, err := s.loadGroupedCounts(platform, "year")
	if err != nil {
		return nil, err
	}
	types, err := s.loadGroupedCounts(platform, "type")
	if err != nil {
		return nil, err
	}
	categories, err := s.loadGroupedCounts(platform, "category")
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

func (s *Store) loadGroupedCounts(platform, field string) (map[string]int, error) {
	switch field {
	case "year", "type", "category":
	default:
		return nil, fmt.Errorf("unsupported group field %q", field)
	}

	rows, err := s.db.Query(`
		SELECT `+field+`, COUNT(*)
		FROM media_items
		WHERE platform = ?
		GROUP BY `+field, platform)
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

func (s *Store) loadMedia(platform, year string) ([]snapchat.MediaItem, error) {
	query := `
		SELECT media_id, zip_index, zip, entry, category, date, year, type, ext, local_path
		FROM media_items
		WHERE platform = ?
	`
	args := []any{platform}
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

func (s *Store) loadJSONFiles(platform string) ([]snapchat.JsonFileRef, error) {
	rows, err := s.db.Query(`
		SELECT zip_index, zip, entry
		FROM json_files
		WHERE platform = ?
		ORDER BY ordinal
	`, platform)
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

func (s *Store) loadConversations(platform string, includeMessages bool) ([]snapchat.Conversation, error) {
	rows, err := s.db.Query(`
		SELECT conversation_id, title, message_count, saved_count, media_count, last_created
		FROM conversations
		WHERE platform = ?
		ORDER BY last_created DESC
	`, platform)
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
			convo.Messages, err = s.loadMessages(platform, convo.ID)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, convo)
	}
	return out, rows.Err()
}

func (s *Store) loadMessages(platform, conversationID string) ([]snapchat.ChatMessage, error) {
	rows, err := s.db.Query(`
		SELECT from_name, content, media_type, created, is_sender, is_saved, media_ids
		FROM messages
		WHERE platform = ? AND conversation_id = ?
		ORDER BY ordinal
	`, platform, conversationID)
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
		SELECT username, full_name, logged_in
		FROM profile
		WHERE profile_id = 1
	`).Scan(&profile.Username, &profile.FullName, &loggedIn)
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
	_, err := s.db.Exec(`
		INSERT INTO profile(profile_id, username, full_name, logged_in)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(profile_id) DO UPDATE SET
			username = excluded.username,
			full_name = excluded.full_name,
			logged_in = excluded.logged_in
	`, profile.Username, profile.FullName, boolToInt(profile.LoggedIn))
	return err
}

func (s *Store) DebugJSON(platform string) (string, error) {
	snapshot, err := s.LoadSnapshot(platform)
	if err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
