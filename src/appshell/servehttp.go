// servehttp.go — HTTP media handler: serve raw media bytes from zip archives.
package appshell

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
