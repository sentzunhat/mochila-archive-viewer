# Update Check Implementation Plan (013, slice 3)

**Created:** 2026-07-21
**Parent Item:** [013](../../work/active/013.md) — Auto-update support for packaged app
**Status:** ready to execute
**Depends on:** slice 2's release pipeline (`.github/workflows/release.yml`) having published at least one tagged GitHub Release — the check has nothing to compare against until then. Safe to build and merge before the first release; it will just report "no update" until a release exists.

---

## Goal

On startup (and via a manual "Check for updates" button in Settings), compare the running build's version against the latest published GitHub Release. Show a small, dismissible notice if newer — never block, never auto-download. Must degrade silently and instantly when offline, since the app is local-first and has to work with no network at all.

---

## Backend

### New file: `src/internal/archive/updatecheck.go` (or a method on `Service` in an existing file — either is fine; a new file keeps `service.go` from growing further)

```go
package archive

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const releasesAPIURL = "https://api.github.com/repos/sentzunhat/mochila-archive-viewer/releases/latest"

type UpdateStatus struct {
	Available bool   `json:"available"`
	Latest    string `json:"latest"`
	URL       string `json:"url"`
}

// CheckForUpdate compares currentVersion against the latest published GitHub
// Release. Never returns an error to the caller — any failure (offline, no
// releases yet, rate limited, malformed response) is treated as "no update
// available" so this can never surface a scary error to a local-first app
// that's expected to work without network access.
func CheckForUpdate(currentVersion string) UpdateStatus {
	none := UpdateStatus{Available: false}
	if currentVersion == "" || currentVersion == "dev" {
		return none // unreleased/dev build — never nag about updating
	}

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest("GET", releasesAPIURL, nil)
	if err != nil {
		return none
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return none // offline, DNS failure, timeout — all land here
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return none // 404 (no releases yet), 403 (rate limited), etc.
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return none
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")
	if latest == "" || !isNewer(latest, current) {
		return none
	}
	return UpdateStatus{Available: true, Latest: release.TagName, URL: release.HTMLURL}
}

// isNewer does a simple dotted-numeric compare (e.g. "0.2.0" > "0.1.0").
// Not a full semver parser (no pre-release/build metadata handling) —
// sufficient for this project's tag scheme (see release.yml: v0.1.0 style).
func isNewer(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var av, bv int
		if i < len(as) {
			av = atoiSafe(as[i])
		}
		if i < len(bs) {
			bv = atoiSafe(bs[i])
		}
		if av != bv {
			return av > bv
		}
	}
	return false
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}
```

Design notes:
- **Package-level function, not a `Service` method** — it needs no store/user state, just the current version string. Keeps it trivially unit-testable (see Testing below) without needing a `*Service` fixture.
- **`isNewer` is deliberately not a semver library dependency.** The project's tags are plain `vMAJOR.MINOR.PATCH` (see `release.yml`); pulling in a semver package for three-segment integer comparison isn't worth the new dependency. If tags ever grow pre-release suffixes (`v0.2.0-beta`), revisit.

### `appshell/app.go` binding

```go
// CheckForUpdate compares the running build against the latest GitHub
// Release. Never errors to the frontend — see archive.CheckForUpdate.
func (a *App) CheckForUpdate() archive.UpdateStatus {
	return archive.CheckForUpdate(a.version)
}
```

Note the return type has no `error` — unlike most bindings in this file (`GetMediaItem`, `GetConversation`, etc. all return `(T, error)`). That's deliberate: every failure mode is already folded into `UpdateStatus{Available: false}` inside `CheckForUpdate`, so there's nothing for the frontend to catch or display as an error. Keep it that way; don't add an error return "for consistency" — it would just be permanently nil.

### Settings persistence (disable-check flag)

`AppSettings` (in `appshell/app.go`) is process-lifetime only today — `var appSettings = &AppSettings{...}` is a package global, never written to disk, reset on every restart. Adding an `UpdateCheckEnabled bool` field to it is the path of least resistance and matches the existing (limited) persistence model:

```go
type AppSettings struct {
	Pagesize           int   `json:"pageSize"`
	Homedir            string
	ProfileID          int64 `json:"profileId"`
	LoggedIn           bool  `json:"loggedIn"`
	UpdateCheckEnabled bool  `json:"updateCheckEnabled"`
}

var appSettings = &AppSettings{Pagesize: 180, ProfileID: 1, LoggedIn: false, UpdateCheckEnabled: true}
```

Update `SaveAppSettings` to persist the new field alongside the existing ones. **Known limitation, inherited, not introduced by this change:** because `AppSettings` isn't written to SQLite or a config file, disabling the check only lasts until the app restarts, at which point it defaults back to enabled. If that's not acceptable, persisting `AppSettings` properly is a separate, larger item — flagging it here rather than scope-creeping it into this one.

---

## Frontend (`src/frontend/src/App.svelte`)

### Import + state

```typescript
// add to the wailsjs import block
CheckForUpdate,
```

```typescript
let updateStatus: { available: boolean; latest: string; url: string } | null = null;
```

### Trigger on startup

In `onMount`, after the existing `AppVersion()` call:

```typescript
try {
  updateStatus = await CheckForUpdate();
} catch (e) {
  // CheckForUpdate never errors per the backend contract, but guard anyway —
  // an update check must never block or break startup.
}
```

### Settings modal — extend the existing "About" section

`App.svelte` already has an "About" `settings-section` (around line 1377) showing Version/Backend/Storage. Add an update row and a manual check button there rather than a new section — it's the natural home:

```svelte
<section class="settings-section">
  <h3>About</h3>
  <dl class="settings-list">
    <div><dt>Version</dt><dd>Mochila {appVersion}</dd></div>
    <div><dt>Backend</dt><dd>Go with Wails v2</dd></div>
    <div><dt>Storage</dt><dd>{appState.storePath || "~/.mochila/database.sqlite"}</dd></div>
  </dl>
  {#if updateStatus?.available}
    <p class="settings-hint">
      <a href={updateStatus.url} target="_blank" rel="noreferrer">
        {updateStatus.latest} available — download
      </a>
    </p>
  {/if}
  <button class="secondary-button" on:click={async () => { updateStatus = await CheckForUpdate(); }}>
    Check for updates
  </button>
</section>
```

`target="_blank"` opens the release page in the system browser — Wails handles external links this way by default; no special handling needed (consistent with how the app has no other outbound links today, so verify this once live rather than assume).

### No settings toggle UI in this slice

The backend `UpdateCheckEnabled` flag (above) exists so a future pass can add a checkbox without backend changes, but wiring an actual toggle into the Settings UI is deferred — the check is a single lightweight GitHub API call with a 3s timeout, silent on failure; the privacy tradeoff (revealing the user's IP to GitHub once per launch) is real but minor enough not to block shipping the core feature waiting on UI for an opt-out. Document the behavior in the README instead (see below), and add the toggle if a user actually asks.

---

## README addition

Under the existing "Releases" section (added in 013 slices 1–2):

```markdown
The app checks GitHub for a newer release once per launch (a single unauthenticated API
call, ~3s timeout). This requires no account and works over HTTPS to api.github.com only;
it fails silently if you're offline — the app is fully usable with no network access.
```

---

## Testing

`isNewer` and `atoiSafe` are pure functions — add `internal/archive/updatecheck_test.go` following the pattern established by `storage_test.go` (table-driven, `TestXxx` per function):

```go
func TestIsNewer(t *testing.T) {
	cases := []struct{ a, b string; want bool }{
		{"0.2.0", "0.1.0", true},
		{"0.1.0", "0.2.0", false},
		{"1.0.0", "1.0.0", false},
		{"0.10.0", "0.9.0", true}, // numeric, not lexicographic — this is why atoiSafe exists
		{"0.1", "0.1.0", false},   // short tag, treated as 0.1.0
	}
	for _, tc := range cases {
		if got := isNewer(tc.a, tc.b); got != tc.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
```

`CheckForUpdate` itself talks to the real network — not unit tested directly (no HTTP mocking infrastructure exists in this repo yet; adding one for a single call is disproportionate). Verify manually instead: temporarily hardcode an old `currentVersion` like `"0.0.1"` and confirm the Settings modal shows an update notice once slice 2's pipeline has published a real tagged release; confirm it's silent with network disabled (airplane mode / disconnect Wi-Fi).

---

## Execution order

1. `archive.CheckForUpdate` + `isNewer`/`atoiSafe` + test — pure Go, no UI, verify with `go test ./internal/archive/...`.
2. `App.CheckForUpdate` binding + `wails generate module` to regenerate bindings.
3. `AppSettings.UpdateCheckEnabled` field (backend only — no UI toggle this slice, see above).
4. Frontend: `onMount` call + Settings "About" section update.
5. README addition.
6. Manual verification against a real published release (needs slice 2's first tag to exist — until then, verify the "silently does nothing" path only).

## Risk

Low — the entire design is built around "never break, never block, never error visibly." The one thing to watch: confirm `target="_blank"` links actually open the system browser in the packaged Wails app, not just in `wails dev`'s browser-backed preview — different code paths in Wails runtime, worth a real native-window check before calling this done.
