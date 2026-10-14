# ADR-0053: Semantic UI token contract

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0041](0041-admin-visual-theme-modes.md), [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md)

Admin, member, and public theme surfaces use semantic token names for visual
meaning. Themes may remap values for light, dark, or shadcn modes, but shared
component state and accessibility semantics remain stable.
