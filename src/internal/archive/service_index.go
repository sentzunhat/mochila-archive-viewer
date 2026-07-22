// service_index.go — archive indexing: select zips, trigger indexing, expose summary and zip paths.
package archive

import (
	"path/filepath"

	"mochila-archive-viewer/src/internal/providers/instagram"
	"mochila-archive-viewer/src/internal/providers/snapchat"
	"mochila-archive-viewer/src/internal/types"
)

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

	var idx *types.Index
	var conversations []types.Conversation

	switch platform {
	case "snapchat":
		idx, err = snapchat.IndexZips(paths)
		if err != nil {
			return nil, err
		}
		chatRef := findJsonEntry(idx, "json/chat_history.json")
		if chatRef != nil {
			raw, rerr := snapchat.ReadEntryString(idx.Zips[chatRef.ZipIndex].Path, chatRef.Entry)
			if rerr == nil && raw != "" {
				if convos, perr := snapchat.ParseChatHistory([]byte(raw)); perr == nil {
					conversations = convos
				}
			}
		}
	case "instagram":
		idx, conversations, err = instagram.IndexZips(paths)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrPlatformNotSupported
	}

	if err := s.store.SaveSnapshot(platform, s.activeUserId, ps.Selected, idx, conversations); err != nil {
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
	_ = s.store.WritePlatformSnapshotFile(platform, map[string]any{
		"selected":      ps.Selected,
		"summary":       ps.Summary,
		"mediaCount":    len(idx.Media),
		"jsonFiles":     len(idx.JsonFiles),
		"conversations": len(conversations),
	})

	return ps.Summary, nil
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
