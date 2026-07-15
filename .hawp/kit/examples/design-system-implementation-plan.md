# Design System Implementation Plan

**Created:** 2026-07-13  
**Parent Item:** [012](./012.md) — Frontend Design System (design doc only)  
**Status:** draft — not yet approved for implementation

---

## Overview

This plan describes the code changes needed to consolidate scattered CSS values into the token system defined in `.hawp/kit/examples/design-system.md`. All changes are frontend-only; no backend modifications required.

---

## File List

| File                               | Change Type                                                  |
| ---------------------------------- | ------------------------------------------------------------ |
| `src/frontend/index.html`          | Add Source Pro `<link>` tag                                  |
| `src/frontend/src/style.css`       | Rewrite: replace hardcoded values with CSS custom properties |
| `src/frontend/tailwind.config.cjs` | Align Tailwind colors with design tokens                     |
| `src/frontend/src/main.ts`         | (optional) Set initial body class for default provider theme |

---

## Step 1 — Source Pro Font Integration

### File: `src/frontend/index.html`

Add Google Fonts link in `<head>`:

```html
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link
  href="https://fonts.googleapis.com/css2?family=Source+Pro:wght@400;500;600;700;800&display=swap"
  rel="stylesheet"
/>
```

### File: `src/frontend/src/style.css`

Update body and font declarations:

```css
:root {
  /* ... existing tokens unchanged ... */
  --font-sans:
    "Source Pro", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont,
    "Segoe UI", sans-serif;
}

body {
  font-family: var(--font-sans);
}

/* Remove the old hardcoded font on button/input/select — inherit is enough */
```

---

## Step 2 — Consolidate CSS Custom Properties in `style.css`

### 2a. Add spacing and radius tokens

```css
:root {
  /* ... existing color tokens ... */

  /* Spacing scale (4px grid) */
  --space-0: 0;
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;

  /* Border radius */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-full: 999px;
}
```

### 2b. Replace hardcoded hex values

| Line (approx)                       | Current                                    | New                                                                                   |
| ----------------------------------- | ------------------------------------------ | ------------------------------------------------------------------------------------- |
| 47 (header bg)                      | `color-mix(in srgb, var(--bg) 94%, white)` | ✅ already uses vars — no change needed                                               |
| 102–103 (main padding)              | `padding: 18px 20px 34px`                  | `padding: var(--space-5) var(--space-5) var(--space-8)`                               |
| 163,172,198,200 (border-radius 8px) | `border-radius: 8px`                       | `border-radius: var(--radius-md)`                                                     |
| 247 (.bar radius)                   | `border-radius: 999px`                     | `border-radius: var(--radius-full)`                                                   |
| 265 (.preview bg)                   | `background: #171717`                      | `background: color-mix(in srgb, var(--ink) 80%, transparent)` or new `--surface-dark` |
| 277 (.placeholder text)             | `color: #f6f0d5`                           | `color: rgba(246, 240, 213, 0.9)` → later derive from accent-light                    |
| 375 (.today-hero gradient)          | `#fffef8, #f1edc9`                         | `var(--panel), var(--accent-light)`                                                   |
| 420 (today-hero bg)                 | `background: #fffdd9`                      | `background: var(--accent-light)`                                                     |
| 520 (dark fallback)                 | `background: #111`                         | `background: color-mix(in srgb, var(--ink) 85%, transparent)`                         |

### 2c. Replace hardcoded spacing values

| Context                                        | Current          | New                                                                  |
| ---------------------------------------------- | ---------------- | -------------------------------------------------------------------- |
| `.topbar padding`                              | `18px 20px 14px` | `var(--space-5) var(--space-5) var(--space-3)`                       |
| `.main padding`                                | `18px 20px 34px` | `var(--space-5) var(--space-5) var(--space-8)`                       |
| `.stats gap` / `.controls gap`                 | `10px`           | `var(--space-2)` (close to 8; document as intentional if keeping 10) |
| `.tabs gap`, `.year-list button gap`           | `8px`            | `var(--space-2)`                                                     |
| `.gallery-layout gap` / `.messages-layout gap` | `16px`           | `var(--space-4)`                                                     |
| `.tile-meta padding`                           | `8px`            | `var(--space-2)`                                                     |
| `.message-panel padding`                       | `14px`           | `var(--space-3)` (document deviation)                                |

---

## Step 3 — Provider Theming via Body Class

### In `src/frontend/src/App.svelte` (Svelte) or a Svelte action/store:

When platform switches, update body class:

```typescript
$: if (activePlatform) {
  document.body.className = `platform-${activePlatform}`;
}
```

Or in CSS directly within style.css after the `:root`:

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

**Default theme is already Snapchat (set in CSS `:root`), so no class needed for that case.**

### In `App.svelte` — update active tab indicator styling

The `.active` class on tabs should use provider accent:

```css
button.active,
.tabs button.active {
  border-color: var(--accent-dark);
  background: var(--accent);
}
```

This already works if the body class approach sets `--accent` correctly.

---

## Step 4 — Tailwind Config Alignment

### File: `src/frontend/tailwind.config.cjs`

Update to mirror CSS custom properties (Tailwind can reference them via arbitrary values or extend):

```javascript
module.exports = {
  content: ["./index.html", "./src/**/*.{svelte,ts,js}"],
  theme: {
    extend: {
      colors: {
        archive: {
          bg: "#fbfaf2",
          panel: "#fffef8",
          line: "#ded7ad",
          muted: "#6f6b58",
          ink: "#181712",
        },
        // Provider accents (documented; not actively used by Tailwind classes yet)
        snap: { DEFAULT: "#fffc00", dark: "#cfc600" },
        instagram: { DEFAULT: "#e1306c", dark: "#c13584" },
        facebook: { DEFAULT: "#1877f2", dark: "#0d65d9" },
      },
      fontFamily: {
        sans: ['"Source Pro"', "ui-sans-serif", "system-ui"],
      },
    },
  },
  plugins: [],
};
```

**Note:** Tailwind uses compiled utility classes; the CSS custom properties approach in `style.css` is preferred for theme switching. Tailwind config serves as a secondary reference for any utility-based styling.

---

## Risk Assessment

| Risk                                                      | Level  | Mitigation                                                          |
| --------------------------------------------------------- | ------ | ------------------------------------------------------------------- |
| Source Pro loading latency                                | Low    | Use `display=swap`; app is desktop (Wails) so font caching is fast  |
| CSS var regression during refactor                        | Medium | Change one section at a time; commit each step as a separate change |
| Hardcoded spacing deviations lost in consolidation        | Low    | Document all intentional deviations in code comments                |
| Provider accent colors fail contrast on panel backgrounds | Medium | Preview each theme with the WCAG AA checker before committing       |

---

## Recommended Execution Order

1. **Step 1** — Source Pro font (lowest risk, immediate visual payoff)
2. **Step 3** — Body class + provider theming CSS (enables multi-provider look without changing existing component styles)
3. **Step 2** — Consolidate all hardcoded values into tokens (medium effort, high impact on consistency)
4. **Step 4** — Tailwind config alignment (cosmetic documentation pass)

---

## What Is NOT in This Plan (Future Work)

- Dark mode support (documented as out-of-scope for #012)
- Component-level animations or transitions beyond hover states
- Redesigning existing layouts (only color/typography/spacing changes)
- Adding new providers beyond Snapchat/Instagram/Facebook scaffolding
