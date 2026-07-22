package archive

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"mochila-archive-viewer/src/internal/providers/facebook"
	"mochila-archive-viewer/src/internal/providers/instagram"
	"mochila-archive-viewer/src/internal/providers/snapchat"
	"mochila-archive-viewer/src/internal/types"
)

var ErrPlatformNotSupported = errors.New("platform not yet supported")
var ErrPlatformUnknown = errors.New("unknown platform")
var ErrNotIndexed = errors.New("archive not indexed — run IndexArchives first")

type ArchiveFile struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type ProviderCard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Supported   bool   `json:"supported"`
}

type IndexSummary struct {
	Platform   string         `json:"platform"`
	MediaCount int            `json:"mediaCount"`
	ZipCount   int            `json:"zipCount"`
	Years      map[string]int `json:"years"`
	Types      map[string]int `json:"types"`
	Categories map[string]int `json:"categories"`
}

type Service struct {
	providers []Provider
	platforms map[string]*PlatformState
	store     *Store
	activeUserId int64
}

type JSONPreview struct {
	Entry       string      `json:"entry"`
	Zip         string      `json:"zip"`
	TopLevel    string      `json:"topLevel"`
	Keys        []string    `json:"keys"`
	ItemCount   int         `json:"itemCount"`
	PrettyJSON  string      `json:"prettyJson"`
	StoragePath string      `json:"storagePath"`
	ChildCounts []JSONChild `json:"childCounts,omitempty"`
	SampleJSON  string      `json:"sampleJson,omitempty"`
}

type JSONChild struct {
	Key     string `json:"key"`
	Type    string `json:"type"`
	Records *int   `json:"records,omitempty"`
}

func NewService() (*Service, error) {
	store, err := OpenStore()
	if err != nil {
		return nil, err
	}

	return &Service{
		providers: []Provider{
			snapchat.Provider{},
			instagram.Provider{},
			facebook.Provider{},
		},
		platforms: make(map[string]*PlatformState),
		store:     store,
		activeUserId: 1,
	}, nil
}

func (s *Service) SetActiveUser(userId int64) {
	if s.activeUserId != userId {
		s.activeUserId = userId
		// Cached platform state belongs to the previous user; force a reload.
		s.platforms = make(map[string]*PlatformState)
	}
}

func (s *Service) SelectUser(userId int64) (*Profile, error) {
	p, err := s.store.GetProfileByID(userId)
	if err != nil || p.ID == 0 {
		return &Profile{}, err
	}
	s.SetActiveUser(userId)
	return p, nil
}

func (s *Service) ProviderCards() []ProviderCard {
	cards := make([]ProviderCard, 0, len(s.providers))
	for _, p := range s.providers {
		cards = append(cards, ProviderCard{
			ID:          p.ID(),
			Name:        p.Name(),
			Status:      p.Status(),
			Description: p.Description(),
			Supported:   p.ID() == "snapchat",
		})
	}
	return cards
}

func (s *Service) platform(id string) (*PlatformState, error) {
	switch id {
	case "snapchat", "instagram", "facebook":
	default:
		return nil, ErrPlatformUnknown
	}
	if id != "snapchat" {
		return nil, ErrPlatformNotSupported
	}
	if s.platforms[id] == nil {
		s.platforms[id] = newPlatformState()
	}
	if !s.platforms[id].Loaded {
		if err := s.restorePlatform(id, s.platforms[id]); err != nil {
			return nil, err
		}
	}
	return s.platforms[id], nil
}

// SetSelectedArchives sets the chosen zip files for a platform.
func (s *Service) SetSelectedArchives(platform string, paths []string) ([]ArchiveFile, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	selected := make([]ArchiveFile, 0, len(paths))
	for _, p := range paths {
		selected = append(selected, ArchiveFile{Path: p, Name: filepath.Base(p)})
	}
	ps.Selected = selected
	ps.Summary = nil
	ps.Media = nil
	ps.Conversations = nil
	if err := s.store.SaveSelection(platform, s.activeUserId, ps.Selected); err != nil {
		return nil, err
	}
	return append([]ArchiveFile(nil), ps.Selected...), nil
}

// SelectedArchives returns the currently selected zips for a platform.
func (s *Service) SelectedArchives(platform string) ([]ArchiveFile, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	return append([]ArchiveFile(nil), ps.Selected...), nil
}

func (s *Service) Summary(platform string) (*IndexSummary, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	return ps.Summary, nil
}

// IndexArchives indexes the selected zips for a platform.
func (s *Service) IndexArchives(platform string) (*IndexSummary, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}

	paths := make([]string, len(ps.Selected))
	for i, a := range ps.Selected {
		paths[i] = a.Path
	}

	idx, err := snapchat.IndexZips(paths)
	if err != nil {
		return nil, err
	}
	ps.Index = idx

	chatRef := findJsonEntry(idx, "json/chat_history.json")
	if chatRef != nil {
		raw, err := snapchat.ReadEntryString(idx.Zips[chatRef.ZipIndex].Path, chatRef.Entry)
		if err == nil && raw != "" {
			if convos, err := snapchat.ParseChatHistory([]byte(raw)); err == nil {
				ps.Conversations = convos
			}
		}
	}

	if err := s.cacheMediaFiles(platform, idx); err != nil {
		return nil, err
	}

	if err := s.store.SaveSnapshot(platform, s.activeUserId, ps.Selected, idx, ps.Conversations); err != nil {
		return nil, err
	}

	ps.Summary = &IndexSummary{
		Platform:   platform,
		MediaCount: len(idx.Media),
		ZipCount:   len(idx.Zips),
		Years:      idx.Years,
		Types:      idx.Types,
		Categories: idx.Categories,
	}
	ps.Media = idx.Media
	ps.JsonFiles = idx.JsonFiles
	_ = s.store.WritePlatformSnapshotFile(platform, map[string]any{
		"selected":      ps.Selected,
		"summary":       ps.Summary,
		"mediaCount":    len(idx.Media),
		"jsonFiles":     len(idx.JsonFiles),
		"conversations": len(ps.Conversations),
	})

	return ps.Summary, nil
}

// GetMedia returns media items for a platform, optionally filtered by year.
func (s *Service) GetMedia(platform, year string) ([]types.MediaItem, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Index == nil {
		if ps.Summary != nil {
			return s.store.Media(platform, year, s.activeUserId)
		}
		return nil, ErrNotIndexed
	}
	if year == "" || year == "all" {
		return ps.Index.Media, nil
	}
	out := make([]types.MediaItem, 0)
	for _, m := range ps.Index.Media {
		if m.Year == year {
			out = append(out, m)
		}
	}
	return out, nil
}

// GetMediaPaginated returns a page of media items from the store, narrowed
// by year/category/type/search.
func (s *Service) GetMediaPaginated(platform string, filter MediaFilter, offset, limit int64) ([]types.MediaItem, error) {
	return s.store.MediaPaginated(platform, filter, s.activeUserId, offset, limit)
}

// GetMediaCount returns the total count of media items for a platform matching filter.
func (s *Service) GetMediaCount(platform string, filter MediaFilter) (int64, error) {
	return s.store.MediaCount(platform, filter, s.activeUserId)
}

// GetPlatformStats returns aggregate statistics for a platform.
func (s *Service) GetPlatformStats(platform string) (*PlatformStats, error) {
	stats, err := s.store.PlatformStats(platform, s.activeUserId)
	if err != nil {
		return nil, err
	}
	var imgCount, vidCount int64
	rows, err := s.store.db.Query(`
		SELECT COUNT(*) FROM media_items WHERE platform=? AND user_id=? AND type='image'
		UNION ALL SELECT COUNT(*) FROM media_items WHERE platform=? AND user_id=? AND type='video'
	`, platform, s.activeUserId, platform, s.activeUserId)
	if err == nil {
		var first bool
		for rows.Next() {
			var c int64
			rows.Scan(&c)
			if !first {
				imgCount = c
				first = true
			} else {
				vidCount = c
			}
		}
		rows.Close()
	}
	stats.ImageCount = imgCount
	stats.VideoCount = vidCount
	return stats, nil
}

// GetConversations returns the conversation list (no message bodies) for a platform.
func (s *Service) GetConversations(platform string) ([]types.Conversation, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Index == nil {
		if ps.Summary != nil {
			return s.store.Conversations(platform, s.activeUserId)
		}
		return nil, ErrNotIndexed
	}
	out := make([]types.Conversation, 0, len(ps.Conversations))
	for _, c := range ps.Conversations {
		out = append(out, c)
	}
	return out, nil
}

func (s *Service) JSONFiles(platform string) ([]types.JsonFileRef, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Index == nil {
		if ps.Summary != nil {
			return s.store.JSONFiles(platform, s.activeUserId)
		}
		return nil, ErrNotIndexed
	}
	return append([]types.JsonFileRef(nil), ps.JsonFiles...), nil
}

func (s *Service) JSONPreview(platform string, ordinal int) (*JSONPreview, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}

	jsonFiles := ps.JsonFiles
	if ps.Index == nil && ps.Summary != nil {
		jsonFiles, err = s.store.JSONFiles(platform, s.activeUserId)
		if err != nil {
			return nil, err
		}
	}
	if ordinal < 0 || ordinal >= len(jsonFiles) {
		return nil, fmt.Errorf("json file %d not found", ordinal)
	}

	item := jsonFiles[ordinal]
	storagePath := ""
	if s.store != nil {
		storagePath = s.store.ProviderSnapshotPath(platform)
	}
	zipPath := s.ZipPath(platform, item.ZipIndex)
	if zipPath == "" {
		return &JSONPreview{
			Entry:       item.Entry,
			Zip:         item.Zip,
			StoragePath: storagePath,
			PrettyJSON:  "Zip source is no longer selected in the current session.",
		}, nil
	}

	raw, err := snapchat.ReadEntryString(zipPath, item.Entry)
	if err != nil {
		return nil, err
	}

	preview := &JSONPreview{
		Entry:       item.Entry,
		Zip:         item.Zip,
		StoragePath: storagePath,
	}

	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		preview.TopLevel = "text"
		preview.PrettyJSON = raw
		return preview, nil
	}

	switch value := decoded.(type) {
	case map[string]any:
		preview.TopLevel = "object"
		preview.Keys = make([]string, 0, len(value))
		for key := range value {
			preview.Keys = append(preview.Keys, key)
		}
		sort.Strings(preview.Keys)
		preview.ItemCount = len(value)
	case []any:
		preview.TopLevel = "array"
		preview.ItemCount = len(value)
		if len(value) > 0 {
			if first, ok := value[0].(map[string]any); ok {
				for key := range first {
					preview.Keys = append(preview.Keys, key)
				}
				sort.Strings(preview.Keys)
			}
		}
	default:
		preview.TopLevel = fmt.Sprintf("%T", value)
	}

	pretty, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return nil, err
	}
	preview.PrettyJSON = string(pretty)

	if val, ok := decoded.(map[string]any); ok {
		for _, key := range preview.Keys {
			childVal := val[key]
			ctype := "unknown"
			var records *int
			switch v := childVal.(type) {
			case []any:
				ctype = "array"
				n := len(v)
				records = &n
			case map[string]any:
				ctype = "object"
			default:
				ctype = fmt.Sprintf("%T", v)
			}
			preview.ChildCounts = append(preview.ChildCounts, JSONChild{
				Key:     key,
				Type:    ctype,
				Records: records,
			})
		}
	}

	// Sample data for structured preview
	switch arr := decoded.(type) {
	case []any:
		if len(arr) < 20 {
			preview.SampleJSON = string(pretty)
		} else {
			sample, err := json.MarshalIndent(arr[:20], "", "  ")
			if err == nil {
				preview.SampleJSON = string(sample) + "\n// ... truncated (" + fmt.Sprintf("%d", len(arr)) + " total)"
			}
		}
	case map[string]any:
		preview.SampleJSON = string(pretty)
	default:
		preview.SampleJSON = raw
	}

	return preview, nil
}

// GetConversation returns a full conversation with messages for a platform.
func (s *Service) GetConversation(platform, id string) (*types.Conversation, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Summary != nil && ps.Index == nil {
		return s.store.Conversation(platform, id, s.activeUserId)
	}
	for i, c := range ps.Conversations {
		if c.ID == id {
			return &ps.Conversations[i], nil
		}
	}
	return nil, nil
}

// ZipPath returns the file path of a zip by index within a platform.
func (s *Service) ZipPath(platform string, zipIndex int) string {
	ps := s.platforms[platform]
	if ps == nil || zipIndex < 0 || zipIndex >= len(ps.Selected) {
		return ""
	}
	return ps.Selected[zipIndex].Path
}

func findJsonEntry(idx *types.Index, entry string) *types.JsonFileRef {
	for i, f := range idx.JsonFiles {
		if f.Entry == entry {
			return &idx.JsonFiles[i]
		}
	}
	return nil
}

func (s *Service) restorePlatform(id string, ps *PlatformState) error {
	ps.Loaded = true
	if s.store == nil {
		return nil
	}

	snapshot, err := s.store.LoadSnapshot(id, s.activeUserId)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return nil
	}

	ps.Selected = snapshot.Selected
	ps.Summary = snapshot.Summary
	ps.Media = snapshot.Media
	ps.JsonFiles = snapshot.JsonFiles
	ps.Conversations = snapshot.Conversations
	return nil
}

func (s *Service) StorePath() string {
	if s.store == nil {
		return ""
	}
	return s.store.Path()
}

func (s *Service) LoadProfile() (*Profile, error) {
	if s.store == nil {
		return &Profile{}, nil
	}
	return s.store.LoadProfile()
}

func (s *Service) SaveProfile(profile Profile) (*Profile, error) {
	if s.store == nil {
		return nil, errors.New("archive store is unavailable")
	}
	id, err := s.store.SaveProfile(profile)
	if err != nil {
		return nil, err
	}
	if profile.LoggedIn {
		s.SetActiveUser(id)
	}
	return s.store.GetProfileByID(id)
}

func (s *Service) Logout() (*Profile, error) {
	if s.store == nil {
		return nil, errors.New("archive store is unavailable")
	}
	if err := s.store.Logout(); err != nil {
		return nil, err
	}
	// Drop per-user cached platform state so the next login reloads cleanly.
	s.platforms = make(map[string]*PlatformState)
	return s.store.ActiveUser()
}

func (s *Service) ActiveUser() (*Profile, error) {
    if s.store == nil {
        return &Profile{}, nil
    }
    return s.store.ActiveUser()
}

func (s *Service) AvailableUsers() ([]UserEntry, error) {
    if s.store == nil {
        return []UserEntry{}, nil
    }
    return s.store.AvailableUsers()
}

// GetMediaItem looks up a single media item by id directly from the store,
// scoped to the active user. Unlike MediaItem, this does not depend on the
// in-memory platform cache being populated with a matching-index slice —
// used to resolve media linked from a chat message, which may reference an
// id outside the currently loaded gallery page.
func (s *Service) GetMediaItem(platform string, id int) (*types.MediaItem, error) {
	return s.store.MediaItemByID(platform, id, s.activeUserId)
}

// MediaBytesForUser reads a media item's raw bytes directly from the store
// and its source zip, scoped to an explicit userId rather than the
// service's process-global activeUserId. Used by the HTTP media handler:
// browsers cache GET responses by URL, so correctness for a specific user
// has to come from the request itself (the URL's userId segment), not from
// whichever profile happens to be "active" in the service at request time
// — those can be different mid a profile switch, and unlike RPC calls,
// an HTTP response can outlive the request in the browser's cache.
func (s *Service) MediaBytesForUser(platform string, userId int64, id int) ([]byte, string, error) {
	item, err := s.store.MediaItemByID(platform, id, userId)
	if err != nil {
		return nil, "", err
	}
	if item == nil {
		return nil, "", fmt.Errorf("media item %d not found", id)
	}
	zipPath, err := s.store.ZipPathForUser(platform, userId, item.ZipIndex)
	if err != nil {
		return nil, "", err
	}
	if zipPath == "" {
		return nil, "", fmt.Errorf("zip path for media %d not found", id)
	}
	data, err := snapchat.ReadEntry(zipPath, item.Entry)
	if err != nil {
		return nil, "", err
	}
	return data, item.Ext, nil
}

func (s *Service) cacheMediaFiles(platform string, idx *types.Index) error {
	for i := range idx.Media {
		target, err := s.mediaCachePath(platform, idx.Media[i].ID, idx.Media[i].Ext)
		if err != nil {
			return err
		}
		if _, err := os.Stat(target); err == nil {
			idx.Media[i].LocalPath = target
			continue
		}
		data, err := snapchat.ReadEntry(idx.Zips[idx.Media[i].ZipIndex].Path, idx.Media[i].Entry)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		idx.Media[i].LocalPath = target
	}
	return nil
}

func (s *Service) mediaCachePath(platform string, id int, ext string) (string, error) {
	if s.store == nil {
		return "", errors.New("archive store is unavailable")
	}
	root := s.store.ProviderMediaRoot(platform)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(root, fmt.Sprintf("%06d.%s", id, ext)), nil
}

