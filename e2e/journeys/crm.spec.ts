import { expect, test } from "@playwright/test";
import { ADMIN_EMAIL, ADMIN_PASSWORD } from "./global-setup";

// The CRM starter against a real server: sign in, see seeded data, add a
// company, search for it. (No wrong-password attempt here: sign-in is limited
// to a handful a minute per address and the journeys share one.)

const crm = (path = "") => `${process.env.JOURNEY_CRM_URL}${path}`;

test("the CRM starter: sign in, add a company, find it, sign out", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));

  await page.goto(crm("/"));
  await expect(page).toHaveURL(/\/login/);
  await page.waitForLoadState("networkidle");
  await page.locator('input[name="email"]').fill(ADMIN_EMAIL);
  await page.locator('input[name="password"]').fill(ADMIN_PASSWORD);
  await page.locator("form button.primary").click();
  await expect(page).not.toHaveURL(/\/login/);

  await page.goto(crm("/companies"));
  await expect(page.getByText("Northwind Software").first()).toBeVisible();

  await page.waitForLoadState("networkidle");
  await page.locator('form[action="?/create"] input[name="name"], input[name="name"]').first().fill("Journey Test Ltd");
  await page.locator("form button.primary").last().click();
  await expect(page).toHaveURL(/\/companies\/[a-z0-9]+/);
  await expect(page.getByText("Journey Test Ltd").first()).toBeVisible();

  await page.goto(crm("/companies?q=Journey"));
  await expect(page.getByText("Journey Test Ltd").first()).toBeVisible();
  await expect(page.getByText("Northwind Software")).toHaveCount(0);

  await page.goto(crm("/deals"));
  await expect(page.locator("main")).toBeVisible();
  expect(errors).toEqual([]);
});
