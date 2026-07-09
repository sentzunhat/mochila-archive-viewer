package archive

import "mochila-archive-viewer/src/internal/providers/snapchat"

// PlatformState holds the indexed data for one provider session.
type PlatformState struct {
	Selected      []ArchiveFile
	Index         *snapchat.Index
	Summary       *IndexSummary
	Media         []snapchat.MediaItem
	JsonFiles     []snapchat.JsonFileRef
	Conversations []snapchat.Conversation
	Loaded        bool
}

func newPlatformState() *PlatformState {
	return &PlatformState{}
}
