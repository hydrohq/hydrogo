# Admin content list theme and responsive coverage — 2026-10-10

## Change

Added the Pages and News collection lists to the responsive Admin inventory and to the authenticated WCAG 2.1 A/AA matrix. The lists are now included at 390px, 768px, and 1280px in the responsive spec, under all four Studio families in light and dark modes in the Admin theme test, and in the real Admin-session mobile/tablet overflow and touch-target check.

## Verification

- Playwright discovery from the repository worktree passed: 43 cases across `custom/a11y-responsive.spec.ts` and `custom/admin-management.spec.ts`.
- `git diff --check` passed.
- Browser execution was not available here: the Admin management fixture cannot bind its local server (`listen EPERM` on `127.0.0.1`). No responsive or axe assertions are reported as executed.
