package appshell

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mochila-archive-viewer/src/internal/archive"
	"mochila-archive-viewer/src/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	service *archive.Service
	initErr error
	version string
}

// SetVersion stores the build version injected from main via -ldflags.
func (a *App) SetVersion(v string) {
	a.version = v
}

// AppVersion returns the build version to the frontend.
func (a *App) AppVersion() string {
	if a.version == "" {
		return "dev"
	}
	return a.version
}

// CheckForUpdate compares the running build against the latest GitHub
// Release. Never errors to the frontend — every failure mode already folds
// into archive.UpdateStatus{Available: false} inside archive.CheckForUpdate.
func (a *App) CheckForUpdate() archive.UpdateStatus {
	return archive.CheckForUpdate(a.AppVersion())
}

type FrontendState struct {
	Name      string                 `json:"name"`
	Tagline   string                 `json:"tagline"`
	Providers []archive.ProviderCard `json:"providers"`
	StorePath string                 `json:"storePath"`
	Profile   archive.Profile        `json:"profile"`
}

type PlatformSnapshot struct {
	Selected      []archive.ArchiveFile   `json:"selected"`
	Summary       *archive.IndexSummary   `json:"summary"`
	Media         []types.MediaItem    `json:"media"`
	JsonFiles     []types.JsonFileRef  `json:"jsonFiles"`
	Conversations []types.Conversation `json:"conversations"`
}

func NewApp() *App {
	service, err := archive.NewService()
	return &App{service: service, initErr: err}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetFrontendState() FrontendState {
	tagline := "Local-first archive viewer for exported social data."
	storePath := ""
	if a.initErr != nil {
		tagline = "Local-first archive viewer for exported social data. Storage is currently unavailable."
	} else if a.service != nil {
		storePath = a.service.StorePath()
	}
	providers := []archive.ProviderCard{}
	profile := archive.Profile{}
	if a.service != nil {
		providers = a.service.ProviderCards()
		var userId int64 = 1
		if p, err := a.service.ActiveUser(); err == nil && p != nil && p.ID > 0 {
			profile = *p
			userId = p.ID
		} else if loadedProfile, err := a.service.LoadProfile(); err == nil && loadedProfile != nil {
			profile = *loadedProfile
		}
		a.service.SetActiveUser(userId)
	}
	return FrontendState{
		Name:      "Mochila",
		Tagline:   tagline,
		Providers: providers,
		StorePath: storePath,
		Profile:   profile,
	}
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

// AppSettings holds frontend preferences persisted per-user.
type AppSettings struct {
	Pagesize           int    `json:"pageSize"`
	Homedir            string `json:"homedir"`
	ProfileID          int64  `json:"profileId"`
	LoggedIn           bool   `json:"loggedIn"`
	UpdateCheckEnabled bool   `json:"updateCheckEnabled"`
}

var appSettings = &AppSettings{Pagesize: 180, ProfileID: 1, LoggedIn: false, UpdateCheckEnabled: true}

// GetAppSettings returns the current application settings.
func (a *App) GetAppSettings() (*AppSettings, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return appSettings, nil
}

// SaveAppSettings persists application settings.
func (a *App) SaveAppSettings(settings AppSettings) (*AppSettings, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	appSettings.Pagesize = settings.Pagesize
	appSettings.ProfileID = settings.ProfileID
	appSettings.LoggedIn = settings.LoggedIn
	// UpdateCheckEnabled deliberately not synced from settings here: no UI
	// toggle sends it yet (see 013 slice 3 plan), and the frontend's only
	// current caller sends {pagesize, loggedin} without this field — since
	// Go decodes the absent key as the zero value, blindly copying it would
	// silently flip the check off on every unrelated settings save (e.g.
	// dragging the page-size slider). Wire this once an actual toggle exists.
	if appSettings.Pagesize < 30 {
		appSettings.Pagesize = 30
	} else if appSettings.Pagesize > 500 {
		appSettings.Pagesize = 500
	}
	return appSettings, nil
}

// GetPlatformStats returns platform statistics for the dashboard.
func (a *App) GetPlatformStats(platform string) (*archive.PlatformStats, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.GetPlatformStats(platform)
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

// GetMediaItem returns a single media item's metadata by id, for opening the
// media modal from a chat message's linked media.
func (a *App) GetMediaItem(platform string, id int) (*types.MediaItem, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.GetMediaItem(platform, id)
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

func (a *App) SaveProfile(username, fullName string) (*archive.Profile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.SaveProfile(archive.Profile{
		Username: strings.TrimSpace(username),
		FullName: strings.TrimSpace(fullName),
		LoggedIn: strings.TrimSpace(username) != "",
	})
}

func (a *App) LogoutProfile() (*archive.Profile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.Logout()
}

// ActiveUserProfile returns the currently active user's profile.
func (a *App) ActiveUserProfile() (*archive.Profile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.ActiveUser()
}

// AvailableUsers returns all known user profiles.
func (a *App) AvailableUsers() ([]archive.Profile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.AvailableUsers()
}

// SelectUser activates a specific user by ID for scoped data access.
func (a *App) SelectUser(id int64) (*archive.Profile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.SelectUser(id)
}

// ServeHTTP handles /media/{platform}/{userId}/{id} requests from the
// webview. userId comes from the URL, not the service's active-session
// state — browser GET responses can be cached by URL, and media_id is only
// unique per (platform, user_id), so correctness has to be self-contained
// in the request rather than depend on whichever profile happens to be
// "active" in the process when the request is handled.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/media/"), "/")
	if len(parts) != 3 {
		http.NotFound(w, r)
		return
	}
	platform := parts[0]
	userId, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data, ext, err := a.service.MediaBytesForUser(platform, userId, id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if data == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", mimeFor(ext))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Write(data)
}

func mimeFor(ext string) string {
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
