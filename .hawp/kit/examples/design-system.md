# Mochila Design System Reference

**Created:** 2026-07-13
**Updated:** 2026-07-21 — reconciled against what actually shipped (Tailwind activation, per-platform theming); see the note at the end of each section that changed.
**Version:** v0.2
**Scope:** Frontend UI tokens, typography, color theming per provider

---

## 1. Typography

### Primary Typeface — Inter (shipped; Source Pro not adopted)

**Correction (2026-07-21):** the v0.1 draft of this doc specified switching to Source Pro via Google Fonts. That never shipped — the app still uses Inter, loaded as a system/bundled font stack with no external network request. This is deliberate, not an oversight: Mochila is local-first and works fully offline (see README), and a Google Fonts `<link>` would be the first network dependency in the UI shell. Keeping Inter is the recommended default; switching fonts is an open decision for the user, not something to assume — see the implementation plan's "Open decision" section.

**Current stack:** `Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif` (`src/frontend/src/style.css`, `body`)

### Type Scale

| Level         | Size    | Weight  | Line Height | Usage                                   |
| ------------- | ------- | ------- | ----------- | --------------------------------------- |
| h1            | 28px    | 600     | 1.0         | Page title (Mochila)                    |
| h2            | 18px    | 600     | 1.3         | Section headers                         |
| body          | inherit | 400     | 1.5         | Default text                            |
| caption/label | 12–13px | 700–800 | 1.4         | Muted text, timestamps, buttons, labels |

### Font Usage Rules

- **h1:** `font-size: 28px; font-weight: 600; line-height: 1; margin: 0;`
- **h2:** `font-size: 18px; font-weight: 600; line-height: 1.3; margin: 0; letter-spacing: 0;`
- **body:** `font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color: var(--ink);` (size inherited) — hardcoded directly today, not yet a `--font-sans` token; see § 7.1
- **eyebrow/caption:** `font-size: 12–13px; font-weight: 700; color: var(--muted);`
- **button text:** `font-weight: 800; font-size: inherit; letter-spacing: 0;`

---

## 2. Color Palette — Core Tokens

### Shared (Base) Colors

| Token     | Hex       | Usage                                            |
| --------- | --------- | ------------------------------------------------ |
| `--bg`    | `#fbfaf2` | Page background (warm paper)                     |
| `--ink`   | `#181712` | Primary text, headings (near-black)              |
| `--muted` | `#6f6b58` | Secondary text, timestamps, labels, hints        |
| `--line`  | `#ded7ad` | Borders, dividers, tab separators                |
| `--panel` | `#fffef8` | Card surfaces, list backgrounds, modal backdrops |

### Accent Colors — Provider-Specific

**Correction (2026-07-21):** the v0.1 hex values below didn't match what actually shipped (Instagram in particular used a different pink/purple pair than what was implemented). Values below are the real source of truth — `src/frontend/tailwind.config.cjs` (Tailwind utility classes) and the `platformThemes` map in `src/frontend/src/App.svelte` (CSS custom-property values, includes `ink` — not documented in v0.1 at all). Keep both in sync by hand; nothing generates one from the other.

New since v0.1: an **`--accent-ink`** token — the text color to use *on top of* an accent fill. Snapchat's yellow needs dark text; Instagram/Facebook's saturated fills need white text.

#### Snapchat Theme (live)

| Token           | Hex       | Usage                                                     |
| --------------- | --------- | ---------------------------------------------------------- |
| `--accent`      | `#fffc00` | Accent fills (active tabs, badges, highlights)            |
| `--accent-dark` | `#cfc600` | Accent borders, active state strokes, button outlines     |
| `--accent-soft` | `#fffdd9` | Subtle tints, sent-message background                     |
| `--accent-ink`  | `#181712` | Text on top of `--accent` (dark — yellow is light)         |

#### Instagram Theme (live)

| Token           | Hex       | Usage                                                     |
| --------------- | --------- | ---------------------------------------------------------- |
| `--accent`      | `#dd2a7b` | Accent fills, badge text/border                           |
| `--accent-dark` | `#8134af` | Accent borders, active state strokes                       |
| `--accent-soft` | `#fdeef5` | Subtle tints, badge background                             |
| `--accent-ink`  | `#ffffff` | Text on top of `--accent` (white — pink is saturated)      |

Dashboard card top bars use the real multi-stop Instagram gradient, not this flat fill: `bg-gradient-to-r from-[#f58529] via-instagram to-[#515bd4]` (Tailwind arbitrary-value classes, `App.svelte`) — orange → pink → blue, matching Instagram's actual brand mark. Dashboard-only decoration; the `--accent` variable (explorer header strip, active tab, sent-message bubble) stays the flat pink above.

#### Facebook Theme (live)

| Token           | Hex       | Usage                                                     |
| --------------- | --------- | ---------------------------------------------------------- |
| `--accent`      | `#1877f2` | Accent fills, badge text/border                           |
| `--accent-dark` | `#0e5fcb` | Accent borders, active state strokes                       |
| `--accent-soft` | `#e9f2fe` | Subtle tints, badge background                             |
| `--accent-ink`  | `#ffffff` | Text on top of `--accent` (white — blue is saturated)      |

### Theme Scoping Rule (corrected)

**Correction (2026-07-21):** the v0.1 draft specified a body-class + CSS-cascade mechanism (`<body class="platform-snapchat">`). That was never built. The shipped mechanism is a Svelte reactive block in `App.svelte` that calls `document.documentElement.style.setProperty(...)` directly whenever `activePlatform` changes — same effect (the four `--accent*` custom properties update live), different implementation:

```javascript
// src/frontend/src/App.svelte
$: activeTheme = platformThemes[activePlatform] ?? platformThemes.snapchat;
$: if (typeof document !== "undefined" && activeTheme) {
  const s = document.documentElement.style;
  s.setProperty("--accent", activeTheme.accent);
  s.setProperty("--accent-dark", activeTheme.dark);
  s.setProperty("--accent-soft", activeTheme.soft);
  s.setProperty("--accent-ink", activeTheme.ink);
}
```

Shared tokens (`--bg`, `--ink`, `--muted`, `--line`, `--panel`) are never overridden per platform and stay constant.

### Accessibility Baseline

| Pair                         | Contrast Ratio | WCAG AA Pass?                                |
| ---------------------------- | -------------- | -------------------------------------------- |
| `--ink` on `--bg`            | ≈14:1          | ✅ (exceeds AAA)                             |
| `--muted` on `--panel`       | ≈4.6:1         | ✅ (meets AA)                                |
| `--accent-dark` on `--panel` | ≈3.8:1         | ❌ (decorative only — use text in ink/muted) |

**Rule:** Accent colors serve as fills, borders, and decorative elements. Text read on accent surfaces should always use `--ink` or a high-contrast derived color, not rely on accent contrast alone.

---

## 3. Spacing Scale

Values observed in `style.css` are roughly 4px-based with intentional irregularities for visual rhythm. The canonical scale absorbs the majority:

| Token       | Value | Rationale                                                |
| ----------- | ----- | -------------------------------------------------------- |
| `--space-0` | 0     | Reset                                                    |
| `--space-1` | 4px   | Tight micro-spacing (eyebrow margin, label gaps)         |
| `--space-2` | 8px   | Component padding, icon spacing                          |
| `--space-3` | 12px  | Card padding, list item padding                          |
| `--space-4` | 16px  | Layout gaps (gallery, messages), button internal padding |
| `--space-5` | 20px  | Header/main content padding                              |
| `--space-6` | 24px  | Section margins                                          |
| `--space-8` | 32px  | Large section breaks                                     |

**Intentional deviations from the grid:**

- `18px` — header and main content horizontal padding (feels balanced at typical window widths)
- `34px` — bottom main content padding (visual breathing room before footer/threshold)
- `5px` — label margin-bottom (tighter than 4px feels right for form labels in context)
- `9px` — year-list item vertical padding (matches visual line-height of 12px text + gap)

**Rule:** New styles should prefer the canonical scale. Deviations require a documented reason in the code comment.

---

## 4. Border Radius

| Token           | Value | Usage                                        |
| --------------- | ----- | -------------------------------------------- |
| `--radius-sm`   | 4px   | Subtle rounding (nested badges, small pills) |
| `--radius-md`   | 8px   | Primary cards, buttons, tabs, list items     |
| `--radius-lg`   | 12px  | Modals, large panels, highlighted sections   |
| `--radius-full` | 999px | Progress bars, rounded capsules              |

---

## 5. Layout Conventions

| Property              | Value  | Usage                                                        |
| --------------------- | ------ | ------------------------------------------------------------ |
| Page max-width        | 1480px | `main`, `.topbar`, centered content                          |
| Gallery grid gap      | 10px   | Media tiles, zip list                                        |
| Controls grid gap     | 10px   | Filters, selects, inputs                                     |
| Layout column gap     | 16px   | Side-by-side (gallery sidebar + main, messages list + panel) |
| Header sticky top     | 0      | `.topbar` fixed position                                     |
| Sidebar sticky offset | 94px   | `.year-list`, `.conversation-list`                           |

---

## 6. Elevation & Effects

| Token / Rule    | Value                                                                                | Usage                  |
| --------------- | ------------------------------------------------------------------------------------ | ---------------------- |
| Header backdrop | `backdrop-filter: blur(16px); background: color-mix(in srgb, var(--bg) 94%, white);` | Glass effect on scroll |
| Card shadows    | None currently — flat design language                                                | Maintain lightness     |
| Hover states    | Background shift + subtle border accent (`var(--accent-soft)`)                       | No elevation change    |

---

## 7. Implementation Notes for AI Agents

When working with the frontend:

1. **Font family is hardcoded on `body`, not tokenized** (`font-family: Inter, ui-sans-serif, ...` directly in `style.css`) — there is no `--font-sans` custom property today, despite earlier drafts of this doc assuming one. Introducing that token (and deciding whether to keep Inter or add a webfont) is open remaining scope — see the implementation plan.
2. **Use CSS custom properties** instead of raw hex/px values — never introduce new hardcoded colors or sizes without first consulting this document. Four hardcoded hex values still remain in `style.css` as of 2026-07-21 (`.preview` background, `.placeholder` text, the `.today-hero` gradient, `.modal-media` background) — see the implementation plan for exact lines.
3. **Provider theming flows through `document.documentElement.style.setProperty`** in `App.svelte`'s reactive block, not a body class — see the Theme Scoping Rule above. Read the four `--accent*` custom properties in CSS; don't add new platform-conditional CSS selectors.
4. **All providers share the same user_id in the database** (confirmed by `platform_snapshots` table having `(platform, user_id)` composite key) — theming is purely cosmetic, data isolation already handled by backend
5. **When adding new accent colors for providers**, update *both* `tailwind.config.cjs` (Tailwind classes) and the `platformThemes` map in `App.svelte` (CSS variables) — they are two independent sources of truth today, extend the palette tables in Section 2, and document fallback/contrast considerations
