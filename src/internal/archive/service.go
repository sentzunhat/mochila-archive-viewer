package archive

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"mochila-archive-viewer/src/internal/providers/facebook"
	"mochila-archive-viewer/src/internal/providers/instagram"
	"mochila-archive-viewer/src/internal/providers/snapchat"
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
	}, nil
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
	if err := s.store.SaveSelection(platform, ps.Selected); err != nil {
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

	if err := s.store.SaveSnapshot(platform, ps.Selected, idx, ps.Conversations); err != nil {
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
func (s *Service) GetMedia(platform, year string) ([]snapchat.MediaItem, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Index == nil {
		if ps.Summary != nil {
			return s.store.Media(platform, year)
		}
		return nil, ErrNotIndexed
	}
	if year == "" || year == "all" {
		return ps.Index.Media, nil
	}
	out := make([]snapchat.MediaItem, 0)
	for _, m := range ps.Index.Media {
		if m.Year == year {
			out = append(out, m)
		}
	}
	return out, nil
}

// GetConversations returns the conversation list (no message bodies) for a platform.
func (s *Service) GetConversations(platform string) ([]snapchat.Conversation, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Index == nil {
		if ps.Summary != nil {
			return s.store.Conversations(platform)
		}
		return nil, ErrNotIndexed
	}
	out := make([]snapchat.Conversation, 0, len(ps.Conversations))
	for _, c := range ps.Conversations {
		out = append(out, snapchat.Conversation{
			ID:           c.ID,
			Title:        c.Title,
			MessageCount: c.MessageCount,
			SavedCount:   c.SavedCount,
			MediaCount:   c.MediaCount,
			LastCreated:  c.LastCreated,
		})
	}
	return out, nil
}

func (s *Service) JSONFiles(platform string) ([]snapchat.JsonFileRef, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Index == nil {
		if ps.Summary != nil {
			return s.store.JSONFiles(platform)
		}
		return nil, ErrNotIndexed
	}
	return append([]snapchat.JsonFileRef(nil), ps.JsonFiles...), nil
}

func (s *Service) JSONPreview(platform string, ordinal int) (*JSONPreview, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}

	jsonFiles := ps.JsonFiles
	if ps.Index == nil && ps.Summary != nil {
		jsonFiles, err = s.store.JSONFiles(platform)
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
func (s *Service) GetConversation(platform, id string) (*snapchat.Conversation, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Summary != nil && ps.Index == nil {
		return s.store.Conversation(platform, id)
	}
	for i, c := range ps.Conversations {
		if c.ID == id {
			return &ps.Conversations[i], nil
		}
	}
	return nil, nil
}

// MediaItem looks up a media item by ID within a platform's index.
func (s *Service) MediaItem(platform string, id int) *snapchat.MediaItem {
	ps := s.platforms[platform]
	if ps == nil || id < 0 {
		return nil
	}
	if ps.Index != nil && id < len(ps.Index.Media) {
		m := ps.Index.Media[id]
		return &m
	}
	if len(ps.Media) > 0 && id < len(ps.Media) {
		m := ps.Media[id]
		return &m
	}
	return nil
}

// ZipPath returns the file path of a zip by index within a platform.
func (s *Service) ZipPath(platform string, zipIndex int) string {
	ps := s.platforms[platform]
	if ps == nil || zipIndex < 0 || zipIndex >= len(ps.Selected) {
		return ""
	}
	return ps.Selected[zipIndex].Path
}

func findJsonEntry(idx *snapchat.Index, entry string) *snapchat.JsonFileRef {
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

	snapshot, err := s.store.LoadSnapshot(id)
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
	if err := s.store.SaveProfile(profile); err != nil {
		return nil, err
	}
	return s.store.LoadProfile()
}

func (s *Service) MediaSource(platform string, id int) (string, error) {
	item := s.MediaItem(platform, id)
	if item == nil {
		return "", fmt.Errorf("media item %d not found", id)
	}
	if item.LocalPath != "" {
		if _, err := os.Stat(item.LocalPath); err == nil {
			return s.mediaRenderableSource(item)
		}
	}

	zipPath := s.ZipPath(platform, item.ZipIndex)
	if zipPath == "" {
		return "", fmt.Errorf("zip path for media %d not found", id)
	}
	data, err := snapchat.ReadEntry(zipPath, item.Entry)
	if err != nil {
		return "", err
	}
	target, err := s.mediaCachePath(platform, item.ID, item.Ext)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}
	item.LocalPath = target
	return s.mediaRenderableSource(item)
}

func (s *Service) cacheMediaFiles(platform string, idx *snapchat.Index) error {
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

func (s *Service) mediaRenderableSource(item *snapchat.MediaItem) (string, error) {
	if item == nil || item.LocalPath == "" {
		return "", errors.New("media source is unavailable")
	}
	raw, err := os.ReadFile(item.LocalPath)
	if err != nil {
		return "", err
	}
	return "data:" + mimeFromExt(item.Ext) + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func mimeFromExt(ext string) string {
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "gif":
		return "image/gif"
	case "heic":
		return "image/heic"
	case "mp4":
		return "video/mp4"
	case "mov":
		return "video/quicktime"
	case "webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}
