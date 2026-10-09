# Admin Studio UI quality audit — 2026-10-08

## Scope and evidence

This pass inspected the media library surface and the shared Admin shell using
the committed desktop, tablet, and mobile media-library goldens, existing
Chromium/axe acceptance results, current Admin CSS, the responsive E2E contract,
and the Impeccable detector. The detector CLI in this environment exposes
`detect`, not the skill's `audit` command. A fresh responsive browser run was
attempted twice. The fixture's synchronous Go setup commands now accept Node's
sandbox `EPERM` report only when the child process actually exited with status
0; the run then reached server setup and failed because this environment
denies binding `127.0.0.1` (`listen EPERM`). Therefore, no fresh screenshot or
computed browser style was available for this pass.

## Audit health

| Dimension | Score | Evidence |
| --- | ---: | --- |
| Accessibility | 3/4 | Existing Admin axe and keyboard coverage is strong. This pass found the media-card quick actions were 28×28px on touch layouts; CSS and a specific responsive assertion now require 44×44px, but that assertion has not run in a browser here. |
| Performance | 3/4 | Media listing is bounded and assets use responsive thumbnails. No fresh performance profile was available. Detector `transition:width` findings resolve to progress and state indicators, not general layout animation. |
| Theming | 4/4 | Admin shell uses semantic tokens and has four tested theme families with light/dark coverage; existing axe matrix passed 112/112. |
| Responsive design | 3/4 | Existing 390px/768px route matrix passed without overflow. This pass closes a previously unmeasured media action target in CSS and adds a browser assertion; runtime confirmation remains open because the fixture could not start. |
| Implementation integrity | 4/4 | The media screen follows the shared Studio shell and card/control vocabulary. Detector warnings were checked: the 4px accent border is a rich-text blockquote marker; the 2px neutral border marks nested navigation; Inter is self-hosted for the app; width transitions are progress/timer indicators. These are intentional and not media-screen defects. |
| **Total** | **17/20** | **Good, with one touch-target change awaiting browser confirmation.** |

## Findings and changes

### P2 — Media quick actions were below touch-target size

- **Location:** `internal/ui/admin_shell.go`, `AdminMediaCSS`, mobile media-card overlay.
- **Impact:** The open-asset and copy-link controls were 28×28px on touch viewports, making frequent media operations harder to tap accurately.
- **Standard:** The Studio's existing mobile control contract is 44×44px for main actions.
- **Change:** Increased both dimensions to 44px for the touch/narrow layout and extended the responsive E2E assertion to check every visible `.media-thumb-overlay .media-overlay-btn`.
- **Verification:** `go test ./internal/ui -count=1` passed; Playwright listed all five admin management tests. The responsive test was attempted with a writable isolated output directory. Go setup and migration completed, but the fixture could not bind its local test server (`listen EPERM`), so fresh browser measurement and visual golden comparison remain open.

## Detector results verified as intentional

- The side-tab warning at `internal/ui/admin_shell.go` points to the semantic blockquote marker in rich text.
- The neutral border warning points to the nested navigation-tree hierarchy.
- Inter is bundled locally and matches the Studio's product UI direction; it is not fetched from a third-party CDN.
- Width transitions are limited to upload progress, report progress, traffic meters, and toast timers. The detector output alone does not establish a layout-thrashing defect.

## Remaining visual-quality gates

- Re-run `admin routes fit mobile and tablet viewports` and capture the media
  golden at 390px/768px in an environment that permits a local test listener.
- Complete the remaining generated/editorial and Collections theme/accessibility
  matrices recorded in the product plan.
- No desktop/tablet layout change was made; the target-size adjustment applies
  only to touch/narrow media-card controls.
