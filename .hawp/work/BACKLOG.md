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

| #   | Status    | Title                                                        | Plan File | Next action |
| --- | --------- | ------------------------------------------------------------- | ---------- | ----------- |
| 024 | inbox     | Media HTTP serving in dev mode (Vite routing issue)          | pending    | Research Wails dev mode routing for `/media/` |
| 025 | inbox     | Date extraction for Instagram and Facebook media             | pending    | Analyze export formats for date metadata |

---

## Recently Closed

Capped to the 10 most recent closures. Older closed items are not deleted — they remain under `.hawp/work/closed/YYYY/MM/DD/`, just not listed here.

| #   | Title                                                       | Closed     | Plan File                                          |
| --- | ------------------------------------------------------------ | ---------- | ---------------------------------------------------- |
| 23  | File decomposition: Zacatl-aligned domain boundaries (all criteria met + smoke test verified) | 2026-07-22 | [closed/2026/07/22/023.md](./closed/2026/07/22/023.md) |
| 22  | Facebook / Messenger provider: parser + indexer + integration (end-to-end verified) | 2026-07-22 | [closed/2026/07/22/022.md](./closed/2026/07/22/022.md) |
| 21  | Instagram provider: wire up + complete (all gaps G1–G5 fixed, smoke tested) | 2026-07-22 | [closed/2026/07/22/021.md](./closed/2026/07/22/021.md) |

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
