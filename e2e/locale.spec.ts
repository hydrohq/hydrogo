import { test, expect } from "@playwright/test";

// Locale fixtures are opt-in with the visual browser suite because they need a
// running seeded server and intentionally exercise the rendered document root.
test.describe("HydroGo locale and direction contract", () => {
  test.skip(!process.env.HYDRO_VISUAL, "set HYDRO_VISUAL=1 to run locale fixtures");

  test("Turkish renders a bounded LTR document", async ({ page }) => {
    await page.goto("/public/news/tr/html", { waitUntil: "domcontentloaded" });
    await expect(page.locator("html")).toHaveAttribute("dir", "ltr");
    await expect(page.locator("html")).toHaveAttribute("lang", "tr");
  });

  test("Arabic renders RTL without horizontal overflow", async ({ page }) => {
    await page.goto("/public/news/ar/html", { waitUntil: "domcontentloaded" });
    await expect(page.locator("html")).toHaveAttribute("dir", "rtl");
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
    expect(overflow).toBe(false);
  });

  test("long mixed-direction text remains bounded", async ({ page }) => {
    await page.goto("/public/news/tr/html", { waitUntil: "domcontentloaded" });
    await page.evaluate(() => {
      const heading = document.querySelector("h1");
      if (heading) heading.textContent = "Çok uzun başlık — العربية — עברית — 1234567890".repeat(4);
    });
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
    expect(overflow).toBe(false);
  });
});
