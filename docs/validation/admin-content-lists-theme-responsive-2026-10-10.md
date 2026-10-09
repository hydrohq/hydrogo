# Admin content list theme and responsive coverage — 2026-10-10

## Change

Added the Pages and News collection lists, the Pages import screen, News taxonomy management, and the role journey's persisted Page and News draft detail screens to the real Admin-session mobile/tablet overflow and touch-target check and the authenticated WCAG 2.1 A/AA matrix. The responsive inventory also lists Pages and News at 390px, 768px, and 1280px. Theme checks cover all four Studio families in light and dark modes.

## Verification

- Playwright discovery from the repository worktree passed: 43 cases across `custom/a11y-responsive.spec.ts` and `custom/admin-management.spec.ts`.
- `moduleguard`, `migrationguard` (111 migrations), Client/News/Construction Edition `importguard`, and `assetguard` all passed.
- `git diff --check` passed.
- Browser execution was not available here: the Admin management fixture cannot bind its local server (`listen EPERM` on `127.0.0.1`). No responsive or axe assertions are reported as executed.
