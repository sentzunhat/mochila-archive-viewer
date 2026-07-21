# 014 — Browser-driven UI smoke test evidence (2026-07-20)

Environment: `wails dev` from `src/` (must run unsandboxed; sandboxed launch exits with "Development mode exited"). Browser driven against `http://localhost:34115` with live Go bindings.

## Direct evidence

- Login screen is the entry point when logged out. ✓ (screenshot pass 1)
- Login as new user `diego` → dashboard rendered. Initially **no platform cards** — bug B2 below. After fix, all 3 cards render.
- Dashboard for a fresh user shows all-zero "empty" cards — correct per-user scoping.
- Login as `legacy` (profile_id 1, the pre-multi-user data owner) → Snapchat card shows **indexed, 8,617 media, 6 zips, 5,650 photos, 2,967 videos, 33 conversations, 23 JSON files** — matches SQLite counts exactly.
- Logout from dashboard returns to login screen with cleared fields. ✓

## Bugs found and fixed during the pass

- **B1** Stray literal text `// Page size for gallery display` rendered on the dashboard (was raw text in App.svelte markup, line ~1169). Removed.
- **B2** Dashboard cards never rendered: `loadPlatformDashboard` mutated `platformStatsList` with `.push()` — Svelte needs reassignment. Fixed by building a local array and assigning once.
- **B3** Clicking a platform card never left the dashboard: `selectPlatform` didn't clear `showDashboard`. Fixed; also added a "← Dashboard" button in the explorer topbar.
- **B4** `PlatformStats` (storage.go) hardcoded ImageCount/VideoCount to 0 and only counted image+video rows in MediaCount. Rewritten as three straightforward COUNT queries.
- **B5** Login never switched the backend active user: `Service.SaveProfile` saved the profile but left `activeUserId` unchanged, and `Store.SaveProfile` didn't log out other profiles (multiple `logged_in=1` rows possible). Fixed: exclusive login, SaveProfile returns the profile id, service activates it.
- **B6** Stale per-user cache: `Service.platform()` caches PlatformState and never invalidated it on user switch/logout — the explorer showed the previous user's data after login/logout (the reported "logged out but you still go to Snapchat" weirdness). Fixed: cache cleared in SetActiveUser/SelectUser/Logout, and frontend `selectPlatform` reloads the snapshot before paging.
- **B7** Logged-out sessions ran as user_id **0**: `GetFrontendState` took `ActiveUser().ID` (0 when nobody is logged in) and stamped it as active user; historical writes duplicated all archive data under user_id 0 and the only `platform_snapshots` row lived under user 0. Fixed the guard (`p.ID > 0`); repointed the snapshot row to user_id 1.

## Data repairs applied to ~/.mochila/database.sqlite

- Profile 1 (empty username, owner of the 8,617 indexed items) renamed to `legacy` / "Legacy (pre-multi-user data)" so it is reachable from the login screen.
- `platform_snapshots` snapchat row moved user_id 0 → 1.

## Deferred (inference, needs its own item)

- Full duplicate sets remain under user_id 0 in media_items/archive_files/conversations/json_files (8,617/6/33/23 rows). Unreachable after the B7 fix. Deleting them and/or reassigning legacy data to the user's real profile is work item 015.
