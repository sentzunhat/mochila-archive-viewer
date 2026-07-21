# 016 — Gallery filter + Messages auto-load bug fixes, live evidence (2026-07-20)

Environment: `wails dev` from `src/`, browser driven against `http://localhost:34115`, logged in as `legacy` (owner of the 8,617-item Snapchat archive).

## Data Explorer — correction

Earlier note (same day, before this fix pass) claimed the Data Explorer tab had no rendering branch. Live-tested and that was wrong: clicking Data Explorer renders the file list, and it auto-opened `cameos_metadata.json` showing heading ("mydata~....zip · object · 3 items"), top-level keys, snapshot path, and pretty JSON. No fix needed; corrected the record in `active/016.md`.

## Bug 1 — Gallery search/year/category/type filters were inert

Root cause: `visibleMedia = selectedPlatform ? paginatedMedia : searchableMedia.slice(...)`. Since every reachable gallery view has `selectedPlatform` set, the entire client-side `filteredMedia`/`searchableMedia` chain (driven by `selectedCategory`/`selectedType`/`searchQuery`) was dead code, and the year picker's `setYear` refetched an unfiltered `media` array that was never rendered either.

Fix: added `MediaFilter{Year, Category, Type, Search}` threaded through `Store.MediaPaginated`/`MediaCount` (SQL `WHERE` clauses, `LIKE ... ESCAPE '\'` for search) → `Service` → `appshell.App` (Wails binding, `archive.MediaFilter` struct param) → frontend (`currentMediaFilter()`, `reloadFilteredMedia()`).

### Live verification (all counts cross-checked against `sqlite3 ~/.mochila/database.sqlite`)

| Filter | Result |
|---|---|
| Baseline (no filter) | 8,617 |
| Search "chat_media" | 4,548 |
| Category "chat_media" | 4,548 (same field, consistent) |
| Type "video" | 2,967 (matches dashboard's video count) |
| Reset to baseline after each | correctly returns to 8,617 |

### Race condition found and fixed during testing

Rapidly toggling two filters (e.g. type=video then immediately type=all) could leave the UI showing a stale count (observed: "0 of 2,967" — the *previous* filter's count with the *new*, empty result set) when the first (slower) request's response arrived after the second (faster) one. Root cause: `loadMediaBatch`'s `isLoadingMore` guard blocked the second request from an unrelated in-flight first request in some interleavings, and nothing prevented an old response from overwriting a newer one.

Fix: added a `mediaGeneration` counter, bumped on every filter reload; `run()` force-clears `isLoadingMore` so a new filter always preempts an old one; both the count and page fetch inside `loadMediaBatch` check `gen === mediaGeneration` before applying results, discarding stale ones. Re-verified after the fix: sequential filter changes (with realistic ~300ms+ spacing) all settle to the correct count.

### Known follow-up (not a correctness bug)

Switching a filter while up to 180 concurrent `GetMediaSource` calls are still resolving from the *previous* filter makes the UI feel sluggish for a few seconds — a `computer` tool call literally timed out (30s) once during testing while 180 video thumbnails were loading, though the page itself was not frozen (confirmed via direct `javascript_exec` calls succeeding immediately after). Noted in item 016's scope as a performance item, not fixed in this pass.

## Bug 2 — Messages tab: conversation summary showed, but zero message bubbles

Root cause #1: the reactive auto-load guard was `selectedConversation?.messages?.length === 0`. Go's `encoding/json` serializes a nil `[]ChatMessage` slice as `null`, not `[]` (no `omitempty` on the field) — so list-only conversations arrive with `messages: null`, and `null?.length === 0` is `undefined === 0` → always false. The auto-load never fired on initial render; only a manual click (which calls `openConversation` directly) worked.

Root cause #2 (surfaced only after fixing #1): a `loadedConversationIds` Set was added to prevent retry-looping a genuinely-empty conversation. It wasn't cleared when `loadPlatform()` re-fetched a fresh (message-less) snapshot. Sequence: the pre-login `onMount` call to `loadPlatform()` triggered the (now-working) auto-load once, successfully merging real messages and marking the id "loaded" — then the post-login `selectPlatform → loadPlatform` call reset `conversations` back to the message-less snapshot, and the "already loaded" guard silently blocked ever retrying, leaving the UI stuck on the empty state permanently.

Diagnostic method: monkey-patched `window.go.appshell.App.GetConversation` to log invocations — confirmed it was never called despite all reactive-statement conditions appearing true. Direct calls to the binding (`GetConversations` list vs `GetConversation` singular) confirmed `messages: null` in the list and 9 real messages in the singular fetch, isolating the bug to the frontend auto-load orchestration rather than the backend.

Fix: falsy check instead of `=== 0`; clear `loadedConversationIds` inside `loadPlatform()` whenever a fresh snapshot is fetched (not just on logout/user-switch).

### Live verification

Fresh login as `legacy` → Messages tab (no click on any conversation) → heading "olatunjiobabi25 · 9 messages · 9 media references" **and** 9 `.message` article elements rendered. Confirmed via `document.querySelectorAll('.messages .message').length === 9`.

## New work item spun out

- **017** (plan-ready): user asked whether chat media (`mediaType: "MEDIA"`) can be linked to a thumbnail. Confirmed a viable match: the first `|`-delimited token in `messages.media_ids` appears verbatim as a substring in the matching `media_items.entry` filename. Not implemented this pass — needs a batch in-memory lookup (not per-message `LIKE` scans) to scale to the largest conversation's 472 media-bearing messages. See `active/017.md` for the full plan.
