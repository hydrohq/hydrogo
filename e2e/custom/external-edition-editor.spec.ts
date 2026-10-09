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
    const adminHeaders = { "X-Test-Tenant": tenant, "X-Test-Role": "admin" };
    const editorContext = await browser.newContext({ extraHTTPHeaders: roleHeaders });
    const page = await editorContext.newPage();
    const viewerContext = await browser.newContext({ extraHTTPHeaders: { "X-Test-Tenant": tenant, "X-Test-Role": "viewer" } });
    await page.addInitScript({ path: path.resolve("node_modules/axe-core/axe.min.js") });
    const lockedID = `locked-${randomUUID()}`;
    const lockedTitle = "Policy protected record";

    try {
      const seededLockedRecord = await request.post(`${externalOrigin}/api/v1/article`, {
        headers: adminHeaders,
        data: { id: lockedID, title: lockedTitle },
      });
      expect(seededLockedRecord.status()).toBe(201);
      const listResponse = await page.goto(`${externalOrigin}/admin/resources/article`);
      expect(listResponse?.status()).toBe(200);
      await dismissOnboarding(page);
      await expect(page.locator('body[data-studio-role="editor"]')).toBeVisible();
      await expect(page.getByRole("heading", { name: "Articles" })).toBeVisible();
      await expect(page.getByRole("link", { name: lockedTitle })).toBeVisible();
      await expectNoA11yViolations(page, "editor resource list");

      await page.setViewportSize({ width: 390, height: 844 });
      expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(1);
      await page.getByRole("link", { name: "Yeni kayıt" }).click();
      await expect(page).toHaveURL(/\/admin\/resources\/article\/new$/);
      const title = `External editor ${randomUUID()}`;
      const form = page.locator('form[action="/admin/resources/article/new"]');
      await form.getByLabel("Title").fill("   ");
      await form.locator('button[type="submit"]').click();
      await expect(page).toHaveURL(/\/admin\/resources\/article\/new$/);
      await expect(page.getByRole("alert")).toContainText("Bu alanları kontrol edin");
      await expect(page.getByText("Bu alan zorunludur.")).toBeVisible();
      await expect(form.getByLabel("Title")).toHaveValue("   ");
      await expectNoA11yViolations(page, "editor validation feedback");
      await form.getByLabel("Title").fill(title);
      await expectNoA11yViolations(page, "editor create form");

      await editorContext.setExtraHTTPHeaders({ ...roleHeaders, "X-Test-Generated-API-Status": "503" });
      const failedSaveResponse = page.waitForResponse((response) =>
        response.request().method() === "POST" && response.url().endsWith("/admin/resources/article/new"),
      );
      await form.locator('button[type="submit"]').click();
      expect((await failedSaveResponse).status()).toBe(502);
      await expect(page.getByRole("alert")).toContainText("Değişiklikleriniz kaydedilemedi");
      await expect(page.getByRole("alert")).toContainText("Girdiğiniz bilgiler burada. Biraz sonra yeniden kaydetmeyi deneyin.");
      await expect(form.getByLabel("Title")).toHaveValue(title);
      await expectNoA11yViolations(page, "editor service-error feedback");
      const failedCreateListing = await request.get(`${externalOrigin}/api/v1/article`, { headers: roleHeaders });
      expect(failedCreateListing.status()).toBe(200);
      expect(JSON.stringify(await failedCreateListing.json())).not.toContain(title);

      await editorContext.setExtraHTTPHeaders(roleHeaders);
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

      await editorContext.setExtraHTTPHeaders({ ...roleHeaders, "X-Test-Generated-API-Read-Status": "503" });
      const failedListResponse = await page.goto(`${externalOrigin}/admin/resources/article?search=${encodeURIComponent(title)}`);
      expect(failedListResponse?.status()).toBe(502);
      await expect(page.getByRole("alert")).toContainText("Sayfa yüklenemedi");
      await expect(page.getByRole("alert")).toContainText("Sayfayı yenileyip tekrar deneyin");
      const retryListForm = page.locator('form[action="/admin/resources/article"].generated-load-retry');
      await expect(retryListForm).toBeVisible();
      await expect(page.locator("body")).not.toContainText("fixture service unavailable");
      await expectNoA11yViolations(page, "editor generated list load-error feedback");
      await editorContext.setExtraHTTPHeaders(roleHeaders);
      await retryListForm.getByRole("button", { name: "Yeniden dene" }).click();
      await expect(page.getByRole("link", { name: title })).toBeVisible();

      const forgedID = `csrf-${randomUUID()}`;
      const forgedCreate = await request.post(`${externalOrigin}/admin/resources/article/new`, {
        headers,
        form: { _csrf: "forged-token", id: forgedID, title: "Must not be created" },
      });
      expect(forgedCreate.status()).toBe(403);
      expect((await request.get(`${externalOrigin}/api/v1/article/${forgedID}`, { headers })).status()).toBe(404);

      const forgedUpdate = await request.post(`${externalOrigin}${recordPath}`, {
        headers,
        form: { _csrf: "forged-token", title: "Must not be updated" },
      });
      expect(forgedUpdate.status()).toBe(403);
      const forgedUpdateReadback = await request.get(`${externalOrigin}${apiPath}`, { headers });
      expect(forgedUpdateReadback.status()).toBe(200);
      expect((await forgedUpdateReadback.json()).title).toBe(title);

      const forgedDelete = await request.post(`${externalOrigin}${recordPath}/delete`, {
        headers,
        form: { _csrf: "forged-token" },
      });
      expect(forgedDelete.status()).toBe(403);
      expect((await request.get(`${externalOrigin}${apiPath}`, { headers })).status()).toBe(200);

      await recordLink.click();
      const editForm = page.locator(`form[action="${recordPath}"]`);
      await editForm.getByLabel("Title").fill(`${title} updated`);
      await editForm.locator('button[type="submit"]').click();
      await expect(page.getByRole("link", { name: `${title} updated` })).toBeVisible();
      const updated = await request.get(`${externalOrigin}${apiPath}`, { headers });
      expect(updated.status()).toBe(200);
      expect((await updated.json()).title).toBe(`${title} updated`);

      const lockedRecordLink = page.getByRole("link", { name: lockedTitle });
      const lockedRecordPath = await lockedRecordLink.getAttribute("href");
      expect(lockedRecordPath).toBe(`/admin/resources/article/${lockedID}`);
      await lockedRecordLink.click();
      await expect(page).toHaveURL(new RegExp(`${lockedID}$`));
      await expect(page.locator(`form[action="${lockedRecordPath}"]`)).toHaveCount(0);
      await expect(page.locator(`form[action="${lockedRecordPath}/delete"]`)).toHaveCount(0);
      await expect(page.getByText(lockedTitle, { exact: true })).toBeVisible();
      await expectNoA11yViolations(page, "editor read-only record detail");
      const adminUpdate = await request.post(`${externalOrigin}${lockedRecordPath}`, {
        headers,
        form: { _csrf: "external-edition-test-token", title: "Must remain protected" },
      });
      expect(adminUpdate.status()).toBe(403);
      const adminDelete = await request.post(`${externalOrigin}${lockedRecordPath}/delete`, {
        headers,
        form: { _csrf: "external-edition-test-token" },
      });
      expect(adminDelete.status()).toBe(403);
      const apiUpdate = await request.put(`${externalOrigin}/api/v1/article/${lockedID}`, {
        headers,
        data: { title: "Must remain protected" },
      });
      expect(apiUpdate.status()).toBe(403);
      const apiDelete = await request.delete(`${externalOrigin}/api/v1/article/${lockedID}`, { headers });
      expect(apiDelete.status()).toBe(403);
      const lockedReadback = await request.get(`${externalOrigin}/api/v1/article/${lockedID}`, { headers: adminHeaders });
      expect(lockedReadback.status()).toBe(200);
      expect((await lockedReadback.json()).title).toBe(lockedTitle);

      const viewerHeaders = { "X-Test-Tenant": tenant, "X-Test-Role": "viewer" };
      const viewerList = await request.get(`${externalOrigin}/admin/resources/article`, { headers: viewerHeaders });
      expect(viewerList.status()).toBe(200);
      const viewerDirectCreate = await request.get(`${externalOrigin}/admin/resources/article/new`, { headers: viewerHeaders });
      expect(viewerDirectCreate.status()).toBe(403);
      const viewerPage = await viewerContext.newPage();
      await viewerPage.addInitScript({ path: path.resolve("node_modules/axe-core/axe.min.js") });
      const viewerDocument = await viewerPage.goto(`${externalOrigin}/admin/resources/article`);
      expect(viewerDocument?.status()).toBe(200);
      await dismissOnboarding(viewerPage);
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
      await expect(viewerPage.getByText(`${title} updated`, { exact: true })).toBeVisible();
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

      await page.goto(`${externalOrigin}/admin/resources/article`);
      await page.getByRole("link", { name: `${title} updated` }).click();
      const retryEditTitle = `${title} edit retry`;
      const retryEditForm = page.locator(`form[action="${recordPath}"]`);
      await retryEditForm.getByLabel("Title").fill(retryEditTitle);
      await editorContext.setExtraHTTPHeaders({ ...roleHeaders, "X-Test-Generated-API-Status": "503" });
      const [failedEditResponse] = await Promise.all([
        page.waitForNavigation(),
        retryEditForm.locator('button[type="submit"]').click(),
      ]);
      expect(failedEditResponse?.status()).toBe(502);
      await expect(page.getByRole("alert")).toContainText("Değişiklikleriniz kaydedilemedi");
      await expect(page.getByRole("alert")).toContainText("Girdiğiniz bilgiler burada. Biraz sonra yeniden kaydetmeyi deneyin.");
      await expect(retryEditForm.getByLabel("Title")).toHaveValue(retryEditTitle);
      const unchangedAfterFailedEdit = await request.get(`${externalOrigin}${apiPath}`, { headers });
      expect(unchangedAfterFailedEdit.status()).toBe(200);
      expect((await unchangedAfterFailedEdit.json()).title).toBe(`${title} updated`);
      await expectNoA11yViolations(page, "editor edit service-error feedback");

      await editorContext.setExtraHTTPHeaders(roleHeaders);
      await retryEditForm.locator('button[type="submit"]').click();
      await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
      await expect(page.getByRole("link", { name: retryEditTitle })).toBeVisible();
      const savedEditRetry = await request.get(`${externalOrigin}${apiPath}`, { headers });
      expect(savedEditRetry.status()).toBe(200);
      expect((await savedEditRetry.json()).title).toBe(retryEditTitle);

      await page.goto(`${externalOrigin}${recordPath}`);
      page.once("dialog", (dialog) => dialog.accept());
      await editorContext.setExtraHTTPHeaders({ ...roleHeaders, "X-Test-Generated-API-Status": "503" });
      const [failedDeleteResponse] = await Promise.all([
        page.waitForNavigation(),
        page.locator(`form[action="${recordPath}/delete"] button[type="submit"]`).click(),
      ]);
      expect(failedDeleteResponse?.status()).toBe(502);
      await expect(page.getByRole("alert")).toContainText("Kayıt silinemedi");
      await expect(page.getByRole("alert")).toContainText("Yeniden denemeden önce kaydın hâlâ listede olup olmadığını kontrol edin.");
      await expect(page.locator(`form[action="${recordPath}/delete"]`)).toBeVisible();
      const unchangedAfterFailedDelete = await request.get(`${externalOrigin}${apiPath}`, { headers });
      expect(unchangedAfterFailedDelete.status()).toBe(200);
      expect((await unchangedAfterFailedDelete.json()).title).toBe(retryEditTitle);
      await expectNoA11yViolations(page, "editor delete service-error feedback");

      await editorContext.setExtraHTTPHeaders(roleHeaders);
      page.once("dialog", (dialog) => dialog.accept());
      await page.locator(`form[action="${recordPath}/delete"] button[type="submit"]`).click();
      await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
      expect((await request.get(`${externalOrigin}${apiPath}`, { headers })).status()).toBe(404);
    } finally {
      await editorContext.close();
      await viewerContext?.close();
    }
  });

  test("editor can retry a failed record load at the same URL", async ({ browser, request }) => {
    const tenant = `load-retry-${randomUUID()}`;
    const editorHeaders = { "X-Test-Tenant": tenant, "X-Test-Role": "editor" };
    const adminHeaders = { "X-Test-Tenant": tenant, "X-Test-Role": "admin" };
    const id = `load-${randomUUID()}`;
    const title = `Retry record ${randomUUID()}`;
    const seeded = await request.post(`${externalOrigin}/api/v1/article`, { headers: adminHeaders, data: { id, title } });
    expect(seeded.status()).toBe(201);

    const context = await browser.newContext({ extraHTTPHeaders: editorHeaders });
    const page = await context.newPage();
    await page.addInitScript({ path: path.resolve("node_modules/axe-core/axe.min.js") });
    const recordPath = `/admin/resources/article/${id}`;
    try {
      const initial = await page.goto(`${externalOrigin}${recordPath}`);
      expect(initial?.status()).toBe(200);
      await dismissOnboarding(page);
      await expect(page.locator(`form[action="${recordPath}"]`).getByLabel("Title")).toHaveValue(title);

      await context.setExtraHTTPHeaders({ ...editorHeaders, "X-Test-Generated-API-Read-Status": "503" });
      const failed = await page.goto(`${externalOrigin}${recordPath}`);
      expect(failed?.status()).toBe(502);
      await expect(page.getByRole("alert")).toContainText("Sayfa yüklenemedi");
      const retry = page.locator(`form[action="${recordPath}"].generated-load-retry`);
      await expect(retry).toBeVisible();
      await expectNoA11yViolations(page, "editor generated record load-error feedback");

      await context.setExtraHTTPHeaders(editorHeaders);
      await retry.getByRole("button", { name: "Yeniden dene" }).click();
      await expect(page.locator(`form[action="${recordPath}"]`).getByLabel("Title")).toHaveValue(title);
    } finally {
      await context.close().catch(() => {});
    }
  });

  test("editor selects tenant media and persists its reference", async ({ browser, request }) => {
    const tenant = `media-e2e-${randomUUID()}`;
    const headers = { "X-Test-Tenant": tenant, "X-Test-Role": "editor" };
    const context = await browser.newContext({ extraHTTPHeaders: headers });
    const page = await context.newPage();
    await page.addInitScript({ path: path.resolve("node_modules/axe-core/axe.min.js") });

    try {
      const response = await page.goto(`${externalOrigin}/admin/resources/media_article/new`);
      expect(response?.status()).toBe(200);
      await dismissOnboarding(page);
      const form = page.locator('form[action="/admin/resources/media_article/new"]');
      await form.getByRole("button", { name: "Kütüphaneden seç" }).click();
      const picker = page.getByRole("dialog", { name: "Medya kütüphanesi" });
      await expect(picker).toBeVisible();
      const item = page.locator(".admin-media-picker-item");
      await expect(item).toHaveCount(1);
      await expectNoA11yViolations(page, "independent Edition media picker");
      const mediaID = await item.getAttribute("data-asset-id");
      expect(mediaID).toMatch(/^fixture-[a-f0-9]{16}$/);
      await item.click();
      await expect(picker).toBeHidden();
      await expect(form.locator('input[name="cover_media_id"]')).toHaveValue(mediaID!);

      const title = `Media article ${randomUUID()}`;
      await form.getByLabel("Title").fill(title);
      await form.locator('button[type="submit"]').click();
      await expect(page.getByRole("link", { name: title })).toBeVisible();
      const recordPath = await page.getByRole("link", { name: title }).getAttribute("href");
      expect(recordPath).toMatch(/^\/admin\/resources\/media_article\//);
      const apiPath = recordPath!.replace("/admin/resources", "/api/v1");
      const persisted = await request.get(`${externalOrigin}${apiPath}`, { headers });
      expect(persisted.status()).toBe(200);
      expect((await persisted.json()).cover_media_id).toBe(mediaID);

      const crossTenant = await request.post(`${externalOrigin}/api/v1/media_article`, {
        headers: { "X-Test-Tenant": `${tenant}-other`, "X-Test-Role": "editor" },
        data: { id: `cross-${randomUUID()}`, title: "Cross tenant reference", cover_media_id: mediaID },
      });
      expect(crossTenant.status()).toBe(422);
      expect((await request.get(`${externalOrigin}${apiPath}`, { headers })).status()).toBe(200);
    } finally {
      await context.close();
    }
  });
});

async function expectNoA11yViolations(page: import("@playwright/test").Page, context: string): Promise<void> {
  const result = await page.evaluate(async () => (window as any).axe.run(document, {
    runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
  }));
  expect(result.violations.map((violation: any) => violation.id), `${context} WCAG 2.1 A/AA violations`).toEqual([]);
}

async function dismissOnboarding(page: import("@playwright/test").Page): Promise<void> {
  const onboarding = page.locator("#admin-onboarding-dialog");
  await expect(onboarding).toBeVisible();
  await onboarding.getByRole("button", { name: "Bir daha gösterme" }).click();
  await expect(onboarding).toBeHidden();
}
