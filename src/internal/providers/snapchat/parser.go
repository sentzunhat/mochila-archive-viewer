package snapchat

import (
	"encoding/json"
	"sort"

	"mochila-archive-viewer/src/internal/types"
)

type RawMessage struct {
	From      string `json:"From"`
	Content   string `json:"Content"`
	MediaType string `json:"Media Type"`
	Created   string `json:"Created"`
	IsSender  bool   `json:"IsSender"`
	IsSaved   bool   `json:"IsSaved"`
	MediaIDs  string `json:"Media IDs"`
}

type Conversation struct {
	ID           string              `json:"id"`
	Title        string              `json:"title"`
	MessageCount int                 `json:"messageCount"`
	SavedCount   int                 `json:"savedCount"`
	MediaCount   int                 `json:"mediaCount"`
	LastCreated  string              `json:"lastCreated"`
	Messages     []types.ChatMessage `json:"messages"`
}

// ParseChatHistory parses the raw bytes of chat_history.json and returns types.Conversation slice.
func ParseChatHistory(raw []byte) ([]types.Conversation, error) {
	var history map[string][]RawMessage
	if err := json.Unmarshal(raw, &history); err != nil {
		return nil, err
	}

	convos := make([]types.Conversation, 0, len(history))
	for id, rawMessages := range history {
		messages := make([]types.ChatMessage, 0, len(rawMessages))
		for _, m := range rawMessages {
			messages = append(messages, types.ChatMessage{
				From:      m.From,
				Content:   m.Content,
				MediaType: m.MediaType,
				Created:   m.Created,
				IsSender:  m.IsSender,
				IsSaved:   m.IsSaved,
				MediaIDs:  m.MediaIDs,
			})
		}

		title := id
		for _, m := range messages {
			if m.Content != "" || m.From != "" {
				title = m.From
				break
			}
		}

		lastCreated := ""
		for _, m := range messages {
			if m.Created > lastCreated {
				lastCreated = m.Created
			}
		}

		savedCount := 0
		mediaCount := 0
		for _, m := range messages {
			if m.IsSaved {
				savedCount++
			}
			if m.MediaType != "" || m.MediaIDs != "" {
				mediaCount++
			}
		}

		convos = append(convos, types.Conversation{
			ID:           id,
			Title:        title,
			MessageCount: len(messages),
			SavedCount:   savedCount,
			MediaCount:   mediaCount,
			LastCreated:  lastCreated,
			Messages:     messages,
		})
	}

	sort.Slice(convos, func(i, j int) bool {
		return convos[i].LastCreated > convos[j].LastCreated
	})

	return convos, nil
}

