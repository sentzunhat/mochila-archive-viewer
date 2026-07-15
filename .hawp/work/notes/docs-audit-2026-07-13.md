# Documentation Audit Results — 2026-07-13

## README.md Corrections Applied

| Drift                                                                                                                                     | Fix                                                                                                 | Evidence |
| ----------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- | -------- |
| "prepares modular provider lanes for Instagram and Facebook" → "has modular provider lanes for Instagram (active) and Facebook (planned)" | Provider status: `instagram/provider.go` Status="active", `facebook/provider.go` Status="planned"   |
| "Snapchat is the only active provider target right now" → "Snapchat is currently the first-importer target; Instagram support is active"  | Same provider files + `archive/service.go`: `Supported: p.ID() == "snapchat"` (first-importer flag) |
| `inbox/` listed in Layout but doesn't exist                                                                                               | Removed phantom reference from README layout section                                                |
| No Go module path for main.go                                                                                                             | Added "(Go module path: `mochila-archive-viewer/src`)" matching `src/go.mod`                        |

## Missing Notes Restored

- Old prototype note: "The old prototype stays in `archive/snapchat-export/` until the Go app reaches full feature parity across all providers."
- Cache detail line: Provider artifacts under `~/.mochila/indexed/providers/<provider>/`

## Frontend README Verified

No drift found — SQLite paths and cache structure match backend implementation.
