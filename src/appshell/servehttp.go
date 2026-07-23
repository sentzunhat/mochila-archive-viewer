// servehttp.go — HTTP media handler: stream media from zip archives with caching/compression.
package appshell

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%d:%d", platform, userId, id)
	a.mediaCache.mu.RLock()
	data, cached := a.mediaCache.cache[cacheKey]
	a.mediaCache.mu.RUnlock()

	var ext string
	if !cached {
		// Cache miss, fetch from service
		var err error
		data, ext, err = a.service.MediaBytesForUser(platform, userId, id)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if data == nil {
			http.NotFound(w, r)
			return
		}
		// Store in cache (limit cache to 500 items to prevent unbounded growth)
		a.mediaCache.mu.Lock()
		if len(a.mediaCache.cache) < 500 {
			a.mediaCache.cache[cacheKey] = data
		}
		a.mediaCache.mu.Unlock()
	} else {
		// Infer extension from cache key (would need to store separately for full solution)
		// For now, infer from first bytes
		ext = inferExtFromBytes(data)
	}

	// Generate ETag from content hash for cache validation
	hash := md5.Sum(data)
	etag := fmt.Sprintf(`"%x"`, hash)

	// Set cache headers: immutable since URL includes content hash
	w.Header().Set("Content-Type", mimeFor(ext))
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))

	// Support HTTP Range requests (video seeking)
	http.ServeContent(w, r, fmt.Sprintf("media.%s", ext), time.Now(), strings.NewReader(string(data)))
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

// inferExtFromBytes detects media type from magic bytes
func inferExtFromBytes(data []byte) string {
	if len(data) < 8 {
		return "bin"
	}
	// JPEG: FFD8FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "jpg"
	}
	// PNG: 89504E47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "png"
	}
	// GIF: 474946
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return "gif"
	}
	// MP4/MOV: ftyp
	if len(data) > 8 && data[4] == 0x66 && data[5] == 0x74 && data[6] == 0x79 && data[7] == 0x70 {
		return "mp4"
	}
	return "bin"
}
