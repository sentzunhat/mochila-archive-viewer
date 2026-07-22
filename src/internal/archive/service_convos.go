// service_convos.go — conversation access: list threads and fetch a single thread with messages.
package archive

import "mochila-archive-viewer/src/internal/types"

// GetConversations returns the conversation list (no message bodies) for a platform.
func (s *Service) GetConversations(platform string) ([]types.Conversation, error) {
	ps, err := s.platform(platform)
	if err != nil {
		return nil, err
	}
	if ps.Summary == nil {
		return nil, ErrNotIndexed
	}
	return s.store.Conversations(platform, s.activeUserId)
}

// GetConversation returns a full conversation with messages for a platform.
func (s *Service) GetConversation(platform, id string) (*types.Conversation, error) {
	if _, err := s.platform(platform); err != nil {
		return nil, err
	}
	return s.store.Conversation(platform, id, s.activeUserId)
}
