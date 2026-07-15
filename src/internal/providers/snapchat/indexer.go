package snapchat

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
	imagePattern = regexp.MustCompile(`(?i)\.(jpe?g|png|webp|gif)$`)
	videoPattern = regexp.MustCompile(`(?i)\.(mp4|mov|webm)$`)
	datePattern  = regexp.MustCompile(`(?:^|/)(\d{4}-\d{2}-\d{2})[_/]`)
)

func IndexZips(paths []string) (*types.Index, error) {
	idx := &types.Index{
		Categories: make(map[string]int),
		Years:      make(map[string]int),
		Types:      make(map[string]int),
	}

	for zipI, path := range paths {
		r, err := zip.OpenReader(path)
		if err != nil {
			return nil, err
		}

		size := int64(0)
		if fi, err := os.Stat(path); err == nil {
			size = fi.Size()
		}

		meta := types.ZipMeta{
			ZipIndex: zipI,
			Path:     path,
			Name:     filepath.Base(path),
			Entries:  len(r.File),
			Size:     size,
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
					ZipIndex: zipI,
					Zip:      meta.Name,
					Entry:    entry,
				})
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

	return idx, nil
}

// ReadEntry reads the raw bytes of a zip entry.
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

// ReadEntryString reads a zip entry as UTF-8.
func ReadEntryString(zipPath, entry string) (string, error) {
	b, err := ReadEntry(zipPath, entry)
	return string(b), err
}

func extractDate(entry string) string {
	m := datePattern.FindStringSubmatch(entry)
	if m != nil {
		return m[1]
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
