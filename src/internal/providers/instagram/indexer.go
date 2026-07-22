// Package instagram indexes and parses Instagram "Download Your Information"
// export ZIP archives into the shared domain types used by the archive service.
package instagram

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mochila-archive-viewer/src/internal/types"
)

var (
	mediaPattern = regexp.MustCompile(`(?i)\.(jpe?g|png|webp|gif|heic|mp4|mov|webm)$`)
	imagePattern = regexp.MustCompile(`(?i)\.(jpe?g|png|webp|gif|heic)$`)
	videoPattern = regexp.MustCompile(`(?i)\.(mp4|mov|webm)$`)
	datePattern  = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
)

// IndexZips walks the provided ZIP files and returns the media index,
// all parsed DM conversations, and any indexing error.
func IndexZips(paths []string) (*types.Index, []types.Conversation, error) {
	idx := &types.Index{
		Categories: make(map[string]int),
		Years:      make(map[string]int),
		Types:      make(map[string]int),
	}

	// threadRefs collects (zipPath, entry) for every message_*.json file so
	// they can be parsed in a second pass after the media index is complete.
	type threadRef struct{ zipPath, entry string }
	var threadRefs []threadRef

	for zipI, path := range paths {
		r, err := zip.OpenReader(path)
		if err != nil {
			return nil, nil, err
		}

		var realSize int64
		if fi, err := os.Stat(path); err == nil {
			realSize = fi.Size()
		}

		meta := types.ZipMeta{
			ZipIndex: zipI,
			Path:     path,
			Name:     filepath.Base(path),
			Entries:  len(r.File),
			Size:     realSize,
		}

		for _, f := range r.File {
			entry := f.Name
			category := strings.SplitN(entry, "/", 2)[0]
			if category == "" {
				category = "root"
			}
			idx.Categories[category]++

			lentry := strings.ToLower(entry)
			if strings.HasSuffix(lentry, ".json") {
				idx.JsonFiles = append(idx.JsonFiles, types.JsonFileRef{
					ZipIndex: zipI, Zip: meta.Name, Entry: entry,
				})
				base := filepath.Base(entry)
				// message_1.json, message_2.json … inside any inbox subfolder
				if strings.Contains(entry, "/inbox/") && strings.HasPrefix(base, "message_") {
					threadRefs = append(threadRefs, threadRef{zipPath: path, entry: entry})
				}
			}

			if !mediaPattern.MatchString(entry) {
				continue
			}

			date := extractDate(entry)
			year := "unknown"
			if date != "unknown" {
				year = date[:4]
			}
			mtype := typeOf(entry)
			ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(entry)), ".")

			idx.Years[year]++
			idx.Types[mtype]++

			idx.Media = append(idx.Media, types.MediaItem{
				ID:       len(idx.Media),
				ZipIndex: zipI,
				Zip:      meta.Name,
				Entry:    entry,
				Category: category,
				Date:     date,
				Year:     year,
				Type:     mtype,
				Ext:      ext,
			})
		}
		r.Close()
		idx.Zips = append(idx.Zips, meta)
	}

	// Build entry-token → MediaItem.ID lookup so thread parsers can resolve
	// message media URIs without re-scanning the media list.
	mediaByToken := make(map[string]int, len(idx.Media))
	for _, m := range idx.Media {
		mediaByToken[entryToken(m.Entry)] = m.ID
	}

	// Read the account owner's display name from the export so parseThread
	// can set IsSender correctly. Gracefully returns "" on any failure.
	ownerName := readOwnerName(paths)

	// Parse all discovered thread files.
	var conversations []types.Conversation
	for _, ref := range threadRefs {
		conv := parseThread(ref.zipPath, ref.entry, ownerName, mediaByToken)
		if conv != nil {
			conversations = append(conversations, *conv)
		}
	}

	return idx, conversations, nil
}

// entryToken extracts the bare filename stem (no extension, no date prefix)
// that is used as the lookup key in the media index. Matches the logic in
// storage.go's mediaEntryToken so DB lookups resolve correctly.
func entryToken(entry string) string {
	base := filepath.Base(entry)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// readOwnerName tries to read the account owner's display name from the
// personal_information.json file that Instagram includes in all exports.
// Returns "" when the file is absent or unparseable (export is a partial
// download, or Meta changed the schema).
func readOwnerName(paths []string) string {
	candidates := []string{
		"your_instagram_activity/personal_information/personal_information.json",
		"personal/personal_information.json",
	}
	for _, path := range paths {
		for _, c := range candidates {
			data, err := ReadEntryString(path, c)
			if err != nil || data == "" {
				continue
			}
			name := extractOwnerName([]byte(data))
			if name != "" {
				return name
			}
		}
	}
	return ""
}

// ReadEntry reads the raw bytes of a named entry from a ZIP file.
func ReadEntry(zipPath, entry string) ([]byte, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == entry {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, nil
}

// ReadEntryString reads a ZIP entry as a UTF-8 string.
func ReadEntryString(zipPath, entry string) (string, error) {
	b, err := ReadEntry(zipPath, entry)
	return string(b), err
}

func extractDate(entry string) string {
	m := datePattern.FindString(entry)
	if len(m) == 10 {
		return m
	}
	return "unknown"
}

func typeOf(entry string) string {
	if imagePattern.MatchString(entry) {
		return "image"
	}
	if videoPattern.MatchString(entry) {
		return "video"
	}
	return "other"
}
