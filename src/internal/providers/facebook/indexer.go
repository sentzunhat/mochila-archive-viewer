package facebook

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"mochila-archive-viewer/src/internal/types"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// IndexZips walks all provided ZIP files and extracts media + conversations.
// Returns types.Index and Conversations matching the service interface.
func IndexZips(paths []string) (*types.Index, []types.Conversation, error) {
	idx := &types.Index{
		Zips:       make([]types.ZipMeta, 0, len(paths)),
		Media:      make([]types.MediaItem, 0, 1000),
		JsonFiles:  make([]types.JsonFileRef, 0, 50),
		Categories: make(map[string]int),
		Years:      make(map[string]int),
		Types:      make(map[string]int),
	}

	var conversations []types.Conversation
	mediaByURI := make(map[string]int)

	for zipI, zipPath := range paths {
		meta, convos, err := indexZip(zipI, zipPath, idx, mediaByURI)
		if err != nil {
			return nil, nil, fmt.Errorf("index %s: %w", zipPath, err)
		}
		idx.Zips = append(idx.Zips, meta)
		conversations = append(conversations, convos...)
	}

	return idx, conversations, nil
}

// indexZip opens a single ZIP and extracts its contents
func indexZip(zipI int, zipPath string, idx *types.Index, mediaByURI map[string]int) (types.ZipMeta, []types.Conversation, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return types.ZipMeta{}, nil, fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	var realSize int64
	if fi, err := os.Stat(zipPath); err == nil {
		realSize = fi.Size()
	}

	meta := types.ZipMeta{
		ZipIndex: zipI,
		Path:     zipPath,
		Name:     filepath.Base(zipPath),
		Entries:  len(reader.File),
		Size:     realSize,
	}

	var ownerName string
	threads := make(map[string][]string) // threadID -> []message_*.json entries
	var convos []types.Conversation

	// Pass 1: collect metadata, media, and message files
	for _, f := range reader.File {
		entry := f.Name
		lentry := strings.ToLower(entry)

		// Extract owner name from profile
		if f.Name == "personal_information/profile_information/profile_information.json" {
			data, err := readZipFile(f)
			if err == nil {
				var profile struct {
					ProfileV2 struct {
						Name struct {
							FullName string `json:"full_name"`
						} `json:"name"`
					} `json:"profile_v2"`
				}
				if err := json.Unmarshal(data, &profile); err == nil {
					ownerName = profile.ProfileV2.Name.FullName
				}
			}
		}

		// Track JSON files
		if strings.HasSuffix(lentry, ".json") {
			idx.JsonFiles = append(idx.JsonFiles, types.JsonFileRef{
				ZipIndex: zipI,
				Zip:      meta.Name,
				Entry:    entry,
			})
		}

		// Collect media files
		if isMediaFile(entry) {
			id := len(idx.Media)
			ext := strings.ToLower(filepath.Ext(entry))
			if ext != "" {
				ext = ext[1:] // remove leading dot
			}
			category := inferCategory(entry)

			item := types.MediaItem{
				ID:       id,
				ZipIndex: zipI,
				Zip:      meta.Name,
				Entry:    entry,
				Category: category,
				Type:     typeForExt(ext),
				Ext:      ext,
			}
			idx.Media = append(idx.Media, item)
			idx.Types[item.Type]++
			idx.Categories[category]++
			idx.Years["unknown"]++ // Facebook exports don't embed dates in paths
			mediaByURI[normalizeURI(entry)] = id
		}

		// Collect message files
		if isMessageFile(entry) {
			threadID := extractThreadID(entry)
			if threadID != "" {
				threads[threadID] = append(threads[threadID], entry)
			}
		}
	}

	// Pass 2: parse conversations
	for threadID, messageFiles := range threads {
		for _, msgEntry := range messageFiles {
			f, err := findFileInZip(reader, msgEntry)
			if err != nil {
				continue
			}
			data, err := readZipFile(f)
			if err != nil {
				continue
			}
			conv, err := parseThreadFile(data, threadID, ownerName, mediaByURI)
			if err == nil && conv != nil {
				convos = append(convos, *conv)
			}
		}
	}

	return meta, convos, nil
}

// isMediaFile returns true for media in message directories
func isMediaFile(name string) bool {
	name = strings.ToLower(name)
	if !strings.HasPrefix(name, "your_facebook_activity/messages/") {
		return false
	}
	return strings.Contains(name, "/photos/") ||
		strings.Contains(name, "/videos/") ||
		strings.Contains(name, "/gifs/") ||
		strings.Contains(name, "/audio/")
}

// isMessageFile returns true for message_N.json files
func isMessageFile(name string) bool {
	return strings.HasPrefix(name, "your_facebook_activity/messages/inbox/") &&
		regexp.MustCompile(`message_\d+\.json$`).MatchString(name)
}

// extractThreadID extracts the thread folder name
func extractThreadID(entryName string) string {
	parts := strings.Split(entryName, "/")
	if len(parts) < 5 {
		return ""
	}
	if parts[1] != "messages" || parts[2] != "inbox" {
		return ""
	}
	return parts[3]
}

// inferCategory returns category based on path
func inferCategory(entry string) string {
	lower := strings.ToLower(entry)
	if strings.Contains(lower, "/photos/") {
		return "photo"
	}
	if strings.Contains(lower, "/videos/") {
		return "video"
	}
	if strings.Contains(lower, "/gifs/") {
		return "gif"
	}
	if strings.Contains(lower, "/audio/") {
		return "audio"
	}
	return "other"
}

// typeForExt returns media type based on extension
func typeForExt(ext string) string {
	switch strings.ToLower(ext) {
	case "jpg", "jpeg", "png", "gif", "webp", "heic":
		return "image"
	case "mp4", "mov", "webm", "avi":
		return "video"
	default:
		return "other"
	}
}

// readZipFile extracts file contents from a ZIP entry
func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data := make([]byte, 0, f.UncompressedSize64)
	buf := make([]byte, 32768)
	for {
		n, err := rc.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return data, nil
}

// findFileInZip locates a file by name
func findFileInZip(reader *zip.ReadCloser, name string) (*zip.File, error) {
	for _, f := range reader.File {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, fmt.Errorf("file not found: %s", name)
}

// ReadEntry reads raw bytes from a ZIP file
func ReadEntry(zipPath, entry string) ([]byte, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	f, err := findFileInZip(reader, entry)
	if err != nil {
		return nil, err
	}
	return readZipFile(f)
}

// ReadEntryString reads a file from a ZIP as a string
func ReadEntryString(zipPath, entry string) (string, error) {
	data, err := ReadEntry(zipPath, entry)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
