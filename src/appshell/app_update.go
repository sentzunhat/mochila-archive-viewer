// app_update.go — version and auto-update bindings.
package appshell

import "mochila-archive-viewer/src/internal/archive"

// SetVersion stores the build version injected from main via -ldflags.
func (a *App) SetVersion(v string) {
	a.version = v
}

// AppVersion returns the build version to the frontend.
func (a *App) AppVersion() string {
	if a.version == "" {
		return "dev"
	}
	return a.version
}

// CheckForUpdate compares the running build against the latest GitHub
// Release. Never errors to the frontend — every failure mode already folds
// into archive.UpdateStatus{Available: false} inside archive.CheckForUpdate.
func (a *App) CheckForUpdate() archive.UpdateStatus {
	return archive.CheckForUpdate(a.AppVersion())
}
