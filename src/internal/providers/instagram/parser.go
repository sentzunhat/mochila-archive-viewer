package instagram

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"mochila-archive-viewer/src/internal/types"
)

// instagramThread is the top-level object in a message_N.json file.
type instagramThread struct {
	Title        string `json:"title"`
	ThreadPath   string `json:"thread_path"` // stable ID, e.g. "inbox/alice_123456789"
	Participants []struct {
		Name string `json:"name"`
	} `json:"participants"`
	Messages []threadMessage `json:"messages"`
}

type threadMessage struct {
	SenderName  string `json:"sender_name"`
	TimestampMs int64  `json:"timestamp_ms"`
	Content     string `json:"content"`
	Share       struct {
		Link string `json:"link"`
	} `json:"share"`
	// Instagram uses a "photos" / "videos" style but the meta export wraps
	// both under "media" with sub-keys "image" and "video".
	Media []struct {
		Image struct{ Uri string `json:"uri"` } `json:"image"`
		Video struct{ Uri string `json:"uri"` } `json:"video"`
	} `json:"media"`
	Photos []struct{ Uri string `json:"uri"` } `json:"photos"`
	Videos []struct{ Uri string `json:"uri"` } `json:"videos"`
}

// mediaURIs returns all media URIs from a message, checking both the "media"
// wrapper and the flat "photos"/"videos" arrays (format varies by export year).
func (m *threadMessage) mediaURIs() []string {
	var uris []string
	for _, med := range m.Media {
		if med.Image.Uri != "" {
			uris = append(uris, med.Image.Uri)
		}
		if med.Video.Uri != "" {
			uris = append(uris, med.Video.Uri)
		}
	}
	for _, p := range m.Photos {
		if p.Uri != "" {
			uris = append(uris, p.Uri)
		}
	}
	for _, v := range m.Videos {
		if v.Uri != "" {
			uris = append(uris, v.Uri)
		}
	}
	return uris
}

// personalInfo is the schema for personal_information.json.
type personalInfo struct {
	ProfileUser []struct {
		StringMapData map[string]struct {
			Value string `json:"value"`
		} `json:"string_map_data"`
	} `json:"profile_user"`
}

// extractOwnerName parses the raw personal_information.json bytes and returns
// the account owner's display name, or "" on any error.
func extractOwnerName(data []byte) string {
	var pi personalInfo
	if err := json.Unmarshal(data, &pi); err != nil || len(pi.ProfileUser) == 0 {
		return ""
	}
	if v, ok := pi.ProfileUser[0].StringMapData["Name"]; ok {
		return strings.TrimSpace(v.Value)
	}
	return ""
}

// parseThread parses a single message_N.json entry into a Conversation.
// ownerName is the account holder's display name (used to set IsSender).
// mediaByToken maps bare filename stems to MediaItem.ID values so that
// message media attachments can be linked to indexed gallery items.
// Returns nil when the thread has no parseable messages.
func parseThread(zipPath, entry, ownerName string, mediaByToken map[string]int) *types.Conversation {
	data, err := ReadEntryString(zipPath, entry)
	if err != nil || data == "" {
		return nil
	}

	var thread instagramThread
	if err := json.Unmarshal([]byte(data), &thread); err != nil {
		return nil
	}

	participants := make([]string, len(thread.Participants))
	for i, p := range thread.Participants {
		participants[i] = p.Name
	}

	var messages []types.ChatMessage
	var lastCreated string
	mediaCount := 0

	// Instagram messages are stored newest-first; reverse to chronological order.
	for i := len(thread.Messages) - 1; i >= 0; i-- {
		m := thread.Messages[i]

		content := strings.TrimSpace(m.Content)
		mediaURIs := m.mediaURIs()
		hasMedia := len(mediaURIs) > 0

		if content == "" && m.Share.Link == "" && !hasMedia {
			continue
		}

		created := ""
		if m.TimestampMs > 0 {
			created = time.UnixMilli(m.TimestampMs).UTC().Format("2006-01-02T15:04:05Z")
			if created > lastCreated {
				lastCreated = created
			}
		}

		mediaType := "text"
		var mediaIDs string

		if hasMedia {
			mediaType = "media"
			mediaCount++
			// Store the first media item's basename token so the DB can resolve
			// it back to a MediaItem.ID via mediaTokenIndex in storage.go.
			token := entryToken(mediaURIs[0])
			if _, ok := mediaByToken[token]; ok {
				mediaIDs = token
			}
		} else if m.Share.Link != "" {
			mediaType = "link"
			mediaIDs = m.Share.Link
		}

		if content == "" {
			if hasMedia {
				content = "[media]"
			} else {
				content = m.Share.Link
			}
		}

		messages = append(messages, types.ChatMessage{
			From:      m.SenderName,
			Content:   content,
			MediaType: mediaType,
			Created:   created,
			IsSender:  ownerName != "" && m.SenderName == ownerName,
			IsSaved:   false,
			MediaIDs:  mediaIDs,
		})
	}

	if len(messages) == 0 {
		return nil
	}

	// Use thread_path as the stable conversation ID (it's unique per thread in
	// the export and survives re-indexing). Fall back to the filename when absent.
	id := thread.ThreadPath
	if id == "" {
		id = filepath.Base(filepath.Dir(entry)) + "/" + filepath.Base(entry)
	}

	return &types.Conversation{
		ID:           id,
		Title:        thread.Title,
		Participants: participants,
		MessageCount: len(messages),
		SavedCount:   0,
		MediaCount:   mediaCount,
		LastCreated:  lastCreated,
		Messages:     messages,
	}
}
