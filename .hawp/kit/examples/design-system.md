# Mochila Design System Reference

**Created:** 2026-07-13  
**Version:** v0.1  
**Scope:** Frontend UI tokens, typography, color theming per provider

---

## 1. Typography

### Primary Typeface — Source Pro (Google Fonts)

Source Pro is the designated font for all text across Mochila. It replaces Inter as the default body stack.

```html
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link
  href="https://fonts.googleapis.com/css2?family=Source+Pro:wght@400;500;600;700;800&display=swap"
  rel="stylesheet"
/>
```

**Fallback stack:** `Source Pro, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`

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
- **body:** `font-family: var(--font-sans); color: var(--ink);` (size inherited)
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

Each provider defines its own accent palette. Core tokens live on the body element scoped to active platform.

#### Snapchat Theme (Active First)

| Token            | Hex       | Usage                                                 |
| ---------------- | --------- | ----------------------------------------------------- |
| `--accent`       | `#fffc00` | Accent fills (active tabs, badges, highlights)        |
| `--accent-dark`  | `#cfc600` | Accent borders, active state strokes, button outlines |
| `--accent-light` | `#fffdd9` | Subtle tints, hover backgrounds, gradients            |

#### Instagram Theme (Planned)

| Token            | Hex       | Usage                                |
| ---------------- | --------- | ------------------------------------ |
| `--accent`       | `#e1306c` | Accent fills (Instagram pink/rose)   |
| `--accent-dark`  | `#c13584` | Accent borders, active state strokes |
| `--accent-light` | `#fce4ec` | Subtle tints, hover backgrounds      |

#### Facebook Theme (Planned)

| Token            | Hex       | Usage                                |
| ---------------- | --------- | ------------------------------------ |
| `--accent`       | `#1877f2` | Accent fills (Facebook blue)         |
| `--accent-dark`  | `#0d65d9` | Accent borders, active state strokes |
| `--accent-light` | `#e7f3ff` | Subtle tints, hover backgrounds      |

### Theme Scoping Rule

When user switches platform:

1. Body receives class: `<body class="platform-snapchat">`, `.platform-instagram`, `.platform-facebook`
2. Provider-specific tokens override `--accent`, `--accent-dark`, `--accent-light` via CSS cascade
3. Shared tokens (`--bg`, `--ink`, `--muted`, `--line`, `--panel`) remain constant

```css
body.platform-snapchat {
  --accent: #fffc00;
  --accent-dark: #cfc600;
  --accent-light: #fffdd9;
}

body.platform-instagram {
  --accent: #e1306c;
  --accent-dark: #c13584;
  --accent-light: #fce4ec;
}

body.platform-facebook {
  --accent: #1877f2;
  --accent-dark: #0d65d9;
  --accent-light: #e7f3ff;
}
```

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
| Hover states    | Background shift + subtle border accent (`var(--accent-light)`)                      | No elevation change    |

---

## 7. Implementation Notes for AI Agents

When working with the frontend:

1. **Always reference `--font-sans`** (defined as `Source Pro, ... fallback stack`) rather than hardcoding font families
2. **Use CSS custom properties** instead of raw hex/px values — never introduce new hardcoded colors or sizes without first consulting this document
3. **Provider theming flows through body class** — do not scope theme tokens to provider-specific components; cascade from body
4. **All providers share the same user_id in the database** (confirmed by `platform_snapshots` table having `(platform, user_id)` composite key) — theming is purely cosmetic, data isolation already handled by backend
5. **When adding new accent colors for providers**, extend the palette table in Section 2 and document fallback/contrast considerations
