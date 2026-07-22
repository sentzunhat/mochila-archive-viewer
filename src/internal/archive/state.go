package archive

// PlatformState holds the minimal in-process state for one provider.
// Heavy data (media, conversations, JSON files) is always fetched from the
// DB via the Store — this struct only tracks what must survive a DB round-trip:
// the selected zip paths, the index summary, and whether the platform has been
// loaded from the store in this process lifetime.
type PlatformState struct {
	Selected []ArchiveFile
	Summary  *IndexSummary
	Loaded   bool
}

func newPlatformState() *PlatformState {
	return &PlatformState{}
}
