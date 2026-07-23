// Package appshell wires the archive service to the Wails runtime.
// It owns the RPC boundary: every exported method on App is a Wails binding
// that the Svelte frontend calls via window.go.* interop. Handlers here do
// translation, validation, and error shaping only — business logic lives in
// the archive service layer.
package appshell

import (
	"context"
	"sync"

	"mochila-archive-viewer/src/internal/archive"
)

type mediaCache struct {
	mu    sync.RWMutex
	cache map[string][]byte // key: "platform:userId:id"
}

type App struct {
	ctx         context.Context
	service     *archive.Service
	initErr     error
	version     string
	mediaCache  *mediaCache
}

type FrontendState struct {
	Name      string                 `json:"name"`
	Tagline   string                 `json:"tagline"`
	Providers []archive.ProviderCard `json:"providers"`
	StorePath string                 `json:"storePath"`
	Profile   archive.Profile        `json:"profile"`
}

func NewApp() *App {
	service, err := archive.NewService()
	return &App{
		service:    service,
		initErr:    err,
		mediaCache: &mediaCache{cache: make(map[string][]byte)},
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetFrontendState() FrontendState {
	tagline := "Local-first archive viewer for exported social data."
	storePath := ""
	if a.initErr != nil {
		tagline = "Local-first archive viewer for exported social data. Storage is currently unavailable."
	} else if a.service != nil {
		storePath = a.service.StorePath()
	}
	providers := []archive.ProviderCard{}
	profile := archive.Profile{}
	if a.service != nil {
		providers = a.service.ProviderCards()
		var userId int64 = 1
		if p, err := a.service.ActiveUser(); err == nil && p != nil && p.ID > 0 {
			profile = *p
			userId = p.ID
		} else if loadedProfile, err := a.service.LoadProfile(); err == nil && loadedProfile != nil {
			profile = *loadedProfile
		}
		a.service.SetActiveUser(userId)
	}
	return FrontendState{
		Name:      "Mochila",
		Tagline:   tagline,
		Providers: providers,
		StorePath: storePath,
		Profile:   profile,
	}
}
