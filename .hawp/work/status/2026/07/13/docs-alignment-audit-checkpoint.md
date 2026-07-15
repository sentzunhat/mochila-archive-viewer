# Checkpoint: Documentation Alignment Audit — 2026-07-13

## What Changed

### README.md Corrections (4 drift items fixed)

| Before                                                       | After                                                                          | Evidence                                                                                                           |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------ |
| "prepares modular provider lanes for Instagram and Facebook" | "has modular provider lanes for Instagram (active) and Facebook (planned)"     | `src/internal/providers/instagram/provider.go` Status = "active"; `facebook/provider.go` Status = "planned"        |
| "Snapchat is the only active provider target right now"      | "Snapchat is currently the first-importer target; Instagram support is active" | Same provider status files + `archive/service.go` line 81: `Supported: p.ID() == "snapchat"` (first-importer flag) |
| `inbox/` listed in Layout section                            | Removed from layout — directory doesn't exist in repo                          | `find . -type d` output confirmed no `inbox/` directory                                                            |
| No Go module path specified for main.go                      | Added "(Go module path: `mochila-archive-viewer/src`)"                         | `src/go.mod` line 1: `module mochila-archive-viewer/src`                                                           |

### Missing Notes Restored (2 items)

- "The old prototype stays in `archive/snapchat-export/` until the Go app reaches full feature parity across all providers."
- Provider-specific cached artifacts detail: `~/.mochila/indexed/providers/<provider>/`

### Frontend README.md Verified (no drift found)

- SQLite path: `~/.mochila/database.sqlite` ✅ matches `src/internal/archive/storage.go` line 46
- Provider cache root: `~/.mochila/indexed/providers/` ✅ matches `storage.go` line 78
- Snapchat media/snapshot paths ✅ match `storage.go` lines 83, 87

## Sources Checked

- `README.md` — project overview & notes
- `src/frontend/README.md` — frontend-specific claims
- `archive/docs/snapchat-export-notes.md` — archived planning (noted as historical reference)
- `src/go.mod` — module declaration
- `src/internal/providers/*/provider.go` — provider status claims
- `src/internal/archive/service.go` — ProviderCards(), Supported flag
- `src/internal/archive/storage.go` — SQLite paths, cache structure

## Drift Classification

| Item                                                       | Severity   | Type                 | Fix Applied                  |
| ---------------------------------------------------------- | ---------- | -------------------- | ---------------------------- |
| Instagram status: "prepares" → "active"                    | Medium     | Factual inaccuracy   | ✅ Fixed in README.md header |
| Snapchat: "only active provider" → "first-importer target" | Medium     | Outdated scope claim | ✅ Fixed in README.md Notes  |
| `inbox/` directory: claimed exists but doesn't             | Low-Medium | Phantom reference    | ✅ Removed from Layout       |
| No module path for main.go                                 | Low        | Incomplete info      | ✅ Added to Layout           |
| Missing prototype note                                     | Low        | Omission             | ✅ Restored to Notes         |
| Missing cache detail line                                  | Low        | Omission             | ✅ Restored to Notes         |

## Documentation Health Score

| Metric                    | Score    | Notes                                                                    |
| ------------------------- | -------- | ------------------------------------------------------------------------ |
| README accuracy           | 8/10     | Fixed provider status + removed phantom reference; module path now clear |
| Frontend README alignment | 10/10    | Storage paths match backend exactly                                      |
| Provider docs freshness   | 6/10     | Facebook still "planned" across all docs — expected, no action needed    |
| Overall health            | **8/10** | Minor drift identified and corrected; no code changes required           |

### Risk Assessment

- **Risk Level:** Low
- **Critical drift:** None found (inbox/ was a phantom reference, not misleading)
- **Recommended next steps:**
  - When Instagram support becomes fully production-ready, verify frontend README still matches
  - Re-audit when Facebook provider is added to `src/internal/providers/facebook/provider.go`

### Strict Rules Cited (docs-alignment.schema.json)

✅ treat_src_as_single_source_of_truth — all claims verified against `src/` implementation  
✅ no_hallucination_only_verifiable_facts — every drift item backed by file path or command output  
✅ evidence_must_use_file_paths_and_line_ranges — table includes exact source references  
✅ do_not_modify_code_only_suggest_docs — README.md only; no code changes applied
