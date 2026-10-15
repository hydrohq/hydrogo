import { test, expect } from "@playwright/test";

test.describe("HydroGo Admin SPA", () => {
  test("loads admin shell and redirects unauthenticated visitor to login", async ({ page }) => {
    await page.goto("/admin");
    // Should show the SPA title and login form
    await expect(page).toHaveTitle(/HydroGo/i);
    await expect(page.locator('input[type="email"], input[name="email"]')).toBeVisible();
    await expect(page.locator('input[type="password"], input[name="password"]')).toBeVisible();
  });

  test("can authenticate and navigate dashboard modules", async ({ page }) => {
    const email = process.env.HYDRO_ADMIN_EMAIL || process.env.HYDRO_E2E_ADMIN_EMAIL;
    const password = process.env.HYDRO_ADMIN_PASSWORD || process.env.HYDRO_E2E_ADMIN_PASSWORD;

    test.skip(!email || !password, "Admin credentials required for authenticated journey");

    await page.goto("/admin/#/login");
    await page.locator('input[type="email"], input[name="email"]').fill(email!);
    await page.locator('input[type="password"], input[name="password"]').fill(password!);
    await page.locator('button[type="submit"]').click();

    // Verify successful dashboard landing
    await expect(page.locator("aside")).toBeVisible();
    await expect(page.getByText("Dashboard")).toBeVisible();
  });
});
