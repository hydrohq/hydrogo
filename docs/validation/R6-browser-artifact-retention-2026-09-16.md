# Browser E2E CI artifact evidence (2026-09-16)

`e2e/package-lock.json` pins Playwright version to 1.63.0. The CI browser job selects Node.js 22.14.0 via action-SHA pinned `setup-node`; `npm ci` and `npx playwright install chromium` use this pinned toolchain. Test configuration generates traces, screenshots, and videos on test failures; regardless of job outcome, `e2e/artifacts`, `e2e/test-results`, and `e2e/playwright-report` paths are uploaded via `actions/upload-artifact`.

This configuration only improves CI observability; artifact retention and access duration remain subject to GitHub Actions repository policy.
