// app_settings.go — frontend preferences: page size, profile id, update check flag.
package appshell

// AppSettings holds frontend preferences persisted per-user.
type AppSettings struct {
	Pagesize           int    `json:"pageSize"`
	Homedir            string `json:"homedir"`
	ProfileID          int64  `json:"profileId"`
	LoggedIn           bool   `json:"loggedIn"`
	UpdateCheckEnabled bool   `json:"updateCheckEnabled"`
}

var appSettings = &AppSettings{Pagesize: 180, ProfileID: 1, LoggedIn: false, UpdateCheckEnabled: true}

// GetAppSettings returns the current application settings.
func (a *App) GetAppSettings() (*AppSettings, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return appSettings, nil
}

// SaveAppSettings persists application settings.
func (a *App) SaveAppSettings(settings AppSettings) (*AppSettings, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	appSettings.Pagesize = settings.Pagesize
	appSettings.ProfileID = settings.ProfileID
	appSettings.LoggedIn = settings.LoggedIn
	// UpdateCheckEnabled deliberately not synced from settings here: no UI
	// toggle sends it yet (see 013 slice 3 plan), and the frontend's only
	// current caller sends {pagesize, loggedin} without this field — since
	// Go decodes the absent key as the zero value, blindly copying it would
	// silently flip the check off on every unrelated settings save (e.g.
	// dragging the page-size slider). Wire this once an actual toggle exists.
	if appSettings.Pagesize < 30 {
		appSettings.Pagesize = 30
	} else if appSettings.Pagesize > 500 {
		appSettings.Pagesize = 500
	}
	return appSettings, nil
}
