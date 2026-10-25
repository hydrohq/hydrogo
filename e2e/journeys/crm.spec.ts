import { expect, test } from "@playwright/test";
import { ADMIN_EMAIL, ADMIN_PASSWORD } from "./global-setup";

// The CRM starter against a real server: sign in, see seeded data, add a
// company, search for it. (No wrong-password attempt here: sign-in is limited
// to a handful a minute per address and the journeys share one.)

const crm = (path = "") => `${process.env.JOURNEY_CRM_URL}${path}`;

test("the CRM starter: sign in, add a company, find it, trust the dashboard totals", async ({ page, request }) => {
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
  await page.getByRole("button", { name: "New company" }).click();
  await page.getByRole("dialog").locator('input[name="name"]').fill("Journey Test Ltd");
  await page.getByRole("dialog").getByRole("button", { name: "Save" }).click();
  await expect(page).toHaveURL(/\/companies\/[a-z0-9]+/);
  await expect(page.getByText("Journey Test Ltd").first()).toBeVisible();

  await page.goto(crm("/companies?q=Journey"));
  await expect(page.getByText("Journey Test Ltd").first()).toBeVisible();
  await expect(page.getByText("Northwind Software")).toHaveCount(0);

  await page.goto(crm("/deals"));
  await expect(page.locator("main")).toBeVisible();
  const token = process.env.JOURNEY_HYDRO_TOKEN!;
  const headers = { authorization: `Bearer ${token}` };
  const api = `${process.env.JOURNEY_HYDRO_URL}/api/v1/admin/content/deals`;
  const before = (await (await request.get(`${api}/_aggregate?group_by=stage`, { headers })).json()).data.groups as { key: string; count: number }[];
  const open = (before ?? []).filter((g) => ["lead", "qualified", "proposal", "negotiation"].includes(g.key)).reduce((n, g) => n + g.count, 0);

  const company = (await (await request.get(`${process.env.JOURNEY_HYDRO_URL}/api/v1/admin/content/companies?limit=1`, { headers })).json()).data[0].id;
  for (let i = 0; i < 105; i++) {
    const res = await request.post(api, { headers, data: { name: `Bulk ${i}`, company, stage: "lead", amount: 10, currency: "TRY" } });
    expect(res.status()).toBe(201);
  }

  // The dashboard must count every deal, not just the first page of 100.
  await page.goto(crm("/"));
  await expect(page.locator("main")).toContainText(String(open + 105));
  await expect(page.locator("main")).toContainText("1,050");

  expect(errors).toEqual([]);
});
