// store_convos.go — conversation and message queries: list, single-thread, media token resolution.
package archive

import (
	"database/sql"
	"errors"
	"path/filepath"
	"regexp"
	"strings"

	"mochila-archive-viewer/src/internal/types"
)

// chatMediaTokenPattern strips the "YYYY-MM-DD_" date prefix used in Snapchat
// export filenames (see snapchat.extractDate).
var chatMediaTokenPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}_(.+)$`)

// mediaEntryToken extracts the opaque blob id from a media_items.entry path so
// it can be matched against a chat message's media_ids token.
// e.g. "chat_media/2019-04-01_b~EiQSF....jpg" → "b~EiQSF..."
func mediaEntryToken(entry string) string {
	base := filepath.Base(entry)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if m := chatMediaTokenPattern.FindStringSubmatch(base); m != nil {
		return m[1]
	}
	return base
}

type mediaRef struct {
	ID   int
	Type string
}

// mediaTokenIndex maps every media item's blob-id token to its (media_id, type)
// for resolving chat message media references. Built once per conversation-load
// so per-message lookups are O(1) map accesses instead of LIKE scans.
func (s *Store) mediaTokenIndex(platform string, userId int64) (map[string]mediaRef, error) {
	rows, err := s.db.Query(`SELECT media_id, entry, type FROM media_items WHERE platform = ? AND user_id = ?`, platform, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idx := make(map[string]mediaRef)
	for rows.Next() {
		var ref mediaRef
		var entry string
		if err := rows.Scan(&ref.ID, &entry, &ref.Type); err != nil {
			return nil, err
		}
		idx[mediaEntryToken(entry)] = ref
	}
	return idx, rows.Err()
}

// Conversations returns the conversation list (without message bodies) for a (platform, user) pair.
func (s *Store) Conversations(platform string, userId int64) ([]types.Conversation, error) {
	return s.loadConversations(platform, false, userId)
}

// Conversation returns a single conversation with its full message list by ID.
// Returns nil, nil when the conversation ID is not found.
func (s *Store) Conversation(platform, id string, userId int64) (*types.Conversation, error) {
	return s.ConversationByID(platform, id, userId)
}

// ConversationByID returns a single conversation with its full message list,
// fetching only the one matching row instead of scanning all conversations.
// Returns nil, nil when not found.
func (s *Store) ConversationByID(platform, id string, userId int64) (*types.Conversation, error) {
	var convo types.Conversation
	err := s.db.QueryRow(`
		SELECT conversation_id, title, message_count, saved_count, media_count, last_created
		FROM conversations
		WHERE platform = ? AND user_id = ? AND conversation_id = ?
	`, platform, userId, id).Scan(&convo.ID, &convo.Title, &convo.MessageCount, &convo.SavedCount, &convo.MediaCount, &convo.LastCreated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	mediaIndex, err := s.mediaTokenIndex(platform, userId)
	if err != nil {
		return nil, err
	}
	convo.Messages, err = s.loadMessages(platform, id, userId, mediaIndex)
	if err != nil {
		return nil, err
	}
	return &convo, nil
}

func (s *Store) loadConversations(platform string, includeMessages bool, userId int64) ([]types.Conversation, error) {
	// Build the media token index once (not per-conversation) so that opening a
	// conversation with hundreds of media messages doesn't scan media_items
	// hundreds of times.
	var mediaIndex map[string]mediaRef
	if includeMessages {
		var err error
		mediaIndex, err = s.mediaTokenIndex(platform, userId)
		if err != nil {
			return nil, err
		}
	}

	rows, err := s.db.Query(`
		SELECT conversation_id, title, message_count, saved_count, media_count, last_created
		FROM conversations
		WHERE platform = ? AND user_id = ?
		ORDER BY last_created DESC
	`, platform, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []types.Conversation{}
	for rows.Next() {
		var convo types.Conversation
		if err := rows.Scan(&convo.ID, &convo.Title, &convo.MessageCount, &convo.SavedCount, &convo.MediaCount, &convo.LastCreated); err != nil {
			return nil, err
		}
		if includeMessages {
			var err error
			convo.Messages, err = s.loadMessages(platform, convo.ID, userId, mediaIndex)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, convo)
	}
	return out, rows.Err()
}

func (s *Store) loadMessages(platform, conversationID string, userId int64, mediaIndex map[string]mediaRef) ([]types.ChatMessage, error) {
	rows, err := s.db.Query(`
		SELECT from_name, content, media_type, created, is_sender, is_saved, media_ids
		FROM messages
		WHERE platform = ? AND user_id = ? AND conversation_id = ?
		ORDER BY ordinal
	`, platform, userId, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []types.ChatMessage{}
	for rows.Next() {
		var msg types.ChatMessage
		var isSender, isSaved int
		if err := rows.Scan(&msg.From, &msg.Content, &msg.MediaType, &msg.Created, &isSender, &isSaved, &msg.MediaIDs); err != nil {
			return nil, err
		}
		msg.IsSender = isSender == 1
		msg.IsSaved = isSaved == 1
		if msg.MediaIDs != "" {
			// media_ids is " | "-delimited; only the first token has been
			// confirmed to match a media_items filename (see 017).
			token := strings.TrimSpace(strings.SplitN(msg.MediaIDs, "|", 2)[0])
			if ref, ok := mediaIndex[token]; ok {
				mediaID := ref.ID
				msg.MediaID = &mediaID
				msg.LinkedMediaType = ref.Type
			}
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}
