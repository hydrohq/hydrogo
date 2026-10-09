import { randomUUID } from "node:crypto";
import path from "node:path";
import { test, expect } from "@playwright/test";

const externalEdition = process.env.HYDRO_E2E_EXTERNAL_EDITION === "1";
const externalOrigin = process.env.HYDRO_E2E_BASE_URL?.trim() ?? "";
const externalDatabase = process.env.HYDRO_E2E_DATABASE_DSN?.trim() ?? "";

test.describe("independent Edition generated Admin editor journey", () => {
  test.skip(!externalEdition, "requires the external Edition test server and database");

  test.beforeAll(async () => {
    if (!externalOrigin || !externalDatabase) {
      throw new Error("HYDRO_E2E_EXTERNAL_EDITION requires HYDRO_E2E_BASE_URL and HYDRO_E2E_DATABASE_DSN");
    }
    const url = new URL(externalOrigin);
    const host = url.hostname.toLowerCase().replace(/^\[|\]$/g, "");
    if (url.protocol !== "http:" || url.username || url.password || url.pathname !== "/" || url.search || url.hash || !["localhost", "127.0.0.1", "::1"].includes(host)) {
      throw new Error("HYDRO_E2E_BASE_URL must be a loopback HTTP origin without credentials or a path");
    }
    const health = await fetch(`${url.origin}/healthz`);
    if (!health.ok) throw new Error(`external Edition health check returned ${health.status}`);
  });

  test("editor completes generated CRUD while viewer remains denied", async ({ browser, request }) => {
    const tenant = `e2e-${randomUUID()}`;
    const roleHeaders = { "X-Test-Tenant": tenant, "X-Test-Role": "editor" };
    const editorContext = await browser.newContext({ extraHTTPHeaders: roleHeaders });
    const page = await editorContext.newPage();
    await page.addInitScript({ path: path.resolve("node_modules/axe-core/axe.min.js") });

    try {
      const listResponse = await page.goto(`${externalOrigin}/admin/resources/article`);
      expect(listResponse?.status()).toBe(200);
      await expect(page.locator('body[data-studio-role="editor"]')).toBeVisible();
      await expect(page.getByRole("heading", { name: "Articles" })).toBeVisible();
      await expectNoA11yViolations(page, "editor resource list");

      await page.setViewportSize({ width: 390, height: 844 });
      expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(1);
      await page.getByRole("link", { name: "Yeni kayıt" }).click();
      await expect(page).toHaveURL(/\/admin\/resources\/article\/new$/);
      const title = `External editor ${randomUUID()}`;
      const form = page.locator('form[action="/admin/resources/article/new"]');
      await form.getByLabel("Title").fill(title);
      await expectNoA11yViolations(page, "editor create form");
      await form.locator('button[type="submit"]').click();
      await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
      await expect(page.getByRole("link", { name: title })).toBeVisible();
      const recordLink = page.getByRole("link", { name: title });
      const recordPath = await recordLink.getAttribute("href");
      expect(recordPath).toMatch(/^\/admin\/resources\/article\//);
      const apiPath = recordPath!.replace("/admin/resources", "/api/v1");
      const headers = { "X-Test-Tenant": tenant };
      const created = await request.get(`${externalOrigin}${apiPath}`, { headers });
      expect(created.status()).toBe(200);
      expect((await created.json()).title).toBe(title);

      await recordLink.click();
      const editForm = page.locator(`form[action="${recordPath}"]`);
      await editForm.getByLabel("Title").fill(`${title} updated`);
      await editForm.locator('button[type="submit"]').click();
      await expect(page.getByRole("link", { name: `${title} updated` })).toBeVisible();
      const updated = await request.get(`${externalOrigin}${apiPath}`, { headers });
      expect(updated.status()).toBe(200);
      expect((await updated.json()).title).toBe(`${title} updated`);

      const viewerHeaders = { "X-Test-Tenant": tenant, "X-Test-Role": "viewer" };
      const viewerList = await request.get(`${externalOrigin}/admin/resources/article`, { headers: viewerHeaders });
      expect(viewerList.status()).toBe(403);
      const viewerID = `viewer-${randomUUID()}`;
      const viewerCreate = await request.post(`${externalOrigin}/admin/resources/article/new`, {
        headers: viewerHeaders,
        form: { _csrf: "external-edition-test-token", id: viewerID, title: "Must be denied" },
      });
      expect(viewerCreate.status()).toBe(403);
      expect((await request.get(`${externalOrigin}/api/v1/article/${viewerID}`, { headers: { "X-Test-Tenant": tenant } })).status()).toBe(404);

      page.once("dialog", (dialog) => dialog.accept());
      await page.locator(`form[action="${recordPath}/delete"] button[type="submit"]`).click();
      await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
      expect((await request.get(`${externalOrigin}${apiPath}`, { headers })).status()).toBe(404);
    } finally {
      await editorContext.close();
    }
  });
});

async function expectNoA11yViolations(page: import("@playwright/test").Page, context: string): Promise<void> {
  const result = await page.evaluate(async () => (window as any).axe.run(document, {
    runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
  }));
  expect(result.violations.map((violation: any) => violation.id), `${context} WCAG 2.1 A/AA violations`).toEqual([]);
}
