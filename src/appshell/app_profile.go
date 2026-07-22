// app_profile.go — user profile bindings: login, logout, list accounts.
package appshell

import (
	"strings"

	"mochila-archive-viewer/src/internal/archive"
)

func (a *App) SaveProfile(username, fullName string) (*archive.Profile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.SaveProfile(archive.Profile{
		Username: strings.TrimSpace(username),
		FullName: strings.TrimSpace(fullName),
		LoggedIn: strings.TrimSpace(username) != "",
	})
}

func (a *App) LogoutProfile() (*archive.Profile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.Logout()
}

// ActiveUserProfile returns the currently active user's profile.
func (a *App) ActiveUserProfile() (*archive.Profile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.ActiveUser()
}

// AvailableUsers returns all known user profiles.
func (a *App) AvailableUsers() ([]archive.Profile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.AvailableUsers()
}

// SelectUser activates a specific user by ID for scoped data access.
func (a *App) SelectUser(id int64) (*archive.Profile, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	return a.service.SelectUser(id)
}
