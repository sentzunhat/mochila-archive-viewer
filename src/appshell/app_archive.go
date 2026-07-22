// app_archive.go — archive bindings: zip selection, indexing, media, conversations, JSON browser.
package appshell

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"mochila-archive-viewer/src/internal/archive"
	"mochila-archive-viewer/src/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type PlatformSnapshot struct {
	Selected      []archive.ArchiveFile   `json:"selected"`
	Summary       *archive.IndexSummary   `json:"summary"`
	Media         []types.MediaItem       `json:"media"`
	JsonFiles     []types.JsonFileRef     `json:"jsonFiles"`
	Conversations []types.Conversation    `json:"conversations"`
}

func (a *App) GetPlatformSnapshot(platform string) (*PlatformSnapshot, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	selected, err := a.service.SelectedArchives(platform)
	if err != nil {
		return nil, err
	}
	summary, err := a.service.Summary(platform)
	if err != nil {
		return nil, err
	}

	snapshot := &PlatformSnapshot{
		Selected: selected,
		Summary:  summary,
	}
	if summary == nil {
		return snapshot, nil
	}

	media, err := a.service.GetMedia(platform, "all")
	if err != nil && !errors.Is(err, archive.ErrNotIndexed) {
		return nil, err
	}
	conversations, err := a.service.GetConversations(platform)
	if err != nil && !errors.Is(err, archive.ErrNotIndexed) {
		return nil, err
	}
	jsonFiles, err := a.service.JSONFiles(platform)
	if err != nil && !errors.Is(err, archive.ErrNotIndexed) {
		return nil, err
	}
	snapshot.Media = media
	snapshot.JsonFiles = jsonFiles
	snapshot.Conversations = conversations
	return snapshot, nil
}

// SelectArchiveZips opens a file picker and assigns the chosen zips to a platform.
func (a *App) SelectArchiveZips(platform string) ([]archive.ArchiveFile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	home, _ := os.UserHomeDir()
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Choose one or more export zip files",
		DefaultDirectory: filepath.Join(home, "Downloads"),
		Filters:          []runtime.FileFilter{{DisplayName: "Zip Archives", Pattern: "*.zip"}},
	})
	if err != nil {
		return nil, err
	}
	return a.service.SetSelectedArchives(platform, paths)
}

// SetArchiveZipsDirect assigns zip file paths to a platform without opening a dialog.
// Useful for testing in Wails dev mode where native dialogs may hang.
func (a *App) SetArchiveZipsDirect(platform string, pathsStr string) ([]archive.ArchiveFile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	paths := strings.Split(pathsStr, ",")
	for i, p := range paths {
		paths[i] = strings.TrimSpace(p)
	}
	return a.service.SetSelectedArchives(platform, paths)
}

// SelectedArchives returns the selected zips for a platform.
func (a *App) SelectedArchives(platform string) ([]archive.ArchiveFile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.SelectedArchives(platform)
}

// IndexArchives indexes the selected zips for a platform.
func (a *App) IndexArchives(platform string) (*archive.IndexSummary, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.IndexArchives(platform)
}

// GetPlatformStats returns platform statistics for the dashboard.
func (a *App) GetPlatformStats(platform string) (*archive.PlatformStats, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.GetPlatformStats(platform)
}

// GetMedia returns media items for a platform, filtered by year ("" or "all" = all years).
func (a *App) GetMedia(platform, year string) ([]types.MediaItem, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.GetMedia(platform, year)
}

// GetMediaPaginated returns a page of media items from the store, narrowed
// by year/category/type/search.
func (a *App) GetMediaPaginated(platform string, filter archive.MediaFilter, offset, limit int64) ([]types.MediaItem, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.GetMediaPaginated(platform, filter, offset, limit)
}

// GetMediaCount returns the total count of media items for a platform matching filter.
func (a *App) GetMediaCount(platform string, filter archive.MediaFilter) (int64, error) {
	if a.initErr != nil {
		return 0, a.initErr
	}
	return a.service.GetMediaCount(platform, filter)
}

// GetMediaItem returns a single media item's metadata by id, for opening the
// media modal from a chat message's linked media.
func (a *App) GetMediaItem(platform string, id int) (*types.MediaItem, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.GetMediaItem(platform, id)
}

// GetConversations returns the conversation list for a platform.
func (a *App) GetConversations(platform string) ([]types.Conversation, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.GetConversations(platform)
}

// GetConversation returns a full conversation with messages.
func (a *App) GetConversation(platform, id string) (*types.Conversation, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.GetConversation(platform, id)
}

func (a *App) GetJSONFiles(platform string) ([]types.JsonFileRef, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.JSONFiles(platform)
}

func (a *App) GetJSONPreview(platform string, ordinal int) (*archive.JSONPreview, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.JSONPreview(platform, ordinal)
}
