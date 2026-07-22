// service_media.go — media retrieval: paginated gallery, single item, raw bytes for HTTP serving.
package archive

import (
	"fmt"

	"mochila-archive-viewer/src/internal/providers/instagram"
	"mochila-archive-viewer/src/internal/providers/snapchat"
	"mochila-archive-viewer/src/internal/types"
)

// GetMedia returns media items for a platform, optionally filtered by year.
func (s *Service) GetMedia(platform, year string) ([]types.MediaItem, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Summary == nil {
		return nil, ErrNotIndexed
	}
	return s.store.Media(platform, year, s.activeUserId)
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
	return s.store.PlatformStats(platform, s.activeUserId)
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
	var data []byte
	switch platform {
	case "snapchat":
		data, err = snapchat.ReadEntry(zipPath, item.Entry)
	case "instagram":
		data, err = instagram.ReadEntry(zipPath, item.Entry)
	default:
		return nil, "", fmt.Errorf("unsupported platform %q", platform)
	}
	if err != nil {
		return nil, "", err
	}
	return data, item.Ext, nil
}
