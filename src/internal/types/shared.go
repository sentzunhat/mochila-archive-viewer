// Package types defines the shared domain types passed between the archive
// service layer and the Wails frontend bindings. All types must be JSON-safe
// so that Wails can serialize them for the webview RPC bridge.
package types

type MediaItem struct {
	ID        int    `json:"id"`
	ZipIndex  int    `json:"zipIndex"`
	Zip       string `json:"zip"`
	Entry     string `json:"entry"`
	Category  string `json:"category"`
	Date      string `json:"date"`
	Year      string `json:"year"`
	Type      string `json:"type"`
	Ext       string `json:"ext"`
	LocalPath string `json:"-"`
}

type JsonFileRef struct {
	ZipIndex int    `json:"zipIndex"`
	Zip      string `json:"zip"`
	Entry    string `json:"entry"`
}

type Conversation struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Participants []string      `json:"participants"`
	MessageCount int           `json:"messageCount"`
	SavedCount   int           `json:"savedCount"`
	MediaCount   int           `json:"mediaCount"`
	LastCreated  string        `json:"lastCreated"`
	Messages     []ChatMessage `json:"messages"`
}

type ChatMessage struct {
	From      string `json:"from"`
	Content   string `json:"content"`
	MediaType string `json:"mediaType"`
	Created   string `json:"created"`
	IsSender  bool   `json:"isSender"`
	IsSaved   bool   `json:"isSaved"`
	MediaIDs  string `json:"mediaIds"`
	// MediaID is the resolved media_items.media_id this message's first
	// media_ids token matches, or nil when it couldn't be resolved
	// (deleted/unindexed export, sticker, etc).
	MediaID *int `json:"mediaId,omitempty"`
	// LinkedMediaType is the resolved media item's type ("image"/"video"),
	// set alongside MediaID — lets the UI pick <img> vs <video> without a
	// second lookup per message.
	LinkedMediaType string `json:"linkedMediaType,omitempty"`
}

type Index struct {
	Zips       []ZipMeta      `json:"zips"`
	Media      []MediaItem    `json:"media"`
	JsonFiles  []JsonFileRef  `json:"jsonFiles"`
	Categories map[string]int `json:"categories"`
	Years      map[string]int `json:"years"`
	Types      map[string]int `json:"types"`
}

type ZipMeta struct {
	ZipIndex int    `json:"zipIndex"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	Entries  int    `json:"entries"`
	Size     int64  `json:"size"`
}

type Provider interface {
	ID() string
	Name() string
	Status() string
	Description() string
}
