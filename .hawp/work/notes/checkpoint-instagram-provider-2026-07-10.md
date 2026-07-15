# Checkpoint: Instagram Provider Implementation — 2026-07-10

## Current Status
**Build State:** FAILING (exit code 1 from `go build ./...`)
**Root Cause:** Type mismatch between `types.*` and `snapchat.*` packages

## What's Done
1. ✅ Shared types package: `src/internal/types/shared.go` (MediaItem, Index, Conversation, ChatMessage, JsonFileRef, ZipMeta, Provider)
2. ✅ Instagram provider files created:
   - `src/internal/providers/instagram/provider.go` (status: "active")
   - `src/internal/providers/instagram/indexer.go` (walks ZIPs, finds media+JSON)
   - `src/internal/providers/instagram/parser.go` (parses threadMessage JSON)
3. ✅ UNIQUE constraint bug fixed in `storage.go::SaveSelection()` (INSERT → INSERT OR REPLACE)

## What's Incomplete (Blocking Compilation)
4. ⚠️ **Service layer wiring incomplete:**
   - `state.go`: Imports updated to `types.*` from `snapchat.*`
   - `service.go`: Imports NOT fully migrated — still references `snapchat.MediaItem`, `snapchat.Index`, etc. (12+ type mismatches)
   - `storage.go`: Not yet updated (still uses `snapchat.MediaItem`)
   - `appshell/app.go`: Not yet updated

5. ⚠️ **Platform routing not implemented:**
   - `IndexArchives()` always calls `snapchat.IndexZips()` — needs switch to call `instagram.IndexZips()` for instagram platform
   - `ProviderCards().Supported` hardcoded to `p.ID() == "snapchat"` — should check provider status

## Architecture Decision (Made)
- **Pattern:** Provider interface in shared types package; each platform implements it
- **Shared types:** All providers use `internal/types/` for cross-platform compatibility — no local type duplication
- **Indexed storage:** `~/.mochila/indexed/providers/{platform}/media/` with JSON snapshots

## Next Steps (Priority Order)
1. Add `import "mochila-archive-viewer/src/internal/types"` to service.go
2. Replace all `snapchat.MediaItem` → `types.MediaItem` in service.go, storage.go, appshell/app.go
3. Replace all `snapchat.Conversation` → `types.Conversation`
4. Replace all `snapchat.JsonFileRef` → `types.JsonFileRef`
5. Replace all `snapchat.Index` → `*types.Index`
6. Add platform routing switch in `IndexArchives()`: if platform=="instagram" → `instagram.IndexZips()`, else `snapchat.IndexZips()`
7. Update `ProviderCards()`: change `Supported: p.ID() == "snapchat"` to check provider status or supported platforms list
8. Remove unused facebook import (status not yet active)
9. Run `go build ./...` and verify clean compilation
10. End-to-end test with real Instagram export

## Constraints & Notes
- Do NOT introduce new per-provider type definitions — always use shared types package
- Provider interface must include: ID(), Name(), Status(), Description() methods
- Parser implementations should return types.Conversation (not provider-specific) for consistency
- When adding facebook support, reuse the same indexing patterns from instagram/snapchat (~80% pipeline reuse)
