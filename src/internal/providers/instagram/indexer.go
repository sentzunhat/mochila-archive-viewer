package instagram

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mochila-archive-viewer/src/internal/types"
)

var mediaPattern = regexp.MustCompile(`(?i)\.(jpe?g|png|webp|gif|heic|mp4|mov|webm)$`)
var datePattern  = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)

type Index struct {
	types.Index
	conversations []*types.Conversation
}

func (idx *Index) Conversations() []*types.Conversation {
	return idx.conversations
}

func IndexZips(paths []string) (*Index, error) {
	idx := &Index{
		Index: types.Index{
			Categories: make(map[string]int),
			Years:      make(map[string]int),
			Types:      make(map[string]int),
		},
	}

	for zipI, path := range paths {
		r, err := zip.OpenReader(path)
		if err != nil {
			return nil, err
		}

		var realSize int64
		if fi, err := os.Stat(path); err == nil {
			realSize = fi.Size()
		}

		meta := types.ZipMeta{ZipIndex: zipI, Path: path, Name: filepath.Base(path), Entries: len(r.File), Size: realSize}

		for _, f := range r.File {
			entry := f.Name
			category := strings.SplitN(entry, "/", 2)[0]
			if category == "" {
				category = "root"
			}
			idx.Categories[category]++

			lentry := strings.ToLower(entry)
			if strings.HasSuffix(lentry, ".json") {
				idx.JsonFiles = append(idx.JsonFiles, types.JsonFileRef{ZipIndex: zipI, Zip: meta.Name, Entry: entry})

				if strings.Contains(entry, "/inbox/") && strings.HasPrefix(filepath.Base(entry), "message_") {
					convs := parseThread(path, entry)
					if convs != nil {
						idx.conversations = append(idx.conversations, convs)
					}
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
				ID: len(idx.Media), ZipIndex: zipI, Zip: meta.Name, Entry: entry,
				Category: category, Date: date, Year: year, Type: mtype, Ext: ext,
			})
		}

		r.Close()
		idx.Zips = append(idx.Zips, meta)
	}

	return idx, nil
}

func extractDate(entry string) string {
	m := datePattern.FindString(entry)
	if len(m) == 10 {
		return m
	}
	return "unknown"
}

func typeOf(entry string) string {
	ext := strings.ToLower(filepath.Ext(entry))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".heic":
		return "image"
	case ".mp4", ".mov", ".webm":
		return "video"
	default:
		return "other"
	}
}

func ReadEntryString(zipPath, entry string) (string, error) {
	data, err := ReadEntry(zipPath, entry)
	return string(data), err
}

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

	return nil, fmt.Errorf("entry %q not found in zip %s", entry, filepath.Base(zipPath))
}
