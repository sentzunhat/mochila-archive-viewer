package archive

import "mochila-archive-viewer/src/internal/types"

// PlatformState holds the indexed data for one provider session.
type PlatformState struct {
	Selected      []ArchiveFile
	Index         *types.Index
	Summary       *IndexSummary
	Media         []types.MediaItem
	JsonFiles     []types.JsonFileRef
	Conversations []types.Conversation
	Loaded        bool
}

func newPlatformState() *PlatformState {
	return &PlatformState{}
}
