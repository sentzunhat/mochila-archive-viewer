# Status — 2026-07-20 — UI smoke test, user-scoping fixes, work-item alignment

## What happened

Continued from the 2026-07-13 checkpoint. Ran the app live (`wails dev` from `src/`, browser-driven against `http://localhost:34115`) and executed the full login → dashboard → explorer → logout flow. Seven bugs found and fixed; full detail with direct evidence in `.hawp/work/evidence/2026/07/20/014-ui-smoke-test.md`.

## Fixes (direct evidence — all verified live in the browser)

**Frontend (`src/frontend/src/App.svelte`)**
- Removed stray literal `// Page size for gallery display` rendered on the dashboard.
- Dashboard cards now render: `platformStatsList` is reassigned instead of `.push()`-mutated.
- Clicking a platform card now leaves the dashboard (`showDashboard = false`) and refreshes the per-user snapshot before paging; added a "← Dashboard" topbar button.
- `loadMediaBatch` no longer loops until it has fetched the entire archive (was pulling all 8,617 items in one call); it fetches one page and reassigns `paginatedMedia` so the grid actually updates.
- Infinite-scroll observer attaches reactively when the sentinel renders (previously attached before the sentinel existed → never fired).
- A single failed media-source load no longer replaces the whole app with the global error screen.
- Merged the duplicate logout functions into one `handleLogout`; logout clears error state and login fields.
- Login subtitle is platform-neutral ("Personal data archive explorer").
- Deleted seven leftover one-off Python patch scripts from `src/frontend/`.

**Backend (`src/internal/archive/`, `src/appshell/`)**
- Login now actually switches the backend active user: `Store.SaveProfile` returns the profile id and logs out all other profiles (exclusive login); `Service.SaveProfile` activates that id.
- User switch/login/logout invalidates the per-platform state cache — this was the reported "logged out but still in Snapchat data" bug.
- `GetFrontendState` no longer adopts user_id 0 when nobody is logged in (`p.ID > 0` guard). Historical writes under user 0 are why the DB has duplicate data (see item 015).
- `PlatformStats` returns real image/video counts (were hardcoded 0) and counts all media types.

**Data repairs (`~/.mochila/database.sqlite`)**
- Profile 1 (empty username, owns the 8,617 items) renamed to `legacy` so it is reachable from login.
- `platform_snapshots` snapchat row moved from user_id 0 → 1.

## Verification

- `go build ./...` and `go test ./...` pass (no test files exist — see gap below).
- `npm run build` passes (64 KB bundle).
- Live browser flow: login as `legacy` → dashboard shows Snapchat indexed / 8,617 media / 5,650 photos / 2,967 videos / 33 conversations (matches SQLite exactly) → explorer loads 180 tiles → Load-more grows to 360 → ← Dashboard works → logout returns to a clean login screen.
- Not verifiable in the embedded test browser: IntersectionObserver callbacks never fire there (renderer throttling), so auto-scroll loading was verified by code review + manual Load-more only. Check visually in the native window.

## Work-item alignment

- Backlog drift fixed: items 011 (profile mgmt, done) and 012 (design system, plan-ready) existed on disk but were missing from BACKLOG.md — now listed.
- New: 013 auto-update support (inbox, sliced), 014 UI smoke test (done, evidence recorded), 015 legacy data ownership/cleanup (inbox, needs a user decision on data reassignment).

## Open next steps (compoundable)

1. **015** — decide legacy data ownership; delete unreachable user_id 0 duplicates (backup first).
2. **013** — auto-update slices: embed build version → release CI → update check.
3. **012** — design-system tokens (plan-ready).
4. **009** — multi-platform provider support (backend still Snapchat-only; dashboard already renders Instagram/Facebook cards).
5. Add backend tests — `go test ./...` currently runs zero tests; storage/service user-scoping is now complex enough to deserve them.
6. Split App.svelte (~1,280 lines) into login/dashboard/explorer components.
