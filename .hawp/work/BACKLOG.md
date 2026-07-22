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

| #   | Status      | Title                                                        | Plan File                        | Next action                                                |
| --- | ----------- | ------------------------------------------------------------- | --------------------------------- | ------------------------------------------------------------ |
| 20  | plan-ready  | Architecture & code quality audit (Zacatl-aligned)           | [active/020.md](./active/020.md) | Start with Part A (SQL fixes) — highest priority, data loss bug in platform_snapshots PK |
| 9   | parked      | Multi-platform provider scaffolding — investigate             | [active/009.md](./active/009.md) | Unblocked (016 done); size Instagram provider work before starting |

---

## Recently Closed

Capped to the 10 most recent closures. Older closed items are not deleted — they remain under `.hawp/work/closed/YYYY/MM/DD/`, just not listed here.

| #   | Title                                                       | Closed     | Plan File                                          |
| --- | ------------------------------------------------------------ | ---------- | ---------------------------------------------------- |
| 19  | Serve media over HTTP not base64-RPC (filter-switch fix)      | 2026-07-22 | [closed/2026/07/22/019.md](./closed/2026/07/22/019.md) |
| 13  | Auto-update support — full end-to-end test passed             | 2026-07-22 | [closed/2026/07/22/013.md](./closed/2026/07/22/013.md) |
| 18  | Update check blocked by private repo — repo made public      | 2026-07-22 | [closed/2026/07/22/018.md](./closed/2026/07/22/018.md) |
| 12  | Frontend design system — tokens/spacing/typography           | 2026-07-21 | [closed/2026/07/21/012.md](./closed/2026/07/21/012.md) |
| 15  | Legacy data ownership — user_id 0/1 cleanup                  | 2026-07-21 | [closed/2026/07/21/015.md](./closed/2026/07/21/015.md) |
| 16  | Snapchat UI parity+polish vs POC (unblocks IG/FB UI)          | 2026-07-20 | [closed/2026/07/20/016.md](./closed/2026/07/20/016.md) |
| 17  | Link chat message media to indexed media items                | 2026-07-20 | [closed/2026/07/20/017.md](./closed/2026/07/20/017.md) |
| 14  | Browser-driven UI smoke test of the dev build                 | 2026-07-20 | [closed/2026/07/20/014.md](./closed/2026/07/20/014.md) |
| 11  | Profile management — multi-user creation/switching            | 2026-07-13 | [closed/2026/07/13/011.md](./closed/2026/07/13/011.md) |
| 10  | Multi-user isolation — wire profile to indexed data            | 2026-07-13 | [closed/2026/07/13/010.md](./closed/2026/07/13/010.md) |

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
