# Backlog

Active index for current open work in this repository.
Closed history belongs under `.hawp/work/closed/YYYY/MM/DD/` and should not accumulate forever here.
Each row links to its plan file when one exists.

---

## Status Key

| Status        | Meaning                             |
| ------------- | ----------------------------------- |
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

## Recently Closed (Backtracked)

Work was implemented before HAWP tracking was established. Backtracking for visibility.

| #   | Status | Title                                                                   | Plan File                         | Commit  |
| --- | ------ | ----------------------------------------------------------------------- | --------------------------------- | ------- |
| 1   | done   | Stabilize archive viewer app                                            | [001](./closed/2025/07/09/001.md) | cfcd727 |
| 2   | done   | Enhance JSON explorer with structured analysis and media loading states | [002](./closed/2025/07/09/002.md) | 7413f34 |
| 3   | done   | Add search bar, settings page, optimize OnThisDay video                 | [003](./closed/2025/07/09/003.md) | c5af4df |
| 10  | done   | Multi-user isolation — wire profile to indexed data                     | [010](./active/010.md)            | eb2c321 |

---

## Active Work (Compoundable Next Steps)

Items below are compoundable: each can be done independently or together as a batch. Planning first, review before implementation.

| #   | Status      | Title                                               | Plan File                        |
| --- | ----------- | --------------------------------------------------- | -------------------------------- |
| 4   | done        | Backend: thread activeUser through Wails bindings   | [active/004.md](./active/004.md) |
| 5   | done        | Frontend: refine search UX with keyboard shortcuts  | [active/005.md](./active/005.md) |
| 6   | done        | Integration test: build + run wails dev cycle       | [active/006.md](./active/006.md) |
| 7   | done        | Match POC UI structure in new Go implementation     | [active/007.md](./active/007.md) |
| 8   | done        | Fix media source loading gaps across views          | [active/008.md](./active/008.md) |
| 9   | parked      | Multi-platform provider scaffolding — investigate   | [active/009.md](./active/009.md) |
| 10   | done        | Multi-user isolation — wire profile to indexed data | [active/010.md](./active/010.md) | [eb2c321](https://github.com/sentzunhat/mochila-archive-viewer/commit/eb2c321)

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
- Keep Recently Closed capped; do not append completed history forever.
- Work started outside this loop should still get a row added for visibility.
