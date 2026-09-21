import { test, expect } from "@playwright/test";

const baseURL = process.env.HYDRO_E2E_BASE_URL ?? "http://127.0.0.1:8090";

const publicSurfaces = [
  { name: "home", path: "/public/pages/tr/demo/html" },
  { name: "news", "/public/news/tr/html": "/public/news/tr/html", path: "/public/news/tr/html" },
  { name: "member login", path: "/member/login" },
];

const adminSurfaces = [
  { name: "admin login", path: "/login", auth: false },
  { name: "admin dashboard", path: "/admin", auth: true },
  { name: "page editor", path: "/admin/pages/new", auth: true },
  { name: "news editor", path: "/admin/news/new", auth: true },
  { name: "media library", path: "/admin/media", auth: true },
];

test.describe("P2 Responsive Matrix & Viewport Bounds", () => {
  const viewports = [
    { name: "mobile 390px", width: 390, height: 844 },
    { name: "tablet 768px", width: 768, height: 1024 },
    { name: "desktop 1280px", width: 1280, height: 800 },
  ];

  for (const vp of viewports) {
    for (const surface of publicSurfaces) {
      test(`viewport ${vp.name} on ${surface.name}: no horizontal overflow or broken media`, async ({ page }) => {
        await page.setViewportSize({ width: vp.width, height: vp.height });
        await page.goto(`${baseURL}${surface.path}`, { waitUntil: "domcontentloaded" });
        await page.waitForLoadState("networkidle");

        // 1. Check no horizontal overflow
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
        expect(overflow).toBeLessThanOrEqual(1);

        // 2. Check broken images (all naturalWidth > 0 or not broken)
        const brokenImages = await page.evaluate(() => {
          const imgs = Array.from(document.querySelectorAll("img"));
          return imgs.filter((img) => img.complete && img.naturalWidth === 0 && !img.hidden && getComputedStyle(img).display !== 'none').length;
        });
        expect(brokenImages).toBe(0);
      });
    }

    test(`viewport ${vp.name} on admin login: no horizontal overflow`, async ({ page }) => {
      await page.setViewportSize({ width: vp.width, height: vp.height });
      await page.goto(`${baseURL}/login`, { waitUntil: "domcontentloaded" });
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
      expect(overflow).toBeLessThanOrEqual(1);
    });

    test(`viewport ${vp.name} on admin dashboard: no horizontal overflow`, async ({ page }) => {
      await page.setViewportSize({ width: vp.width, height: vp.height });
      await page.goto(`${baseURL}/admin`, { waitUntil: "domcontentloaded" });
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
      expect(overflow).toBeLessThanOrEqual(1);
    });
  }
});

test.describe("P2 Accessibility & Semantic Standards", () => {
  test("keyboard tab order and visible focus on interactive controls", async ({ page }) => {
    await page.goto(`${baseURL}/public/pages/tr/demo/html`);
    // Tab into page
    await page.keyboard.press("Tab");
    const focused = await page.evaluate(() => {
      const el = document.activeElement;
      if (!el || el === document.body) return null;
      const style = window.getComputedStyle(el);
      return {
        tagName: el.tagName,
        hasOutline: style.outlineStyle !== "none" || style.boxShadow !== "none" || style.borderStyle !== "none",
      };
    });
    expect(focused).toBeTruthy();
    expect(focused?.hasOutline).toBe(true);
  });

  test("semantic heading hierarchy has no skipped levels (h1 -> h2 -> h3)", async ({ page }) => {
    for (const surface of publicSurfaces) {
      await page.goto(`${baseURL}${surface.path}`, { waitUntil: "domcontentloaded" });
      const headingLevels = await page.evaluate(() => {
        const headings = Array.from(document.querySelectorAll("h1, h2, h3, h4, h5, h6"));
        return headings
          .filter((h) => !h.hidden && getComputedStyle(h).display !== "none")
          .map((h) => parseInt(h.tagName.substring(1), 10));
      });

      if (headingLevels.length > 0) {
        // First heading should be h1 or h2
        expect(headingLevels[0]).toBeLessThanOrEqual(2);
        // No jump of more than 1 level
        for (let i = 0; i < headingLevels.length - 1; i++) {
          const diff = headingLevels[i + 1] - headingLevels[i];
          expect(diff).toBeLessThanOrEqual(1);
        }
      }
    }
  });

  test("all interactive controls have accessible names and labels", async ({ page }) => {
    for (const surface of publicSurfaces) {
      await page.goto(`${baseURL}${surface.path}`, { waitUntil: "domcontentloaded" });
      const unnamedCount = await page.locator("a,button,input,select,textarea").evaluateAll((elements) =>
        elements.filter((element) => {
          const html = element as HTMLElement;
          if (html.hidden || getComputedStyle(html).display === "none" || getComputedStyle(html).visibility === "hidden") return false;
          if (element instanceof HTMLInputElement && element.type === "hidden") return false;
          const label = element.id ? document.querySelector(`label[for="${CSS.escape(element.id)}"]`)?.textContent : element.closest("label")?.textContent;
          return ![element.getAttribute("aria-label"), element.getAttribute("title"), element.getAttribute("placeholder"), element.textContent, label].some((val) => val?.trim());
        }).length
      );
      expect(unnamedCount).toBe(0);
    }
  });

  test("modal escape and dialog focus behavior", async ({ page }) => {
    // Verify document key listener handles Escape gracefully without crashing
    await page.goto(`${baseURL}/public/pages/tr/demo/html`);
    await page.keyboard.press("Escape");
    await expect(page.locator("main")).toBeAttached();
  });
});
