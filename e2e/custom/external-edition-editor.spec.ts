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

  test("editor completes generated CRUD while viewer remains read-only", async ({ browser, request }) => {
    const tenant = `e2e-${randomUUID()}`;
    const roleHeaders = { "X-Test-Tenant": tenant, "X-Test-Role": "editor" };
    const editorContext = await browser.newContext({ extraHTTPHeaders: roleHeaders });
    const page = await editorContext.newPage();
    const viewerContext = await browser.newContext({ extraHTTPHeaders: { "X-Test-Tenant": tenant, "X-Test-Role": "viewer" } });
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
      const headers = { "X-Test-Tenant": tenant, "X-Test-Role": "editor" };
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
      expect(viewerList.status()).toBe(200);
      const viewerPage = await viewerContext.newPage();
      await viewerPage.addInitScript({ path: path.resolve("node_modules/axe-core/axe.min.js") });
      const viewerDocument = await viewerPage.goto(`${externalOrigin}/admin/resources/article`);
      expect(viewerDocument?.status()).toBe(200);
      await expect(viewerPage.locator('body[data-studio-role="viewer"]')).toBeVisible();
      await expect(viewerPage.getByRole("link", { name: "Yeni kayıt" })).toHaveCount(0);
      await expect(viewerPage.getByRole("link", { name: `${title} updated` })).toBeVisible();
      const viewerNavigation = viewerPage.locator(".studio-sidebar");
      for (const privilegedPath of ["/admin/users", "/admin/permissions", "/admin/security", "/admin/audit", "/admin/backup", "/admin/developer", "/admin/control", "/admin/operations"]) {
        await expect(viewerNavigation.locator(`a[href="${privilegedPath}"]`)).toHaveCount(0);
      }
      await expectNoA11yViolations(viewerPage, "viewer read-only resource list");

      const viewerRecordPath = await viewerPage.getByRole("link", { name: `${title} updated` }).getAttribute("href");
      await viewerPage.getByRole("link", { name: `${title} updated` }).click();
      await expect(viewerPage.locator('form[action*="/admin/resources/"]')).toHaveCount(0);
      await expect(viewerPage.getByText(`${title} updated`)).toBeVisible();
      expect(viewerRecordPath).toBeTruthy();
      const viewerDeniedUpdate = await request.post(`${externalOrigin}${viewerRecordPath}`, {
        headers: viewerHeaders,
        form: { _csrf: "external-edition-test-token", title: "Must remain unchanged" },
      });
      expect(viewerDeniedUpdate.status()).toBe(403);
      const viewerDeniedDelete = await request.post(`${externalOrigin}${viewerRecordPath}/delete`, {
        headers: viewerHeaders,
        form: { _csrf: "external-edition-test-token" },
      });
      expect(viewerDeniedDelete.status()).toBe(403);
      const unchanged = await request.get(`${externalOrigin}${apiPath}`, { headers: { "X-Test-Tenant": tenant, "X-Test-Role": "admin" } });
      expect(unchanged.status()).toBe(200);
      expect((await unchanged.json()).title).toBe(`${title} updated`);

      const viewerID = `viewer-${randomUUID()}`;
      const viewerCreate = await request.post(`${externalOrigin}/admin/resources/article/new`, {
        headers: viewerHeaders,
        form: { _csrf: "external-edition-test-token", id: viewerID, title: "Must be denied" },
      });
      expect(viewerCreate.status()).toBe(403);
      const viewerAPIWrite = await request.post(`${externalOrigin}/api/v1/article`, {
        headers: viewerHeaders,
        data: { id: viewerID, title: "Must be denied" },
      });
      expect(viewerAPIWrite.status()).toBe(403);
      const viewerAPIUpdate = await request.put(`${externalOrigin}${apiPath}`, {
        headers: viewerHeaders,
        data: { title: "Must remain unchanged" },
      });
      expect(viewerAPIUpdate.status()).toBe(403);
      const viewerAPIDelete = await request.delete(`${externalOrigin}${apiPath}`, { headers: viewerHeaders });
      expect(viewerAPIDelete.status()).toBe(403);
      expect((await request.get(`${externalOrigin}/api/v1/article/${viewerID}`, { headers: { "X-Test-Tenant": tenant, "X-Test-Role": "admin" } })).status()).toBe(404);
      expect((await request.get(`${externalOrigin}${apiPath}`, { headers: { "X-Test-Tenant": tenant, "X-Test-Role": "admin" } })).status()).toBe(200);

      page.once("dialog", (dialog) => dialog.accept());
      await page.locator(`form[action="${recordPath}/delete"] button[type="submit"]`).click();
      await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
      expect((await request.get(`${externalOrigin}${apiPath}`, { headers })).status()).toBe(404);
    } finally {
      await editorContext.close();
      await viewerContext?.close();
    }
  });
});

async function expectNoA11yViolations(page: import("@playwright/test").Page, context: string): Promise<void> {
  const result = await page.evaluate(async () => (window as any).axe.run(document, {
    runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
  }));
  expect(result.violations.map((violation: any) => violation.id), `${context} WCAG 2.1 A/AA violations`).toEqual([]);
}
