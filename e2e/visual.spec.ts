import { test, expect } from "@playwright/test";

// Visual checks are opt-in because they require a running seeded server and
// approved goldens. CI enables HYDRO_VISUAL once its baseline artifact store
// is configured; ordinary functional E2E remains fast and deterministic.
test.describe("HydroGo visual contract", () => {
  test.skip(!process.env.HYDRO_VISUAL, "set HYDRO_VISUAL=1 to run approved visual goldens");

  for (const surface of [
    ["home", "/public/pages/tr/demo/html"],
    ["news", "/public/news/tr/html"],
  ] as const) {
    test(`${surface[0]} desktop`, async ({ page }) => {
      await page.goto(surface[1], { waitUntil: "domcontentloaded" });
      await page.waitForTimeout(300);
      await page.evaluate(() => document.fonts?.ready);
      await expect(page).toHaveScreenshot(`${surface[0]}-desktop.png`, {
        fullPage: true,
        animations: "disabled",
        caret: "hide",
        maxDiffPixelRatio: 0.02,
      });
    });
  }
});
