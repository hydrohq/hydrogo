# HydroGo semantic UI contract

All first-party surfaces should consume semantic tokens rather than hard-coded
palette values. The Admin Studio currently provides the reference vocabulary;
public themes may map the same names to a different visual identity.

Core exposes dependency-free semantic adapters for `native`, `shadcn`,
`material`, and `carbon` through `NewDefaultDesignSystemRegistry`. These
adapters only map intent to class names; provider CSS/runtime packages remain
Client Edition choices and can be replaced by a custom registry provider.

| Token group | Required semantic names | Purpose |
|---|---|---|
| Type | `--font-sans`, `--font-mono`, `--text-*` | readable hierarchy and code/data text |
| Spacing | `--space-*` | predictable layout rhythm |
| Color | `--bg`, `--surface`, `--ink`, `--muted`, `--brand`, `--danger` | light/dark and shadcn mappings |
| Shape | `--radius-sm`, `--radius-md`, `--radius-lg` | consistent control geometry |
| Elevation | `--shadow-xs`, `--shadow-md`, `--shadow-lg` | restrained depth |
| Motion | `--motion-fast`, `--motion-normal`, `--motion-slow` | animation budget and reduced-motion override |

Components must preserve semantic HTML, visible focus, keyboard operation, and
an accessible name. A theme can change token values, but not the meaning of a
component state. New shared components belong in `internal/ui`; edition-specific
markup remains in the edition theme.
