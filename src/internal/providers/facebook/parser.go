package facebook

import (
	"encoding/json"
	"fmt"
	"mochila-archive-viewer/src/internal/types"
	"time"
)

// Thread structure matches Facebook export format
type Thread struct {
	Participants []struct {
		Name string `json:"name"`
	} `json:"participants"`
	Messages []Message `json:"messages"`
}

// Message matches a single message object in the export
type Message struct {
	SenderName    string      `json:"sender_name"`
	TimestampMs   int64       `json:"timestamp_ms"`
	Content       string      `json:"content"`
	Photos        []MediaItem `json:"photos"`
	Videos        []MediaItem `json:"videos"`
	Gifs          []MediaItem `json:"gifs"`
	AudioFiles    []MediaItem `json:"audio_files"`
	Reactions     []Reaction  `json:"reactions"`
	Sticker       *MediaItem  `json:"sticker"`
}

// MediaItem is a photo/video/gif/audio reference with URI and timestamp
type MediaItem struct {
	URI               string `json:"uri"`
	CreationTimestamp int64  `json:"creation_timestamp"`
}

// Reaction is emoji + actor name
type Reaction struct {
	Reaction string `json:"reaction"`
	Actor    string `json:"actor"`
}

// parseThreadFile unmarshals a single message_N.json file and converts it to types.Conversation.
// mediaByURI maps normalized URIs to their media IDs for linking messages to media items.
func parseThreadFile(
	data []byte,
	threadID string,
	ownerName string,
	mediaByURI map[string]int,
) (*types.Conversation, error) {
	var thread Thread
	if err := json.Unmarshal(data, &thread); err != nil {
		return nil, fmt.Errorf("unmarshal thread JSON: %w", err)
	}

	if len(thread.Messages) == 0 {
		return nil, nil
	}

	// Build participant list and determine if owner is in this thread
	participants := make([]string, len(thread.Participants))
	for i, p := range thread.Participants {
		participants[i] = p.Name
	}

	// Convert messages
	messages := make([]types.ChatMessage, 0, len(thread.Messages))
	for _, msg := range thread.Messages {
		m := types.ChatMessage{
			From:      msg.SenderName,
			Content:   msg.Content,
			Created:   msTimestampToRFC3339(msg.TimestampMs),
			IsSender:  msg.SenderName == ownerName,
			MediaType: mediaTypeForMessage(msg),
		}

		// Attach first media item found (photo > video > gif > audio)
		mediaID := resolveMessageMedia(msg, mediaByURI)
		if mediaID > 0 {
			m.MediaID = &mediaID
			m.LinkedMediaType = determineLinkedMediaType(msg)
		}

		messages = append(messages, m)
	}

	// Use participant names joined as title (first + second if available)
	title := threadID
	if len(participants) > 0 {
		title = participants[0]
		if len(participants) > 1 {
			title += ", " + participants[1]
		}
		if len(participants) > 2 {
			title += fmt.Sprintf(" +%d", len(participants)-2)
		}
	}

	return &types.Conversation{
		ID:           threadID,
		Title:        title,
		MessageCount: len(messages),
		Messages:     messages,
	}, nil
}

// msTimestampToRFC3339 converts milliseconds since epoch to RFC3339 string
func msTimestampToRFC3339(ms int64) string {
	sec := ms / 1000
	nsec := (ms % 1000) * 1_000_000
	t := time.Unix(sec, nsec).UTC()
	return t.Format(time.RFC3339)
}

// mediaTypeForMessage returns the general media type based on what's attached
func mediaTypeForMessage(msg Message) string {
	if msg.Content != "" {
		return "TEXT"
	}
	if len(msg.Photos) > 0 {
		return "PHOTO"
	}
	if len(msg.Videos) > 0 {
		return "VIDEO"
	}
	if len(msg.Gifs) > 0 {
		return "GIF"
	}
	if len(msg.AudioFiles) > 0 {
		return "AUDIO"
	}
	if msg.Sticker != nil {
		return "STICKER"
	}
	return "TEXT"
}

// resolveMessageMedia finds the first media item's ID from the mediaByURI map
func resolveMessageMedia(msg Message, mediaByURI map[string]int) int {
	// Try in order: photos, videos, gifs, audio
	if len(msg.Photos) > 0 && msg.Photos[0].URI != "" {
		return mediaByURI[normalizeURI(msg.Photos[0].URI)]
	}
	if len(msg.Videos) > 0 && msg.Videos[0].URI != "" {
		return mediaByURI[normalizeURI(msg.Videos[0].URI)]
	}
	if len(msg.Gifs) > 0 && msg.Gifs[0].URI != "" {
		return mediaByURI[normalizeURI(msg.Gifs[0].URI)]
	}
	if len(msg.AudioFiles) > 0 && msg.AudioFiles[0].URI != "" {
		return mediaByURI[normalizeURI(msg.AudioFiles[0].URI)]
	}
	return 0
}

// determineLinkedMediaType returns "video" or "image" based on attached media
func determineLinkedMediaType(msg Message) string {
	if len(msg.Videos) > 0 {
		return "video"
	}
	if len(msg.Gifs) > 0 {
		return "image"
	}
	if len(msg.AudioFiles) > 0 {
		return "audio"
	}
	return "image"
}

// normalizeURI ensures consistent lookup in mediaByURI map
func normalizeURI(uri string) string {
	return uri
}
