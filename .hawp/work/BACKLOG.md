# Backlog

Active index for current open work in this repository.
Closed history belongs under `.hawp/work/closed/YYYY/MM/DD/` and should not accumulate forever here.
Each row links to its plan file when one exists.

---

## Status Key

| Status        | Meaning                             |
| ------------- | ------------------------------------ |
| `inbox`       | Received, not yet analyzed          |
| `analyzing`   | Under investigation                 |
| `plan-ready`  | Plan written, awaiting review       |
| `approved`    | Plan approved, ready to implement   |
| `in-progress` | Being implemented                   |
| `parked`      | Deferred without closing            |
| `done`        | Implemented and verified            |
| `blocked`     | Blocked — reason noted in plan file |
| `wont-fix`    | Decided not to fix — reason noted   |

---

## Active Work

| #   | Status      | Title                                                | Plan File                        | Next action                                                |
| --- | ----------- | ----------------------------------------------------- | --------------------------------- | ------------------------------------------------------------ |
| 9   | parked      | Multi-platform provider scaffolding — investigate    | [active/009.md](./active/009.md) | Unblocked (016 done); size Instagram provider work before starting |
| 13  | in-progress | Auto-update support — slice 3 built, tested live      | [active/013.md](./active/013.md) | Blocked on 018's decision |
| 18  | blocked     | Update check can never succeed while repo is private | [active/018.md](./active/018.md) | Needs user decision: make repo public, or a different distribution channel |

---

## Recently Closed

Capped to the 10 most recent closures. Older closed items are not deleted — they remain under `.hawp/work/closed/YYYY/MM/DD/`, just not listed here.

| #   | Title                                                       | Closed     | Plan File                                          |
| --- | ------------------------------------------------------------ | ---------- | ---------------------------------------------------- |
| 12  | Frontend design system — tokens/spacing/typography           | 2026-07-21 | [closed/2026/07/21/012.md](./closed/2026/07/21/012.md) |
| 15  | Legacy data ownership — user_id 0/1 cleanup                  | 2026-07-21 | [closed/2026/07/21/015.md](./closed/2026/07/21/015.md) |
| 16  | Snapchat UI parity+polish vs POC (unblocks IG/FB UI)          | 2026-07-20 | [closed/2026/07/20/016.md](./closed/2026/07/20/016.md) |
| 17  | Link chat message media to indexed media items                | 2026-07-20 | [closed/2026/07/20/017.md](./closed/2026/07/20/017.md) |
| 14  | Browser-driven UI smoke test of the dev build                 | 2026-07-20 | [closed/2026/07/20/014.md](./closed/2026/07/20/014.md) |
| 11  | Profile management — multi-user creation/switching            | 2026-07-13 | [closed/2026/07/13/011.md](./closed/2026/07/13/011.md) |
| 10  | Multi-user isolation — wire profile to indexed data            | 2026-07-13 | [closed/2026/07/13/010.md](./closed/2026/07/13/010.md) |
| 8   | Fix media source loading gaps across views                    | 2026-07-09 | [closed/2026/07/09/008.md](./closed/2026/07/09/008.md) |
| 7   | Match POC UI structure in new Go implementation                | 2026-07-09 | [closed/2026/07/09/007.md](./closed/2026/07/09/007.md) |
| 6   | Integration test: build + run wails dev cycle (backtracked)   | 2025-07-09 | [closed/2025/07/09/006.md](./closed/2025/07/09/006.md) |

---

## Archive

- Closed work: `.hawp/work/closed/`
- Status reports: `.hawp/work/status/`
- Evidence: `.hawp/work/evidence/`
- Decisions: `.hawp/work/decisions/`

---

## Notes

- Check this file before starting any new item.
- Each item gets one plan file under `.hawp/work/active/` — no two agents on the same ID.
- Deferred items can move to `.hawp/work/parked/` without being closed.
- On close, move the plan file to `.hawp/work/closed/YYYY/MM/DD/`.
- Keep Recently Closed capped at 10 items; do not append completed history forever. Nothing is deleted — everything closed lives under `.hawp/work/closed/`.
- Work started outside this loop should still get a row added for visibility.
- Compacted 2026-07-21: moved 004-008, 010-012, 014-017 from `active/` to `closed/`; trimmed this table from 14 rows (11 done) down to the 2 genuinely open items.
