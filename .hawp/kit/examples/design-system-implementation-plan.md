# Design System Implementation Plan

**Created:** 2026-07-13
**Rewritten:** 2026-07-21 — most of the original v1 plan (Steps 1, 3, 4) shipped differently or not at all between 2026-07-13 and now (Tailwind activation, theming). This version reflects actual current code, not the original draft.
**Parent Item:** [012](./012.md) — Frontend Design System
**Status:** ready to execute — small, low-risk, mechanical

---

## What already shipped (no longer part of this plan)

Between the original plan (2026-07-13) and now, the following happened as part of other work items, in a different shape than originally drafted:

- **Tailwind activated** (`@tailwind base/components/utilities` in `style.css`, `preflight: false` so it layers on top of the existing hand-written CSS without resetting it). Originally planned as "Step 4 — Tailwind config alignment"; actually done first and more thoroughly — login and dashboard are now built with Tailwind utilities, not just a config file nobody used.
- **Per-platform theming**, originally planned as "Step 3 — body class". Shipped instead as a Svelte reactive block calling `document.documentElement.style.setProperty()` on `--accent`/`--accent-dark`/`--accent-soft`/`--accent-ink` (see `design-system.md` § Theme Scoping Rule for the exact code). Functionally the same outcome (CSS variables update on platform switch), different mechanism — **do not implement the body-class version**, it would conflict with what's live.
- **Real per-provider colors are live**: Snapchat yellow, Instagram pink/purple + a real gradient top bar, Facebook blue — see `tailwind.config.cjs` and the `platformThemes` map in `App.svelte`. The original plan's Instagram hex values (`#e1306c`/`#c13584`) don't match what shipped (`#dd2a7b`/`#8134af`) — `design-system.md` has been corrected to the real values as of 2026-07-21.

**What this means for this plan:** only the original "Step 2" (hardcoded hex → token consolidation) is still real, unstarted work. Steps 1, 3, 4 below are rewritten accordingly.

---

## Open decision: keep Inter, or add a webfont?

The original plan assumed switching to Source Pro via a Google Fonts `<link>`. That's a real product decision, not a mechanical change — **flagging it rather than assuming an answer**:

- **Keep Inter (recommended, no changes needed)**: already loaded, zero network dependency. Mochila is explicitly local-first and works fully offline (see README) — a Google Fonts request would be the UI's first network call, which is a meaningful architectural change for a cosmetic one.
- **Switch to a webfont**: requires either (a) a `<link>` to Google Fonts (adds the network dependency above, and the font simply won't load / silently falls back when offline — acceptable degradation, but worth being explicit that this is what happens), or (b) bundling font files into the app so it stays fully offline (larger binary, no network dependency, more setup).

Recommendation: keep Inter unless there's a specific brand reason to change. If you do want a webfont, bundling (b) fits the app's stated architecture better than a remote `<link>` (a).

---

## Step 1 — Formalize the font as a token (small, optional)

Currently `font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;` is hardcoded directly on `body` in `style.css` — there's no `--font-sans` custom property, despite `design-system.md`'s Typography section implying one exists. Purely a cleanup for consistency with the rest of the token system; skip if not worth the churn.

```css
/* src/frontend/src/style.css, in :root */
--font-sans: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;

/* body rule becomes */
body {
  margin: 0;
  background: var(--bg);
  color: var(--ink);
  font-family: var(--font-sans);
}
```

---

## Step 2 — Replace the remaining hardcoded hex values

The only real drift left. Confirmed by direct file audit on 2026-07-21 (exact current line numbers, `src/frontend/src/style.css`):

| Line | Selector           | Current                                              | Fix                                                                                                                    |
| ---- | ------------------ | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| 276  | `.preview`          | `background: #171717;`                               | New token `--surface-dark: #171717;` in `:root`, then `background: var(--surface-dark);` — this is the dark tile background behind loading photo/video thumbnails, intentionally darker than any existing token (not derivable from `--ink` via `color-mix` without a visible shift — checked, `color-mix(in srgb, var(--ink) 80%, transparent)` renders lighter than the current `#171717`). |
| 288  | `.placeholder`       | `color: #f6f0d5;`                                     | Same rationale — this is cream text *on the dark `.preview` background above*, not related to any existing light-mode token. New token `--surface-dark-ink: #f6f0d5;` in `:root`, used only by `.placeholder`.                                                                                    |
| 386  | `.today-hero` gradient | `linear-gradient(135deg, #fffef8, #f1edc9)`         | `#fffef8` is already `--panel` verbatim — swap directly: `linear-gradient(135deg, var(--panel), var(--accent-soft))`. Verify visually: `--accent-soft` is `#fffdd9` (Snapchat) by default, close to but not identical to `#f1edc9` — expected, since this hero block should ideally react to the active platform accent like the rest of the header now does, and currently doesn't (separate finding below). |
| 531  | `.modal-media`       | `background: #111;`                                   | New token `--surface-dark: #171717;` (reuse from line 276) — same "media letterboxing" concept, close enough hex distance (`#111` vs `#171717`) that using one token for both is the right call, not two near-duplicate ones.                                                                     |

Net result: two new tokens (`--surface-dark`, `--surface-dark-ink`) added to `:root`, both existing accent tokens (`--panel`, `--accent-soft`) reused at line 386 instead of introduced.

### Related finding: `.today-hero` doesn't follow the active platform accent

While auditing line 386, noticed the "On This Day" hero card's background gradient and radial highlight (`rgba(255, 252, 0, 0.65)` — hardcoded Snapchat yellow) don't update when switching platforms, unlike the header strip / active tab / sent-message bubble (all of which already read `--accent`). Minor visual inconsistency once Instagram/Facebook explorers exist — worth fixing in the same pass as the token swap above:

```css
.today-hero {
  background:
    radial-gradient(circle at 20% 20%, color-mix(in srgb, var(--accent) 65%, transparent), transparent 30%),
    linear-gradient(135deg, var(--panel), var(--accent-soft));
}
```

---

## Step 3 — Tailwind config alignment

Already done (see "What already shipped" above) — `tailwind.config.cjs` has `archive.*`, `snapchat.*`, `instagram.*`, `facebook.*` color groups matching `design-system.md` § 2 exactly as of the 2026-07-21 correction. **Nothing to do here.**

---

## Execution order

1. Step 2 (hex → token) — the only functionally meaningful change, ~10 lines across `style.css`, no new dependencies, no visual regression risk beyond a quick eyeball check of the gallery tile placeholder, the media modal letterbox, and the On This Day hero.
2. Step 1 (font token) — optional, purely organizational, zero visual change since Inter's value is copied verbatim into the new variable.
3. Resolve the "Open decision" above if a webfont is wanted — otherwise nothing further to do.

## Risk

Low. All changes are token substitutions with identical or near-identical resulting values (the `.today-hero` gradient shifts from a fixed cream to a platform-reactive tint — visually check after switching platforms, not just on Snapchat). No layout, spacing, or structural changes.

## Out of scope (unchanged from original plan)

- Dark mode support
- Component-level animations/transitions beyond existing hover states
- Redesigning layouts (color/token-only pass)
