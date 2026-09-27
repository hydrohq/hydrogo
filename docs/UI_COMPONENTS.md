# Shared UI components

HydroGo now exposes a small, server-rendered component contract in
`internal/ui/components.go`. The helpers provide the repeated primitives used
by Admin, public themes, and member surfaces:

- `Button` — semantic action button with escaped attributes and disabled state
- `Field` and `FormError` — labeled controls and non-sensitive validation copy
- `Table` — captioned metadata table with scoped column headers
- `Pagination` — bounded, rel-marked previous/next navigation
- `Empty`, `Loading`, and `Error` — consistent status and failure states

The helpers return `template.HTML` only after escaping every caller-provided
value. They intentionally do not accept arbitrary markup or runtime errors.
New server-rendered surfaces should use these helpers instead of introducing a
local variant. Existing legacy templates can migrate incrementally; the
semantic selectors (`button`, `form-error`, `ui-table`, `ui-pagination`, and
state classes) remain compatible with both Hydro and shadcn visual modes.

The public page design system in `modules/designsystem` provides typed Templ
blocks for hero, CTA, pricing, testimonials, before/after, counters, form
success, contact, FAQ, copy, sticky headers, video, masonry, and image
hotspots. The same components are exposed through `internal/ui` for
first-party server-rendered views. They use semantic theme tokens, native
browser controls where possible, and a small deferred script for optional
interactions. FAQ blocks add `FAQPage` JSON-LD to the document head.

Browser-only components (Modal, Toast, Tabs) still need a lifecycle and
focus-management contract before being shared safely.
