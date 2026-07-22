// service_json.go — JSON file browser: list zip entries and render structured previews.
package archive

import (
	"encoding/json"
	"fmt"
	"sort"

	"mochila-archive-viewer/src/internal/providers/snapchat"
	"mochila-archive-viewer/src/internal/types"
)

type JSONPreview struct {
	Entry       string      `json:"entry"`
	Zip         string      `json:"zip"`
	TopLevel    string      `json:"topLevel"`
	Keys        []string    `json:"keys"`
	ItemCount   int         `json:"itemCount"`
	PrettyJSON  string      `json:"prettyJson"`
	StoragePath string      `json:"storagePath"`
	ChildCounts []JSONChild `json:"childCounts,omitempty"`
	SampleJSON  string      `json:"sampleJson,omitempty"`
}

type JSONChild struct {
	Key     string `json:"key"`
	Type    string `json:"type"`
	Records *int   `json:"records,omitempty"`
}

func (s *Service) JSONFiles(platform string) ([]types.JsonFileRef, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Summary == nil {
		return nil, ErrNotIndexed
	}
	return s.store.JSONFiles(platform, s.activeUserId)
}

func (s *Service) JSONPreview(platform string, ordinal int) (*JSONPreview, error) {
	if _, err := s.platform(platform); err != nil {
		return nil, err
	}

	jsonFiles, err := s.store.JSONFiles(platform, s.activeUserId)
	if err != nil {
		return nil, err
	}
	if ordinal < 0 || ordinal >= len(jsonFiles) {
		return nil, fmt.Errorf("json file %d not found", ordinal)
	}

	item := jsonFiles[ordinal]
	storagePath := ""
	if s.store != nil {
		storagePath = s.store.ProviderSnapshotPath(platform)
	}
	zipPath := s.ZipPath(platform, item.ZipIndex)
	if zipPath == "" {
		return &JSONPreview{
			Entry:       item.Entry,
			Zip:         item.Zip,
			StoragePath: storagePath,
			PrettyJSON:  "Zip source is no longer selected in the current session.",
		}, nil
	}

	raw, err := snapchat.ReadEntryString(zipPath, item.Entry)
	if err != nil {
		return nil, err
	}

	preview := &JSONPreview{
		Entry:       item.Entry,
		Zip:         item.Zip,
		StoragePath: storagePath,
	}

	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		preview.TopLevel = "text"
		preview.PrettyJSON = raw
		return preview, nil
	}

	switch value := decoded.(type) {
	case map[string]any:
		preview.TopLevel = "object"
		preview.Keys = make([]string, 0, len(value))
		for key := range value {
			preview.Keys = append(preview.Keys, key)
		}
		sort.Strings(preview.Keys)
		preview.ItemCount = len(value)
	case []any:
		preview.TopLevel = "array"
		preview.ItemCount = len(value)
		if len(value) > 0 {
			if first, ok := value[0].(map[string]any); ok {
				for key := range first {
					preview.Keys = append(preview.Keys, key)
				}
				sort.Strings(preview.Keys)
			}
		}
	default:
		preview.TopLevel = fmt.Sprintf("%T", value)
	}

	pretty, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return nil, err
	}
	preview.PrettyJSON = string(pretty)

	if val, ok := decoded.(map[string]any); ok {
		for _, key := range preview.Keys {
			childVal := val[key]
			ctype := "unknown"
			var records *int
			switch v := childVal.(type) {
			case []any:
				ctype = "array"
				n := len(v)
				records = &n
			case map[string]any:
				ctype = "object"
			default:
				ctype = fmt.Sprintf("%T", v)
			}
			preview.ChildCounts = append(preview.ChildCounts, JSONChild{
				Key:     key,
				Type:    ctype,
				Records: records,
			})
		}
	}

	switch arr := decoded.(type) {
	case []any:
		if len(arr) < 20 {
			preview.SampleJSON = string(pretty)
		} else {
			sample, err := json.MarshalIndent(arr[:20], "", "  ")
			if err == nil {
				preview.SampleJSON = string(sample) + "\n// ... truncated (" + fmt.Sprintf("%d", len(arr)) + " total)"
			}
		}
	case map[string]any:
		preview.SampleJSON = string(pretty)
	default:
		preview.SampleJSON = raw
	}

	return preview, nil
}
