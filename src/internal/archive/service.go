// Package archive provides the SQLite storage layer and service orchestration
// for indexing and querying social-media export archives.
package archive

import (
	"errors"

	"mochila-archive-viewer/src/internal/providers/facebook"
	"mochila-archive-viewer/src/internal/providers/instagram"
	"mochila-archive-viewer/src/internal/providers/snapchat"
)

var ErrPlatformNotSupported = errors.New("platform not yet supported")
var ErrPlatformUnknown = errors.New("unknown platform")
var ErrNotIndexed = errors.New("archive not indexed — run IndexArchives first")

type ArchiveFile struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type ProviderCard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Supported   bool   `json:"supported"`
}

type IndexSummary struct {
	Platform   string         `json:"platform"`
	MediaCount int            `json:"mediaCount"`
	ZipCount   int            `json:"zipCount"`
	Years      map[string]int `json:"years"`
	Types      map[string]int `json:"types"`
	Categories map[string]int `json:"categories"`
}

type Service struct {
	providers    []Provider
	platforms    map[string]*PlatformState
	store        *Store
	activeUserId int64
}

func NewService() (*Service, error) {
	store, err := OpenStore()
	if err != nil {
		return nil, err
	}

	return &Service{
		providers: []Provider{
			snapchat.Provider{},
			instagram.Provider{},
			facebook.Provider{},
		},
		platforms:    make(map[string]*PlatformState),
		store:        store,
		activeUserId: 1,
	}, nil
}

func (s *Service) SetActiveUser(userId int64) {
	if s.activeUserId != userId {
		s.activeUserId = userId
		// Cached platform state belongs to the previous user; force a reload.
		s.platforms = make(map[string]*PlatformState)
	}
}

func (s *Service) SelectUser(userId int64) (*Profile, error) {
	p, err := s.store.GetProfileByID(userId)
	if err != nil || p.ID == 0 {
		return &Profile{}, err
	}
	s.SetActiveUser(userId)
	return p, nil
}

func (s *Service) ProviderCards() []ProviderCard {
	cards := make([]ProviderCard, 0, len(s.providers))
	for _, p := range s.providers {
		cards = append(cards, ProviderCard{
			ID:          p.ID(),
			Name:        p.Name(),
			Status:      p.Status(),
			Description: p.Description(),
			Supported:   p.ID() == "snapchat" || p.ID() == "instagram" || p.ID() == "facebook",
		})
	}
	return cards
}

func (s *Service) platform(id string) (*PlatformState, error) {
	switch id {
	case "snapchat", "instagram", "facebook":
	default:
		return nil, ErrPlatformUnknown
	}
	if s.platforms[id] == nil {
		s.platforms[id] = newPlatformState()
	}
	if !s.platforms[id].Loaded {
		if err := s.restorePlatform(id, s.platforms[id]); err != nil {
			return nil, err
		}
	}
	return s.platforms[id], nil
}

func (s *Service) restorePlatform(id string, ps *PlatformState) error {
	ps.Loaded = true
	if s.store == nil {
		return nil
	}

	snapshot, err := s.store.LoadSnapshot(id, s.activeUserId)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return nil
	}

	ps.Selected = snapshot.Selected
	ps.Summary = snapshot.Summary
	return nil
}

func (s *Service) StorePath() string {
	if s.store == nil {
		return ""
	}
	return s.store.Path()
}
