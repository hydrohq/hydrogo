import { test, expect } from "@playwright/test";

// Visual checks are opt-in because they require a running seeded server and
// approved goldens. CI enables HYDRO_VISUAL once its baseline artifact store
// is configured; ordinary functional E2E remains fast and deterministic.
test.describe("HydroGo visual contract", () => {
  test.skip(!process.env.HYDRO_VISUAL, "set HYDRO_VISUAL=1 to run approved visual goldens");

  const stableSurfaces = [
    { id: "home", path: "/public/pages/tr/demo/html", auth: false },
    { id: "news", path: "/public/news/tr/html", auth: false },
    { id: "admin-login", path: "/login", auth: false },
    { id: "admin-dashboard", path: "/admin", auth: true },
    { id: "admin-pages-editor", path: "/admin/pages/new", auth: true },
    { id: "admin-news-editor", path: "/admin/news/new", auth: true },
    { id: "admin-media-library", path: "/admin/media", auth: true },
  ] as const;

  for (const surface of stableSurfaces) {
    test(`${surface.id} visual baseline`, async ({ page }) => {
      await page.goto(surface.path, { waitUntil: "domcontentloaded" });
      await page.waitForTimeout(300);
      await page.evaluate(() => document.fonts?.ready);
      await expect(page).toHaveScreenshot(`${surface.id}.png`, {
        fullPage: true,
        animations: "disabled",
        caret: "hide",
        maxDiffPixelRatio: 0.02,
      });
    });
  }
});
