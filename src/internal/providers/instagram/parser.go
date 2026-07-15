package instagram

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"mochila-archive-viewer/src/internal/types"
)

type instagramThread struct {
	Title        string `json:"title"`
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
	Media []struct {
		Image struct{ Uri string `json:"uri"` } `json:"image"`
		Video struct{ Uri string `json:"uri"` } `json:"video"`
	} `json:"media"`
}

func parseThread(zipPath, entry string) *types.Conversation {
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
	lastTime := int64(0)

	for _, m := range thread.Messages {
		content := strings.TrimSpace(m.Content)
		if content == "" && m.Share.Link == "" {
			continue
		}
		if content == "" {
			content = "Shared a link"
		}

		hasAttachment := len(m.Media) > 0 || m.Share.Link != ""

		msg := types.ChatMessage{
			From:      m.SenderName,
			Content:   content,
			MediaType: "link",
			IsSender:  false,
			IsSaved:   false,
		}
		if hasAttachment {
			msg.MediaType = "media"
		}
		if m.Share.Link != "" {
			msg.MediaIDs = m.Share.Link
		}

		messages = append(messages, msg)
		if m.TimestampMs > lastTime {
			lastTime = m.TimestampMs
		}
	}

	if len(messages) == 0 {
		return nil
	}

	// Convert timestamp to formatted date string
	created := ""
	if lastTime > 0 {
		// Use simple epoch formatting for now - can be refined later
		created = "unknown"
	}

	return &types.Conversation{
		ID:           filepath.Base(entry),
		Title:        thread.Title,
		Participants: participants,
		MessageCount: len(messages),
		MediaCount:   0,
		SavedCount:   0,
		LastCreated:  created,
		Messages:     messages,
	}
}
