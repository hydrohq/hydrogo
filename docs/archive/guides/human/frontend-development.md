# Frontend Development (Human Guide)

HydroGo encompasses two distinct frontend systems:

| | Public Site Presentation (Themes) | Admin Studio Panel |
|---|---|---|
| **Audience** | Public site visitors | Content creators, editors, administrators |
| **Location** | `themes/<theme-name>/` | `modules/designsystem/`, `internal/ui/` |
| **Template Syntax** | Standard Go HTML Templates (`html/template`) | Templ (`.templ` typed templates) |
| **Styling** | Tailwind CSS compiled per theme | Tailwind CSS compiled per admin bundle |
| **Hot Reloading** | Fast template re-render on save | Auto-compiled in live reload when running `hydro run` |

---

## 1. Customizing the Public Site (Themes)

Inside a theme directory such as `themes/corporate/`:
```text
layouts/base.html        ← Root HTML skeleton (headers, footers, meta tags)
blocks/hero.html          ← Reusable modular blocks (homepage, landing pages)
blocks/cta.html
blocks/faq.html
...
static/theme.source.css  ← SOURCE CSS: Apply custom style rules and Tailwind directives here
static/theme.css         ← COMPILED CSS: Built automatically; do not edit directly
```

HTML layout and block templates can be edited directly without compile steps. When updating colors, utility classes, or styles in `theme.source.css`, recompile the theme stylesheet using the theme build script.

**Cache Busting:** Whenever theme CSS is recompiled, update the asset cache version parameter (`theme.css?v=...`) to ensure browsers immediately fetch the updated stylesheet.

---

## 2. Modifying the Admin Studio

Admin Studio views and UI components are authored using **Templ** (`.templ` files). When running `hydro run`, changes to `.templ` files are watched and recompiled automatically.

> [!WARNING]
> Never manually edit files ending in `_templ.go`. These are machine-generated Go files; any manual changes will be overwritten on the next compile step.

---

## 3. High-Fidelity UI Guidelines (React / Next.js Quality in Go SSR)

HydroGo provides modern, fluid server-rendered UI without JavaScript framework bloat, leveraging typed Templ components, Tailwind CSS, and HTMX.

### Step 1: Establish Visual Direction & Design Tokens
Define visual scale, font hierarchies, spacing tokens, border radii, and color palettes in theme tokens rather than scattering arbitrary values across templates. Consult [`DESIGN.md`](../../../DESIGN.md) and the [semantic UI token contract](../../architecture/0053-semantic-ui-token-contract.md).

### Step 2: Assemble Views Using Typed Components
Explore existing building blocks in `internal/ui/components/` and `modules/designsystem/`. Use typed primitives like `ui.Button`, `ui.Input`, `ui.Card`, and `ui.Form` to inherit accessibility defaults, proper HTML escaping, and CSRF protection automatically. Place custom client business logic under `custom/` rather than modifying `generated/` artifacts.

### Step 3: Design for All Interaction States
Do not design solely for the ideal populated state. Explicitly implement and verify:
- Loading states and skeletons
- Empty states with clear calls to action
- Validation error banners and field-level error feedback
- Success confirmations
- Disabled controls
- Keyboard focus rings and ARIA labels

### Step 4: Responsive & Visual Quality Verification
- Verify viewports across mobile (360px), tablet (768px), and desktop (1440px) to prevent horizontal overflow.
- Confirm keyboard navigation reaches all interactive controls with clear focus indicators.
- Respect `prefers-reduced-motion` in animations and transitions.
- Test with real content in both light and dark color modes.
