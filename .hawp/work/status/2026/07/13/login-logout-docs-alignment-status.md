# Status Report

#### Intent

Capture a current-state summary of the app work items the user tested, with emphasis on login/logout behavior and docs alignment.

#### Current State

The repo has a live Go + Wails app in `src/`, with backend/frontend work partially aligned to the HAWP backlog. The current implementation still has some documented drift:

- The frontend still presents a Snapchat-first flow in several places.
- The backend currently only allows the Snapchat platform through `Service.platform(...)`.
- The docs/backlog say Instagram support is active, but the live code still blocks non-Snapchat platforms.
- The logout/login UI flow is present, but it does not fully reset the app into a clean documented state.

#### What Was Inspected

- `.hawp/kit/start-here.md`
- `.hawp/kit/usage/status-report.md`
- `.hawp/work/BACKLOG.md`
- `.hawp/work/STATUS.md`
- `.hawp/work/active/004.md`
- `.hawp/work/active/005.md`
- `.hawp/work/active/006.md`
- `.hawp/work/active/007.md`
- `.hawp/work/active/008.md`
- `.hawp/work/active/009.md`
- `.hawp/work/active/010.md`
- `.hawp/work/active/011.md`
- `.hawp/work/active/012.md`
- `README.md`
- `src/frontend/README.md`
- `src/appshell/app.go`
- `src/internal/archive/service.go`
- `src/frontend/src/App.svelte`
- repo status / untracked file list

#### What Changed

No code changes were made in this checkpoint. I created this report as the handoff artifact.

#### What Was Directly Verified

- `README.md` says the live app “starts with Snapchat” and that Instagram support is active.
- `src/frontend/README.md` still documents Snapchat-specific cache paths.
- `src/internal/archive/service.go` contains `Supported: p.ID() == "snapchat"` and returns `ErrPlatformNotSupported` for any platform other than Snapchat.
- `src/appshell/app.go` still exposes `SaveProfile`, `LogoutProfile`, `ActiveUserProfile`, `AvailableUsers`, and `SelectUser`.
- `src/frontend/src/App.svelte` has both login/dashboard state and the archive UI state in one component.
- `src/frontend/src/App.svelte` logout currently clears some UI state, but it does not change the active platform or rebuild a clean neutral landing state.
- The frontend still hardcodes the login subtitle to `Snapchat Data Explorer`.
- Active work items `004` through `012` exist, with `012` marked `plan-ready`; `009` is parked; the rest are mostly marked `done` or `inbox`.

#### What Remains Unproven

- Whether the user’s “after logout you go to Snapchat” report is caused by the current frontend rendering path, by a stale cached build, or by an unverified runtime flow in Wails dev.
- Whether Instagram-related docs are already outdated on purpose or whether the implementation is still behind the documented state.
- Whether the active backlog items were completed in separate branches or only partially reflected in the live checkout.

#### Constraints

- Scope stayed limited to the current repository and the files tied to the user’s bug report.
- I did not modify code in this pass.
- Findings are based on static inspection, not a fresh runtime smoke test.

#### Help Wanted

Review the platform routing and login/logout UX together, because the current code and docs disagree on what should happen after logout and which providers are actually supported.

#### Suggested Next Step

Normalize the live app state transitions first:

1. Make logout return to a neutral login/profile landing state instead of a Snapchat-branded screen.
2. Reconcile the provider support flags in the backend and docs.
3. Run the Wails dev cycle and confirm the observed login/logout behavior in a live window.
