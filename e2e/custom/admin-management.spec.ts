import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import type { ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { runCommand, spawnServer, stopServer } from "./server-process";

const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
const backupSigningSeed = "ab".repeat(32);
let server: ChildProcess | undefined;
let serverArgs: string[] = [];
let serverEnv: Record<string, string | undefined> = {};
let tempDir = "";
let baseURL = "";
let viewerStorageState: Awaited<ReturnType<BrowserContext["storageState"]>> | undefined;
let adminStorageState: Awaited<ReturnType<BrowserContext["storageState"]>> | undefined;
let tokenLifetimeBaseline: { passwordReset: string; invitation: string } | undefined;
let restrictedMutationTargets: string[] = [];

test.describe("single Edition administrator management", () => {
  test.describe.configure({ mode: "serial" });
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-admin-management-"));
    const dialect = process.env.HYDRO_E2E_DATABASE_DIALECT ?? "sqlite";
    const database = process.env.HYDRO_E2E_DATABASE_DSN ?? path.join(tempDir, "edition.sqlite");
    if (dialect === "postgres" && !process.env.HYDRO_E2E_DATABASE_DSN) {
      throw new Error("HYDRO_E2E_DATABASE_DSN is required for the PostgreSQL Admin management run");
    }
    if (dialect === "postgres" && !process.env.HYDRO_PG_PRIVILEGED_ROLE) {
      throw new Error("HYDRO_PG_PRIVILEGED_ROLE must name the PostgreSQL backup operator role for the Admin management run");
    }
    runCommand("go", ["run", "./cmd/hydro", "migrate", "--database-dialect", dialect, "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
    runCommand("go", ["run", "./cmd/hydro", "create-admin", "--email", "admin@example.test", "--password-stdin", "--database-dialect", dialect, "--database-dsn", database], { cwd: repoRoot, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    runCommand("go", ["run", "./tests/e2e/support/seedroles", "--database-dialect", dialect, "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    serverArgs = ["run", "./cmd/hydro", "serve", "--dev", "--database-dialect", dialect, "--database-dsn", database, "--address", `127.0.0.1:${port}`];
    serverEnv = { ...process.env, HYDRO_PUBLIC_ORIGIN: baseURL, HYDRO_MEDIA_DIR: path.join(tempDir, "media"), HYDRO_BACKUP_SIGNING_SEED: backupSigningSeed, HYDRO_PREVIEW_SIGNING_KEY: "cd".repeat(32) };
    await startAdminServer();
  }, 60_000);

  test.afterEach(async ({}, testInfo) => {
    if (testInfo.status !== "passed") return;
    // Reset the process-local IP login budget between independent browser journeys.
    // Keep the same DB so users, permissions, and sessions remain available.
    await stopServer(server);
    await startAdminServer();
  });

  test.afterAll(async () => {
    await stopServer(server);
    if (tempDir) await rm(tempDir, { recursive: true, force: true });
  });

  test("editor and viewer sessions enforce the Edition role boundary", async ({ browser }) => {
    test.setTimeout(120_000);
    const expectAxeClean = async (page: Page, label: string) => {
      const result = await page.evaluate(async () => (window as any).axe.run(document, {
        runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
      }));
      expect(result.violations.map((violation: any) => ({
        id: violation.id,
        impact: violation.impact,
        nodes: violation.nodes.map((node: any) => ({ target: node.target, summary: node.failureSummary })),
      })), `${label} WCAG 2.1 A/AA violations`).toEqual([]);
    };
    let createdPageID = "";
    let createdNewsID = "";
    const adminContext = await browser.newContext();
    const adminPage = await adminContext.newPage();
    await adminPage.goto(`${baseURL}/login`);
    await adminPage.getByLabel("Email address").fill("admin@example.test");
    await adminPage.getByLabel("Password").fill(adminPassword);
    await adminPage.getByRole("button", { name: "Sign in" }).click();
    await expect(adminPage).toHaveURL(/\/admin(?:\?.*)?$/);
    expect((await adminPage.goto(`${baseURL}/admin/security/token-lifetimes`))?.status()).toBe(200);
    tokenLifetimeBaseline = {
      passwordReset: await adminPage.locator('select[name="password_reset_ttl"]').inputValue(),
      invitation: await adminPage.locator('select[name="invitation_ttl"]').inputValue(),
    };
    expect(tokenLifetimeBaseline.passwordReset).not.toBe("");
    expect(tokenLifetimeBaseline.invitation).not.toBe("");
    const adminNavPaths = await adminPage.locator('.studio-sidebar a[href^="/admin"]').evaluateAll((links) =>
      [...new Set(links.map((link) => new URL((link as HTMLAnchorElement).href).pathname))],
    );
    expect(adminNavPaths.length).toBeGreaterThan(0);
    for (const path of adminNavPaths) {
      const response = await adminPage.goto(`${baseURL}${path}`);
      expect(response?.status(), `admin navigation link ${path} must open`).toBe(200);
    }
    restrictedMutationTargets = [];
    for (const path of [
      "/admin/users",
      "/admin/permissions",
      "/admin/security",
      "/admin/security/token-lifetimes",
      "/admin/audit",
      "/admin/backup",
      "/admin/control",
      "/admin/operations",
      "/admin/developer",
      "/admin/developer/resources",
    ]) {
      const response = await adminPage.goto(`${baseURL}${path}`);
      expect(response?.status(), `administrator mutation inventory page ${path} must open`).toBe(200);
      const actions = await adminPage.locator('form[method="post"]').evaluateAll((forms) =>
        forms.flatMap((form) => {
          const element = form as HTMLFormElement;
          const buttons = [...element.querySelectorAll<HTMLButtonElement | HTMLInputElement>("button[formaction],input[formaction]")];
          return [element.action || location.href, ...buttons.map((button) => button.formAction)]
            .filter((action) => action && new URL(action, location.href).origin === location.origin);
        }),
      );
      restrictedMutationTargets.push(...actions);
    }
    restrictedMutationTargets = [...new Set(restrictedMutationTargets)];
    expect(restrictedMutationTargets.length, "the privileged Admin pages must expose mutation forms").toBeGreaterThan(0);
    const adminCSRFResponse = await adminPage.request.get(`${baseURL}/csrf-token`);
    expect(adminCSRFResponse.status()).toBe(200);
    const { token: adminCSRFToken } = await adminCSRFResponse.json();
    expect((await adminContext.cookies(baseURL)).find((cookie) => cookie.name === "hydro_csrf")?.value).toBe(adminCSRFToken);
    for (const target of restrictedMutationTargets) {
      const route = new URL(target).pathname;
      const missingCSRF = await adminPage.request.post(target, { form: {} });
      expect(missingCSRF.status(), `Admin mutation ${route} must reject missing CSRF`).toBe(403);
      const mismatchedCSRF = await adminPage.request.post(target, { form: { _csrf: "deliberately-mismatched-csrf-token" } });
      expect(mismatchedCSRF.status(), `Admin mutation ${route} must reject mismatched CSRF`).toBe(403);
    }
    const adminNewsCSRF = { "X-CSRF-Token": adminCSRFToken };
    const adminNewsInput = {
      locale: "tr",
      slug: "role-admin-news-api",
      title: "Admin API created news",
      body: "This content is persisted through the News API.",
    };
    const deniedAdminNewsMissingCSRF = await adminPage.request.post(`${baseURL}/api/news`, { data: adminNewsInput });
    expect(deniedAdminNewsMissingCSRF.status(), "Admin News API create rejects missing CSRF").toBe(403);
    const deniedAdminNewsMismatchedCSRF = await adminPage.request.post(`${baseURL}/api/news`, {
      headers: { "X-CSRF-Token": "deliberately-mismatched-csrf-token" },
      data: adminNewsInput,
    });
    expect(deniedAdminNewsMismatchedCSRF.status(), "Admin News API create rejects mismatched CSRF").toBe(403);
    const createdAdminNewsResponse = await adminPage.request.post(`${baseURL}/api/news`, {
      headers: adminNewsCSRF,
      data: adminNewsInput,
    });
    expect(createdAdminNewsResponse.status(), "Admin with valid CSRF can create through the News API").toBe(200);
    const createdAdminNews = await createdAdminNewsResponse.json();
    expect(createdAdminNews.status).toBe("draft");
    expect(createdAdminNews.version).toBe(1);
    const updatedAdminNewsResponse = await adminPage.request.put(`${baseURL}/api/news/${createdAdminNews.id}`, {
      headers: adminNewsCSRF,
      data: { ...adminNewsInput, title: "Admin API updated news", version: createdAdminNews.version },
    });
    expect(updatedAdminNewsResponse.status(), "Admin with valid CSRF can update through the News API").toBe(200);
    const updatedAdminNews = await updatedAdminNewsResponse.json();
    expect(updatedAdminNews.title).toBe("Admin API updated news");
    expect(updatedAdminNews.version).toBe(2);
    const adminNewsReadback = await adminPage.request.get(`${baseURL}/api/news/${createdAdminNews.id}`);
    expect(adminNewsReadback.status()).toBe(200);
    expect(await adminNewsReadback.json()).toMatchObject({
      id: createdAdminNews.id,
      title: "Admin API updated news",
      status: "draft",
      version: 2,
    });
    const scheduledAdminNewsResponse = await adminPage.request.post(`${baseURL}/api/news/${createdAdminNews.id}/schedule`, {
      headers: adminNewsCSRF,
      data: { version: updatedAdminNews.version, scheduled_at: "2030-01-01T12:00:00Z" },
    });
    expect(scheduledAdminNewsResponse.status(), "Admin with valid CSRF can schedule through the News API").toBe(200);
    const scheduledAdminNews = await scheduledAdminNewsResponse.json();
    expect(scheduledAdminNews).toMatchObject({ status: "scheduled", version: 3 });
    const publishedAdminNewsResponse = await adminPage.request.post(
      `${baseURL}/api/news/${createdAdminNews.id}/publish?version=${scheduledAdminNews.version}`,
      { headers: adminNewsCSRF },
    );
    expect(publishedAdminNewsResponse.status(), "Admin with valid CSRF can publish through the News API").toBe(200);
    const publishedAdminNews = await publishedAdminNewsResponse.json();
    expect(publishedAdminNews).toMatchObject({ status: "published", version: 4 });
    const retractedAdminNewsResponse = await adminPage.request.post(
      `${baseURL}/api/news/${createdAdminNews.id}/retract?version=${publishedAdminNews.version}`,
      { headers: adminNewsCSRF },
    );
    expect(retractedAdminNewsResponse.status(), "Admin with valid CSRF can retract through the News API").toBe(200);
    const retractedAdminNews = await retractedAdminNewsResponse.json();
    expect(retractedAdminNews).toMatchObject({ status: "retracted", version: 5 });
    const workflowReadback = await adminPage.request.get(`${baseURL}/api/news/${createdAdminNews.id}`);
    expect(workflowReadback.status()).toBe(200);
    expect(await workflowReadback.json()).toMatchObject({
      id: createdAdminNews.id,
      title: "Admin API updated news",
      status: "retracted",
      version: 5,
    });
    const adminTaxonomyURL = `${baseURL}/admin/news/taxonomy`;
    expect((await adminPage.goto(adminTaxonomyURL))?.status(), "Admin can open the News taxonomy manager").toBe(200);
    const taxonomyMissingCSRF = await adminPage.request.post(adminTaxonomyURL, {
      form: { id: "role-admin-api-term", slug: "role-admin-api-term", kind: "tag", labels_json: '{"tr":"Yönetici etiketi"}' },
    });
    expect(taxonomyMissingCSRF.status(), "News taxonomy create rejects missing CSRF").toBe(403);
    const taxonomyMismatchedCSRF = await adminPage.request.post(adminTaxonomyURL, {
      form: {
        _csrf: "deliberately-mismatched-csrf-token",
        id: "role-admin-api-term",
        slug: "role-admin-api-term",
        kind: "tag",
        labels_json: '{"tr":"Yönetici etiketi"}',
      },
    });
    expect(taxonomyMismatchedCSRF.status(), "News taxonomy create rejects mismatched CSRF").toBe(403);
    const taxonomyCreate = await adminPage.request.post(adminTaxonomyURL, {
      maxRedirects: 0,
      form: {
        _csrf: adminCSRFToken,
        id: "role-admin-api-term",
        slug: "role-admin-api-term",
        kind: "tag",
        labels_json: '{"tr":"Yönetici etiketi"}',
      },
    });
    expect(taxonomyCreate.status(), "Admin with valid CSRF can create a taxonomy term").toBe(303);
    expect((await adminPage.goto(adminTaxonomyURL))?.status()).toBe(200);
    await expect(adminPage.locator("table tbody")).toContainText("role-admin-api-term");
    const taxonomyUpdate = await adminPage.request.post(adminTaxonomyURL, {
      maxRedirects: 0,
      form: {
        _csrf: adminCSRFToken,
        id: "role-admin-api-term",
        slug: "role-admin-api-term-updated",
        kind: "category",
        labels_json: '{"tr":"Yönetici kategorisi"}',
      },
    });
    expect(taxonomyUpdate.status(), "Admin with valid CSRF can update a taxonomy term").toBe(303);
    expect((await adminPage.goto(adminTaxonomyURL))?.status()).toBe(200);
    await expect(adminPage.locator("table tbody")).toContainText("role-admin-api-term-updated");
    const taxonomyDelete = await adminPage.request.post(`${adminTaxonomyURL}/role-admin-api-term/delete`, {
      maxRedirects: 0,
      form: { _csrf: adminCSRFToken },
    });
    expect(taxonomyDelete.status(), "Admin with valid CSRF can delete a taxonomy term").toBe(303);
    expect((await adminPage.goto(adminTaxonomyURL))?.status()).toBe(200);
    await expect(adminPage.locator("table tbody")).not.toContainText("role-admin-api-term");
    await adminPage.goto(`${baseURL}/admin/users`);
    const managementTargets: Record<string, { id: string; role: string; permissions: string; disable: string; currentRole: string; currentContentWrite: string }> = {};
    for (const email of ["editor@example.test", "viewer@example.test"]) {
      const row = adminPage.locator(".studio-table tbody tr").filter({ has: adminPage.getByText(email, { exact: true }) });
      const roleAction = await row.locator('form[action$="/role"]').getAttribute("action");
      const permissionAction = await row.locator('form[action$="/permissions"]').getAttribute("action");
      const disableAction = await row.locator('form[action$="/disable"]').getAttribute("action");
      const currentRole = await row.locator('select[name="role"]').inputValue();
      const currentContentWrite = await row.locator('select[name="perm_admin.content.write"]').inputValue();
      expect(roleAction).toBeTruthy();
      expect(permissionAction).toBeTruthy();
      expect(disableAction).toBeTruthy();
      const id = new URL(roleAction!, baseURL).pathname.split("/").at(-2)!;
      expect(new URL(permissionAction!, baseURL).pathname.split("/").at(-2)).toBe(id);
      expect(new URL(disableAction!, baseURL).pathname.split("/").at(-2)).toBe(id);
      managementTargets[email] = { id, role: roleAction!, permissions: permissionAction!, disable: disableAction!, currentRole, currentContentWrite };
    }
    adminStorageState = await adminContext.storageState();
    await adminContext.close();

    for (const account of [
      { email: "editor@example.test", expectedRole: "editor", allowedPath: "/admin/pages/new" },
      { email: "viewer@example.test", expectedRole: "viewer", allowedPath: "/admin/pages/new" },
    ]) {
      const context = await browser.newContext();
      const page = await context.newPage();
      await page.goto(`${baseURL}/login`);
      await page.getByLabel("Email address").fill(account.email);
      await page.getByLabel("Password").fill(adminPassword);
      await page.getByRole("button", { name: "Sign in" }).click();
      await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
      if (account.expectedRole === "viewer") viewerStorageState = await context.storageState();

      const visibleNavPaths = await page.locator('.studio-sidebar a[href^="/admin"]').evaluateAll((links) =>
        [...new Set(links.map((link) => new URL((link as HTMLAnchorElement).href).pathname))],
      );
      for (const path of visibleNavPaths) {
        const response = await page.goto(`${baseURL}${path}`);
        expect(response?.status(), `${account.expectedRole} navigation link ${path} must open`).toBe(200);
      }
      const visibleNavPathSet = new Set(visibleNavPaths);
      for (const path of adminNavPaths.filter((candidate) => !visibleNavPathSet.has(candidate))) {
        const response = await page.goto(`${baseURL}${path}`);
        expect(response?.status(), `${account.expectedRole} must be denied hidden Admin navigation route ${path}`).toBe(403);
      }

      const allowed = await page.goto(`${baseURL}${account.allowedPath}`);
      const accountSettings = await page.goto(`${baseURL}/admin/account`);
      expect(accountSettings?.status()).toBe(200);
      await expect(page.locator(`body[data-studio-role="${account.expectedRole}"]`)).toBeVisible();
      if (account.expectedRole === "editor") {
        expect(allowed?.status()).toBe(200);
        await expect(page.locator(`body[data-studio-role="${account.expectedRole}"]`)).toBeVisible();
        await expect(page.locator('.studio-sidebar a[href="/admin/pages"]')).toBeVisible();
        await page.goto(`${baseURL}/admin/pages`);
        await expect(page.locator('.studio-heading a[href="/admin/pages/new"]')).toBeVisible();
        await page.locator('.studio-search').click();
        await expect(page.locator('.studio-search-dialog .search-result-item').filter({ hasText: "Yeni Sayfa Ekle" })).toBeVisible();
        await expect(page.locator('.studio-search-dialog .search-result-item').filter({ hasText: "Yeni Haber Yaz" })).toBeVisible();
        await expect(page.locator('.studio-search-dialog .search-result-item').filter({ hasText: "Medya Yükle" })).toBeVisible();
        for (const hiddenCommand of ["Yeni Ürün Ekle", "Siparişler", "Üye Yönetimi", "Raporlar ve Analitik", "Yönetim Araçları", "Operasyon Durumu", "Yedek İndir", "Güvenlik & Yetkiler"]) {
          await expect(page.locator('.studio-search-dialog .search-result-item').filter({ hasText: hiddenCommand }), `editor command ${hiddenCommand} must require its capability`).toHaveCount(0);
        }
        expect((await page.goto(`${baseURL}/admin/media`))?.status()).toBe(200);
        const importPage = await page.goto(`${baseURL}/admin/pages/import`);
        expect(importPage?.status(), "an Editor with content.write can open the import route directly").toBe(200);
        await expect(page.locator('form[action="/admin/pages/import"]')).toBeVisible();
        const importCSV = Buffer.from("title,slug,body\nImported editor page,role-editor-imported-page,Imported through the authenticated page import wizard.\nDuplicate row,role-editor-imported-page,This row must be skipped.\nInvalid row,,This row has no slug.\n");
        const importForm = page.locator('form[action="/admin/pages/import"]');
        await importForm.locator('input[name="file"]').setInputFiles({ name: "pages.csv", mimeType: "text/csv", buffer: importCSV });
        await importForm.getByRole("button", { name: "Sütunları incele" }).click();
        await expect(page.locator('[data-import-wizard]')).toBeVisible();
        await page.locator('[name="map_title"]').selectOption("0");
        await page.locator('[name="map_slug"]').selectOption("1");
        await page.locator('[name="map_body"]').selectOption("2");
        const importPreviewForm = page.locator('form[action="/admin/pages/import"]');
        await importPreviewForm.locator('input[name="file"]').setInputFiles({ name: "pages.csv", mimeType: "text/csv", buffer: importCSV });
        await importPreviewForm.getByRole("button", { name: "Aktarımı başlat" }).click();
        await expect(page.locator('[data-import-preview]')).toContainText("1 yeni taslak aktarılabilir. 1 yinelenen satır ve 1 geçersiz satır atlanacak.");
        const importConfirmForm = page.locator('form[action="/admin/pages/import"]');
        await importConfirmForm.locator('input[name="file"]').setInputFiles({ name: "pages.csv", mimeType: "text/csv", buffer: importCSV });
        await importConfirmForm.getByRole("button", { name: "Taslakları aktar" }).click();
        await expect(page.getByRole("status")).toContainText("1 taslak eklendi. 2 satır atlandı.");
        await page.goto(`${baseURL}/admin/pages`);
        await expect(page.locator('.studio-table')).toContainText("Imported editor page");
        const importedPageRow = page.locator('.studio-table tbody tr').filter({ hasText: "Imported editor page" });
        const importedPageLink = importedPageRow.locator('a[href^="/admin/pages/"]');
        await expect(importedPageLink).toBeVisible();
        const importedPageID = new URL(await importedPageLink.getAttribute("href") ?? "", baseURL).pathname.split("/").at(-1)!;
        const importedAPI = await page.request.get(`${baseURL}/api/pages/${importedPageID}`);
        expect(importedAPI.status(), "an Editor import must persist and be readable through the Edition API").toBe(200);
        expect(await importedAPI.text()).toContain("role-editor-imported-page");
        await page.goto(`${baseURL}/admin/pages/import`);
        const repeatedImportForm = page.locator('form[action="/admin/pages/import"]');
        await repeatedImportForm.locator('input[name="file"]').setInputFiles({ name: "pages.csv", mimeType: "text/csv", buffer: importCSV });
        await repeatedImportForm.getByRole("button", { name: "Sütunları incele" }).click();
        await page.locator('[name="map_title"]').selectOption("0");
        await page.locator('[name="map_slug"]').selectOption("1");
        await page.locator('[name="map_body"]').selectOption("2");
        const repeatedPreviewForm = page.locator('form[action="/admin/pages/import"]');
        await repeatedPreviewForm.locator('input[name="file"]').setInputFiles({ name: "pages.csv", mimeType: "text/csv", buffer: importCSV });
        await repeatedPreviewForm.getByRole("button", { name: "Aktarımı başlat" }).click();
        await expect(page.locator('[data-import-preview]')).toContainText("0 yeni taslak aktarılabilir. 2 yinelenen satır ve 1 geçersiz satır atlanacak.");
        const repeatedConfirmForm = page.locator('form[action="/admin/pages/import"]');
        await repeatedConfirmForm.locator('input[name="file"]').setInputFiles({ name: "pages.csv", mimeType: "text/csv", buffer: importCSV });
        await repeatedConfirmForm.getByRole("button", { name: "Taslakları aktar" }).click();
        await expect(page.getByRole("status")).toContainText("0 taslak eklendi. 3 satır atlandı.");
        await page.goto(`${baseURL}/admin/pages`);
        await expect(page.locator('.studio-table tbody tr').filter({ hasText: "Imported editor page" })).toHaveCount(1);
        const xlsxPath = path.join(repoRoot, "e2e", "custom", "fixtures", "pages-import.xlsx");
        await page.goto(`${baseURL}/admin/pages/import`);
        const xlsxImportForm = page.locator('form[action="/admin/pages/import"]');
        await xlsxImportForm.locator('input[name="file"]').setInputFiles(xlsxPath);
        await xlsxImportForm.getByRole("button", { name: "Sütunları incele" }).click();
        await page.locator('[name="map_title"]').selectOption("0");
        await page.locator('[name="map_slug"]').selectOption("1");
        await page.locator('[name="map_body"]').selectOption("2");
        const xlsxPreviewForm = page.locator('form[action="/admin/pages/import"]');
        await xlsxPreviewForm.locator('input[name="file"]').setInputFiles(xlsxPath);
        await xlsxPreviewForm.getByRole("button", { name: "Aktarımı başlat" }).click();
        await expect(page.locator('[data-import-preview]')).toContainText("1 yeni taslak aktarılabilir. 0 yinelenen satır ve 0 geçersiz satır atlanacak.");
        const xlsxConfirmForm = page.locator('form[action="/admin/pages/import"]');
        await xlsxConfirmForm.locator('input[name="file"]').setInputFiles(xlsxPath);
        await xlsxConfirmForm.getByRole("button", { name: "Taslakları aktar" }).click();
        await expect(page.getByRole("status")).toContainText("1 taslak eklendi. 0 satır atlandı.");
        await page.goto(`${baseURL}/admin/pages`);
        const xlsxImportedRow = page.locator('.studio-table tbody tr').filter({ hasText: "XLSX editor page" });
        await expect(xlsxImportedRow).toHaveCount(1);
        const xlsxImportedLink = xlsxImportedRow.locator('a[href^="/admin/pages/"]');
        const xlsxImportedID = new URL(await xlsxImportedLink.getAttribute("href") ?? "", baseURL).pathname.split("/").at(-1)!;
        const xlsxImportedAPI = await page.request.get(`${baseURL}/api/pages/${xlsxImportedID}`);
        expect(xlsxImportedAPI.status(), "an Editor XLSX import must persist and be readable through the Edition API").toBe(200);
        expect(await xlsxImportedAPI.text()).toContain("role-editor-xlsx-page");
        const createForm = page.locator('form[action="/admin/pages"]');
        await page.goto(`${baseURL}${account.allowedPath}`);
        await createForm.locator('[name="slug"]').fill("role-editor-page");
        await createForm.locator('[name="title"]').fill("Editor session page");
        await createForm.locator(".richtext-editor [contenteditable=true]").fill("Created by an authenticated Edition editor.");
        await createForm.getByRole("button", { name: "Taslak oluştur" }).click();
        await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);
        createdPageID = new URL(page.url()).pathname.split("/").at(-1) || "";
        await page.goto(`${baseURL}/admin/news/new`);
        await page.locator('#news-form [name="title"]').fill("Editor role news draft");
        await page.locator('#news-form [name="slug"]').fill("role-editor-news-draft");
        await page.locator('#news-form .richtext-editor [contenteditable="true"]').fill("A draft for the Viewer read-only route check.");
        await page.getByRole("button", { name: "Taslağı kaydet" }).click();
        await expect(page).toHaveURL(/\/admin\/news\/[^/?]+$/);
        createdNewsID = new URL(page.url()).pathname.split("/").at(-1) || "";
        const newsTaxonomyURL = `${baseURL}/admin/news/taxonomy`;
        expect((await page.goto(newsTaxonomyURL))?.status(), "Editor content.write permits News taxonomy management").toBe(200);
        const editorNewsCSRFResponse = await page.request.get(`${baseURL}/csrf-token`);
        expect(editorNewsCSRFResponse.status()).toBe(200);
        const { token: editorNewsCSRF } = await editorNewsCSRFResponse.json();
        expect((await context.cookies(baseURL)).find((cookie) => cookie.name === "hydro_csrf")?.value).toBe(editorNewsCSRF);
        const editorTaxonomyCreate = await page.request.post(newsTaxonomyURL, {
          maxRedirects: 0,
          form: {
            _csrf: editorNewsCSRF,
            id: "role-editor-news-taxonomy",
            slug: "role-editor-news-tag",
            kind: "tag",
            labels_json: '{"tr":"Editör etiketi"}',
          },
        });
        expect(editorTaxonomyCreate.status(), "Editor with valid CSRF can create a taxonomy term").toBe(303);
        expect((await page.goto(newsTaxonomyURL))?.status()).toBe(200);
        await expect(page.locator("table tbody")).toContainText("role-editor-news-tag");
        const editorTaxonomyUpdate = await page.request.post(newsTaxonomyURL, {
          maxRedirects: 0,
          form: {
            _csrf: editorNewsCSRF,
            id: "role-editor-news-taxonomy",
            slug: "role-editor-news-category",
            kind: "category",
            labels_json: '{"tr":"Editör kategorisi"}',
          },
        });
        expect(editorTaxonomyUpdate.status(), "Editor with valid CSRF can update a taxonomy term").toBe(303);
        expect((await page.goto(newsTaxonomyURL))?.status()).toBe(200);
        await expect(page.locator("table tbody")).toContainText("role-editor-news-category");
        const editorTaxonomyDelete = await page.request.post(`${newsTaxonomyURL}/role-editor-news-taxonomy/delete`, {
          maxRedirects: 0,
          form: { _csrf: editorNewsCSRF },
        });
        expect(editorTaxonomyDelete.status(), "Editor with valid CSRF can delete a taxonomy term").toBe(303);
        const protectedTaxonomyCreate = await page.request.post(newsTaxonomyURL, {
          maxRedirects: 0,
          form: {
            _csrf: editorNewsCSRF,
            id: "role-news-taxonomy-protected",
            slug: "role-news-taxonomy-original",
            kind: "tag",
            labels_json: '{"tr":"Korunan etiket"}',
          },
        });
        expect(protectedTaxonomyCreate.status()).toBe(303);
      } else {
        expect(allowed?.status()).toBe(403);
        expect((await page.goto(`${baseURL}/admin/media`))?.status()).toBe(403);
        expect(createdPageID).not.toBe("");
        expect((await page.goto(`${baseURL}/admin/pages`))?.status()).toBe(200);
        await expect(page.locator('.studio-sidebar a[href="/admin/pages"]')).toBeVisible();
        await expect(page.locator('.studio-sidebar a[href="/admin/news"]')).toBeVisible();
        await expect(page.locator('.studio-sidebar a[href="/admin/users"]')).toHaveCount(0);
        await expect(page.locator('.studio-sidebar a[href="/admin/reports"]')).toHaveCount(0);
        await expect(page.locator('.studio-sidebar a[href="/admin/media"]')).toHaveCount(0);
        await expect(page.locator('.studio-heading a[href="/admin/pages/new"]')).toHaveCount(0);
        await page.locator('.studio-search').click();
        await expect(page.locator('.studio-search-dialog .search-result-item').filter({ hasText: "Yeni Sayfa Ekle" })).toHaveCount(0);
        await expect(page.locator('.studio-search-dialog .search-result-item').filter({ hasText: "Yeni Haber Yaz" })).toHaveCount(0);
        for (const hiddenCommand of ["Yeni Ürün Ekle", "Siparişler", "Medya Yükle", "Üye Yönetimi", "Raporlar ve Analitik", "Yönetim Araçları", "Operasyon Durumu", "Yedek İndir", "Güvenlik & Yetkiler"]) {
          await expect(page.locator('.studio-search-dialog .search-result-item').filter({ hasText: hiddenCommand }), `viewer command ${hiddenCommand} must require its capability`).toHaveCount(0);
        }
        await expect(page.locator('a[href="/admin/pages/import"]')).toHaveCount(0);
        expect((await page.request.get(`${baseURL}/admin/pages/import`)).status(), "Viewer must be denied the hidden import deep link").toBe(403);
        await expect(page.locator(`a[href="/admin/pages/${createdPageID}"]`)).toBeVisible();
        expect((await page.goto(`${baseURL}/admin/pages/${createdPageID}`))?.status()).toBe(200);
        await expect(page.locator('form[action^="/admin/pages/"]')).toHaveCount(0);
        await expect(page.locator('iframe.readonly-content-preview')).toBeVisible();
        expect((await page.request.get(`${baseURL}/api/pages/${createdPageID}`)).status()).toBe(200);
        expect((await page.goto(`${baseURL}/admin/news`))?.status(), "Viewer content.read grants access to the news list").toBe(200);
        await expect(page.locator('.studio-sidebar a[href="/admin/news"]')).toBeVisible();
        await expect(page.locator('.studio-heading a[href="/admin/news/new"]')).toHaveCount(0);
        const newsCSRFResponse = await page.request.get(`${baseURL}/csrf-token`);
        expect(newsCSRFResponse.status()).toBe(200);
        const { token: newsCSRFToken } = await newsCSRFResponse.json();
        expect((await context.cookies(baseURL)).find((cookie) => cookie.name === "hydro_csrf")?.value).toBe(newsCSRFToken);
        const viewerTaxonomyURL = `${baseURL}/admin/news/taxonomy`;
        expect((await page.request.get(viewerTaxonomyURL)).status(), "Viewer must be denied the News taxonomy management deep link").toBe(403);
        const viewerTaxonomyUpdate = await page.request.post(viewerTaxonomyURL, {
          maxRedirects: 0,
          form: {
            _csrf: newsCSRFToken,
            id: "role-news-taxonomy-protected",
            slug: "role-news-taxonomy-forged",
            kind: "category",
            labels_json: '{"tr":"Sahte güncelleme"}',
          },
        });
        expect(viewerTaxonomyUpdate.status(), "Viewer cannot update taxonomy with valid CSRF").toBe(403);
        const viewerTaxonomyDelete = await page.request.post(`${viewerTaxonomyURL}/role-news-taxonomy-protected/delete`, {
          maxRedirects: 0,
          form: { _csrf: newsCSRFToken },
        });
        expect(viewerTaxonomyDelete.status(), "Viewer cannot delete taxonomy with valid CSRF").toBe(403);
        const viewerNewsCreate = await page.request.post(`${baseURL}/admin/news`, {
          form: { _csrf: newsCSRFToken, slug: "role-viewer-news", title: "Viewer cannot create news", body: "Denied by content.write." },
        });
        expect(viewerNewsCreate.status(), "Viewer content.read does not grant content.write").toBe(403);
        await page.goto(`${baseURL}/admin/news`);
        await expect(page.locator('.studio-table')).not.toContainText("Viewer cannot create news");
        expect(createdNewsID).not.toBe("");
        expect((await page.goto(`${baseURL}/admin/news/${createdNewsID}`))?.status(), "Viewer content.read grants read-only news detail access").toBe(200);
        await expect(page.locator("#news-title-readonly")).toHaveValue("Editor role news draft");
        await expect(page.locator(".readonly-content-preview")).toContainText("A draft for the Viewer read-only route check.");
        await expect(page.locator('form[action^="/admin/news/"]')).toHaveCount(0);
        const a11yContext = await browser.newContext({ storageState: await context.storageState() });
        try {
          const a11yPage = await a11yContext.newPage();
          await a11yPage.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
          expect((await a11yPage.goto(`${baseURL}/admin/news/${createdNewsID}`))?.status()).toBe(200);
          const viewerNewsA11y = await a11yPage.evaluate(async () => (window as any).axe.run(document, {
            runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
          }));
          expect(viewerNewsA11y.violations.map((violation: any) => ({
            id: violation.id,
            impact: violation.impact,
            nodes: violation.nodes.map((node: any) => ({ target: node.target, summary: node.failureSummary })),
          })), "Viewer read-only News detail WCAG 2.1 A/AA violations").toEqual([]);
        } finally {
          await a11yContext.close();
        }
        const viewerNewsUpdate = await page.request.post(`${baseURL}/admin/news/${createdNewsID}`, {
          form: { _csrf: newsCSRFToken, version: "1", slug: "role-editor-news-draft", title: "Viewer altered news", body: "This must be denied." },
        });
        expect(viewerNewsUpdate.status(), "Viewer content.read does not grant news updates").toBe(403);
        for (const attempt of [
          { action: "schedule", form: { version: "1", scheduled_at: "2030-01-01T12:00" } },
          { action: "publish", form: { version: "1" } },
          { action: "retract", form: { version: "1" } },
        ]) {
          const denied = await page.request.post(`${baseURL}/admin/news/${createdNewsID}/${attempt.action}`, {
            form: { _csrf: newsCSRFToken, ...attempt.form },
          });
          expect(denied.status(), `Viewer cannot ${attempt.action} an Editor news draft`).toBe(403);
        }
        const apiCSRFHeaders = { "X-CSRF-Token": newsCSRFToken };
        const deniedNewsAPIMutations = [
          {
            name: "create",
            request: () => page.request.post(`${baseURL}/api/news`, {
              headers: apiCSRFHeaders,
              data: { locale: "tr", slug: "role-viewer-news-api", title: "Viewer API create denied", body: "Must not persist." },
            }),
          },
          {
            name: "update",
            request: () => page.request.put(`${baseURL}/api/news/${createdNewsID}`, {
              headers: apiCSRFHeaders,
              data: { title: "Viewer API update denied", body: "Must not persist." },
            }),
          },
          ...(["schedule", "publish", "retract"] as const).map((action) => ({
            name: action,
            request: () => page.request.post(`${baseURL}/api/news/${createdNewsID}/${action}`, {
              headers: apiCSRFHeaders,
              data: { version: 1, scheduled_at: "2030-01-01T12:00" },
            }),
          })),
        ];
        for (const attempt of deniedNewsAPIMutations) {
          expect((await attempt.request()).status(), `Viewer cannot ${attempt.name} through the News API`).toBe(403);
        }
        expect((await page.goto(`${baseURL}/admin/news`))?.status()).toBe(200);
        await expect(page.locator(".studio-table")).not.toContainText("Viewer API create denied");
        await page.goto(`${baseURL}/admin/news/${createdNewsID}`);
        await expect(page.locator("#news-title-readonly")).toHaveValue("Editor role news draft");
        await expect(page.locator("#news-status-readonly")).toHaveValue("draft");
        const taxonomyVerifierContext = await browser.newContext({ storageState: adminStorageState });
        try {
          const taxonomyVerifier = await taxonomyVerifierContext.newPage();
          expect((await taxonomyVerifier.goto(`${baseURL}/admin/news/taxonomy`))?.status()).toBe(200);
          await expect(taxonomyVerifier.locator("table tbody")).toContainText("role-news-taxonomy-original");
          await expect(taxonomyVerifier.locator("table tbody")).not.toContainText("role-news-taxonomy-forged");
        } finally {
          await taxonomyVerifierContext.close();
        }
      }

      if (account.expectedRole === "editor") {
        await expect(page.locator('.studio-sidebar a[href="/admin/users"]')).toHaveCount(0);
      }
      for (const route of [
        "/admin/users",
        "/admin/permissions",
        "/admin/security",
        "/admin/audit",
        "/admin/backup",
        "/admin/control",
        "/admin/operations",
        "/admin/developer",
        "/admin/developer/resources",
        "/admin/security/token-lifetimes",
        "/admin/reports/export?range=week",
      ]) {
        const denied = await page.goto(`${baseURL}${route}`);
        expect(denied?.status(), `${account.expectedRole} must be denied ${route}`).toBe(403);
      }
      const mutation = await page.request.post(`${baseURL}/admin/pages`, { form: { slug: "role-denied", title: "Role denied" } });
      expect(mutation.status()).toBe(403);

      const csrfResponse = await page.request.get(`${baseURL}/csrf-token`);
      expect(csrfResponse.status()).toBe(200);
      const { token } = await csrfResponse.json();
      expect((await context.cookies(baseURL)).find((cookie) => cookie.name === "hydro_csrf")?.value).toBe(token);
      for (const target of restrictedMutationTargets) {
        const route = new URL(target).pathname;
        const validCSRF = await page.request.post(target, { form: { _csrf: token } });
        expect(validCSRF.status(), `${account.expectedRole} must be denied privileged mutation ${route} with valid CSRF`).toBe(403);
        const missingCSRF = await page.request.post(target, { form: {} });
        expect(missingCSRF.status(), `${account.expectedRole} privileged mutation ${route} must reject missing CSRF`).toBe(403);
        const mismatchedCSRF = await page.request.post(target, { form: { _csrf: "deliberately-mismatched-csrf-token" } });
        expect(mismatchedCSRF.status(), `${account.expectedRole} privileged mutation ${route} must reject mismatched CSRF`).toBe(403);
      }
      if (account.expectedRole === "viewer") {
        const deniedImport = await page.request.post(`${baseURL}/admin/pages/import`, {
          form: { _csrf: token },
        });
        expect(deniedImport.status(), "a valid-CSRF Viewer import POST must be denied before file validation").toBe(403);
      }
      for (const restrictedMutation of [
        { route: "/admin/security/bans", form: { address: "198.51.100.223", duration: "1h" } },
        { route: "/admin/security/bans/remove", form: { address: "198.51.100.223" } },
        { route: "/admin/security/token-lifetimes", form: { password_reset_ttl: "1h", invitation_ttl: "24h" } },
        { route: "/admin/backup", form: { password: adminPassword } },
      ]) {
        const denied = await page.request.post(`${baseURL}${restrictedMutation.route}`, {
          form: { _csrf: token, ...restrictedMutation.form },
        });
        expect(denied.status(), `${account.expectedRole} cannot submit ${restrictedMutation.route} even with valid CSRF`).toBe(403);
      }
      const securityStateContext = await browser.newContext({ storageState: adminStorageState });
      try {
        const securityStatePage = await securityStateContext.newPage();
        expect((await securityStatePage.goto(`${baseURL}/admin/security`))?.status()).toBe(200);
        await expect(securityStatePage.locator("body")).not.toContainText("198.51.100.223");
        expect((await securityStatePage.goto(`${baseURL}/admin/security/token-lifetimes`))?.status()).toBe(200);
        await expect(securityStatePage.locator('select[name="password_reset_ttl"]')).toHaveValue(tokenLifetimeBaseline!.passwordReset);
        await expect(securityStatePage.locator('select[name="invitation_ttl"]')).toHaveValue(tokenLifetimeBaseline!.invitation);
      } finally {
        await securityStateContext.close();
      }
      if (account.expectedRole === "editor") {
        const createdFolder = await page.request.post(`${baseURL}/admin/media/folders`, {
          maxRedirects: 0,
          form: { _csrf: token, name: "role-editor-media-folder" },
        });
        expect(createdFolder.status(), "editor with media.manage may create a media folder").toBe(303);
        const mediaAfterCreate = await page.request.get(`${baseURL}/admin/media`);
        expect(mediaAfterCreate.status()).toBe(200);
        expect(await mediaAfterCreate.text()).toContain("role-editor-media-folder");

      } else {
        const deniedFolder = await page.request.post(`${baseURL}/admin/media/folders`, {
          maxRedirects: 0,
          form: { _csrf: token, name: "role-viewer-media-folder" },
        });
        expect(deniedFolder.status(), "viewer without media.manage cannot create a media folder").toBe(403);
      }
      const otherAccount = account.expectedRole === "editor" ? "viewer@example.test" : "editor@example.test";
      const target = managementTargets[otherAccount];
      const deniedRoleChange = await page.request.post(new URL(target.role, baseURL).href, {
        form: { _csrf: token, role: account.expectedRole === "editor" ? "editor" : "viewer" },
      });
      expect(deniedRoleChange.status(), `${account.expectedRole} cannot change another user's role`).toBe(403);
      const deniedPermissionChange = await page.request.post(new URL(target.permissions, baseURL).href, {
        form: { _csrf: token, "perm_admin.content.write": "deny" },
      });
      expect(deniedPermissionChange.status(), `${account.expectedRole} cannot change another user's permissions`).toBe(403);
      const deniedAccountDisable = await page.request.post(new URL(target.disable, baseURL).href, { form: { _csrf: token } });
      expect(deniedAccountDisable.status(), `${account.expectedRole} cannot disable another user's account`).toBe(403);

      const stateContext = await browser.newContext({ storageState: adminStorageState });
      try {
        const statePage = await stateContext.newPage();
        expect((await statePage.goto(`${baseURL}/admin/users`))?.status()).toBe(200);
        const stateRow = statePage.locator(".studio-table tbody tr").filter({ has: statePage.getByText(otherAccount, { exact: true }) });
        await expect(stateRow.locator('select[name="role"]'), `${account.expectedRole} role-change denial must leave the target role unchanged`).toHaveValue(target.currentRole);
        await expect(stateRow.locator('select[name="perm_admin.content.write"]'), `${account.expectedRole} permission denial must leave the target override unchanged`).toHaveValue(target.currentContentWrite);
        await expect(stateRow, `${account.expectedRole} account-disable denial must leave the target active`).toContainText("Aktif");
      } finally {
        await stateContext.close();
      }

      if (account.expectedRole === "editor") {
        const overrideContext = await browser.newContext({ storageState: adminStorageState });
        const overridePage = await overrideContext.newPage();
        try {
          const editorPermissionForm = async () => {
            await overridePage.goto(`${baseURL}/admin/users`);
            const editorRow = overridePage.locator(".studio-table tbody tr").filter({ has: overridePage.getByText("editor@example.test", { exact: true }) });
            const form = editorRow.locator('form[action$="/permissions"]');
            const details = editorRow.locator("details");
            await details.evaluate((element) => { (element as HTMLDetailsElement).open = true; });
            await expect(details).toHaveAttribute("open", "");
            return form;
          };
          const saveMediaPermission = async (value: "deny" | "allow" | "inherit") => {
            const form = await editorPermissionForm();
            await form.locator('[name="perm_admin.media.manage"]').selectOption(value);
            await expect(form.locator(".."), "permission details must remain open after changing the select").toHaveAttribute("open", "");
            const action = await form.getAttribute("action");
            expect(action).toBeTruthy();
            await expect(form.locator('button[type="submit"]')).toBeVisible();
            await expect(form.locator('button[type="submit"]')).toBeEnabled();
            const formValues = await form.evaluate((element) =>
              Object.fromEntries(new FormData(element as HTMLFormElement).entries()) as Record<string, string>,
            );
            const saved = await overridePage.request.post(new URL(action!, baseURL).href, { form: formValues });
            expect(saved.status(), `Admin can set editor media.manage to ${value}`).toBe(204);
          };
          const signInFreshEditor = async () => {
            const freshContext = await browser.newContext();
            const freshPage = await freshContext.newPage();
            await freshPage.goto(`${baseURL}/login`);
            await freshPage.getByLabel("Email address").fill("editor@example.test");
            await freshPage.getByLabel("Password").fill(adminPassword);
            await freshPage.getByRole("button", { name: "Sign in" }).click();
            await expect(freshPage).toHaveURL(/\/admin(?:\?.*)?$/);
            const csrfResponse = await freshPage.request.get(`${baseURL}/csrf-token`);
            expect(csrfResponse.status()).toBe(200);
            return { context: freshContext, page: freshPage, csrf: (await csrfResponse.json()).token as string };
          };

          await saveMediaPermission("deny");
          const revokedSession = await page.request.post(`${baseURL}/admin/media/folders`, {
            maxRedirects: 0,
            form: { _csrf: token, name: "role-editor-media-denied-folder" },
          });
          expect(revokedSession.status(), "changing permissions revokes the editor's existing session").toBe(401);
          const deniedEditor = await signInFreshEditor();
          const deniedFolder = await deniedEditor.page.request.post(`${baseURL}/admin/media/folders`, {
            maxRedirects: 0,
            form: { _csrf: deniedEditor.csrf, name: "role-editor-media-denied-folder" },
          });
          expect(deniedFolder.status(), "a valid-CSRF editor request must honor media.manage deny override").toBe(403);
          await deniedEditor.context.close();
          const adminMediaAfterDeny = await overridePage.request.get(`${baseURL}/admin/media`);
          expect(adminMediaAfterDeny.status()).toBe(200);
          expect(await adminMediaAfterDeny.text()).not.toContain("role-editor-media-denied-folder");

          await saveMediaPermission("allow");
          const allowedEditor = await signInFreshEditor();
          const allowedFolder = await allowedEditor.page.request.post(`${baseURL}/admin/media/folders`, {
            maxRedirects: 0,
            form: { _csrf: allowedEditor.csrf, name: "role-editor-media-override-folder" },
          });
          expect(allowedFolder.status(), "a valid-CSRF editor request must honor media.manage allow override").toBe(303);
          await allowedEditor.context.close();
          const adminMediaAfterAllow = await overridePage.request.get(`${baseURL}/admin/media`);
          expect(await adminMediaAfterAllow.text()).toContain("role-editor-media-override-folder");

          await saveMediaPermission("inherit");
        } finally {
          await overrideContext.close();
        }
      }
      await context.close();
    }
    const verificationContext = await browser.newContext({ storageState: adminStorageState });
    const mediaAdmin = await verificationContext.newPage();
    const mediaListing = await mediaAdmin.request.get(`${baseURL}/admin/media`);
    expect(mediaListing.status()).toBe(200);
    const mediaHTML = await mediaListing.text();
    expect(mediaHTML).toContain("role-editor-media-folder");
    expect(mediaHTML).not.toContain("role-viewer-media-folder");
    await verificationContext.close();
  });

  test("legacy item-list controls follow the selected Admin locale", async ({ browser }) => {
    expect(adminStorageState, "the preceding role test must create an authenticated state").toBeTruthy();
    const context = await browser.newContext({ storageState: adminStorageState });
    await context.addCookies([{ name: "hydro_admin_locale", value: "en", url: baseURL }]);
    const page = await context.newPage();
    await page.goto(`${baseURL}/admin/news/new`);
    await expect(page.locator("html")).toHaveAttribute("lang", "en");
    await expect(page.locator("#theme-toggle-btn")).toHaveAccessibleName(/theme; click to change/);
    await page.locator("#ui-family-select").selectOption("aurora");
    await expect(page.locator(".studio-toast").last()).toContainText("Aurora appearance applied.");
    await page.locator("#theme-toggle-btn").click();
    await expect(page.locator(".studio-toast").last()).toContainText(/theme applied\./);
    await page.locator(".studio-search").click();
    const englishPalette = page.locator(".studio-search-dialog");
    await expect(englishPalette).toHaveAttribute("aria-label", "Search Studio");
    await expect(englishPalette.locator(".search-section-title")).toHaveText("Quick actions & commands");
    await expect(englishPalette.locator(".search-footer")).toContainText("Navigate");
    const englishSearch = englishPalette.getByRole("combobox", { name: "Search Studio and run commands" });
    await expect(englishSearch).toHaveAttribute("placeholder", "Search screens and commands (⌘K)…");
    await expect(englishPalette.locator(".search-result-item").filter({ hasText: "Write News" })).toContainText("Create a news draft");
    await englishSearch.fill("WRITE");
    await expect(englishPalette.locator(".search-result-item").filter({ hasText: "Write News" })).toBeVisible();
    await englishSearch.press("Escape");
    await page.evaluate(() => (window as any).HydroToast("Locale check", "info", "", () => {}, 30_000));
    await expect(page.getByRole("button", { name: "Undo" })).toBeVisible();
    const media = page.locator(".media-select-field").filter({ has: page.locator('[data-media-target="media_cover_media_id"]') });
    await expect(media.locator(".media-select-empty span")).toHaveText("No image selected");
    await expect(media.locator(".media-select-preview img")).toHaveAttribute("alt", "Selected image");
    await page.getByRole("button", { name: "Add item" }).click();
    const row = page.locator("[data-item-list][data-field-name='key_takeaways'] .item-list-row").first();
    await expect(row.locator("input")).toHaveAttribute("placeholder", "Enter an important item…");
    await expect(row.locator("input")).toHaveAccessibleName("Önemli Noktalar (Key Takeaways) 1");
    await expect(row.getByRole("button")).toHaveAccessibleName("Remove item 1 from Önemli Noktalar (Key Takeaways)");
    await context.addCookies([{ name: "hydro_admin_locale", value: "tr", url: baseURL }]);
    await page.reload();
    await expect(page.locator("#theme-toggle-btn")).toHaveAccessibleName(/tema; değiştirmek için tıklayın/);
    await page.locator("#ui-family-select").selectOption("hydrogo");
    await expect(page.locator(".studio-toast").last()).toContainText("Görünüm HydroGo olarak değiştirildi.");
    await page.locator("#theme-toggle-btn").click();
    await expect(page.locator(".studio-toast").last()).toContainText("tema uygulandı.");
    await page.locator(".studio-search").click();
    const turkishPalette = page.locator(".studio-search-dialog");
    await expect(turkishPalette).toHaveAttribute("aria-label", "Studio içinde ara");
    await expect(turkishPalette.locator(".search-section-title")).toHaveText("Hızlı Menü & Komutlar");
    await expect(turkishPalette.locator(".search-result-item").filter({ hasText: "Yeni Haber Yaz" })).toContainText("Haber taslağı oluştur");
    const turkishSearch = turkishPalette.getByRole("combobox", { name: "Studio içinde ara ve komut çalıştır" });
    await turkishSearch.fill("YENİ HABER");
    await expect(turkishPalette.locator(".search-result-item").filter({ hasText: "Yeni Haber Yaz" })).toBeVisible();
    await turkishSearch.press("Escape");
    await page.evaluate(() => (window as any).HydroToast("Dil kontrolü", "info", "", () => {}, 30_000));
    await expect(page.getByRole("button", { name: "Geri Al" })).toBeVisible();
    await expect(page.locator(".media-select-empty span")).toHaveText("Görsel seçilmedi");
    await expect(page.locator(".media-select-preview img")).toHaveAttribute("alt", "Seçilen görsel");
    await page.getByRole("button", { name: /Madde ekle/ }).click();
    const turkishRow = page.locator("[data-item-list][data-field-name='key_takeaways'] .item-list-row").first();
    await expect(turkishRow.locator("input")).toHaveAttribute("placeholder", "Önemli bir madde girin…");
    await expect(turkishRow.getByRole("button")).toHaveAccessibleName("Önemli Noktalar (Key Takeaways) 1 maddesini sil");
    await context.close();
  });

  test("session expiry warning follows the Admin language and stays accessible", async ({ browser }) => {
    expect(adminStorageState, "the preceding role test must create an authenticated state").toBeTruthy();
    for (const locale of ["en", "tr"] as const) {
      const context = await browser.newContext({ storageState: adminStorageState });
      await context.addCookies([{ name: "hydro_admin_locale", value: locale, url: baseURL }]);
      const page = await context.newPage();
      await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
      await page.clock.install();
      let resolveAuthRequest = () => {};
      const authRequest = new Promise<void>((resolve) => { resolveAuthRequest = resolve; });
      await page.route("**/auth/me", async (route) => {
        resolveAuthRequest();
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({ session_expires_at: new Date(Date.now() + 60_000).toISOString() }),
        });
      });
      await page.goto(`${baseURL}/admin`);
      await authRequest;
      await page.clock.runFor(5);
      const title = locale === "en" ? "Your session will expire soon" : "Oturumunuz yakında sona erecek";
      const dialog = page.getByRole("alertdialog", { name: title });
      await expect(dialog).toBeVisible();
      await expect(dialog.getByRole("button", { name: locale === "en" ? "Extend session" : "Oturumu uzat" })).toBeVisible();
      const audit = await page.evaluate(async () => (window as any).axe.run(document, {
        runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
      }));
      expect(audit.violations.map((violation: any) => violation.id), `${locale} session warning WCAG violations`).toEqual([]);
      await context.close();
    }
  });

  test("admin login opens the Edition management areas and rejects anonymous access", async ({ page, browser }) => {
    test.setTimeout(180_000);
    for (const route of [
      "/admin",
      "/admin/pages/new",
      "/admin/news/new",
      "/admin/media",
      "/admin/users",
      "/admin/permissions",
      "/admin/security",
      "/admin/security/token-lifetimes",
      "/admin/audit",
      "/admin/backup",
      "/admin/control",
      "/admin/operations",
      "/admin/developer",
      "/admin/developer/resources",
      "/admin/reports",
      "/admin/reports/export?range=week",
    ]) {
      const denied = await page.request.get(`${baseURL}${route}`, { maxRedirects: 0 });
      expect([401, 302, 303], `anonymous access to ${route} must require login`).toContain(denied.status());
      if (denied.status() !== 401) expect(denied.headers().location, `${route} must redirect to login`).toContain("/login");
    }

    await page.goto(`${baseURL}/login`);
    const loginResponse = await page.goto(`${baseURL}/login`);
    const loginHTML = await loginResponse?.text() ?? "";
    const csp = loginResponse?.headers()["content-security-policy"] ?? "";
    for (const [, kind, body] of loginHTML.matchAll(/<(script|style)(?:\s[^>]*)?>([\s\S]*?)<\/\1>/g)) {
      const hash = `sha256-${Buffer.from(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(body))).toString("base64")}`;
      expect(csp).toContain(`'${hash}'`);
      expect(csp).toContain(kind === "script" ? "script-src" : "style-src");
    }
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await page.reload();
    const loginA11y = await page.evaluate(async () => (window as any).axe.run(document, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
    }));
    expect(loginA11y.violations.map((violation: any) => ({
      id: violation.id,
      impact: violation.impact,
      description: violation.description,
      nodes: violation.nodes.map((node: any) => ({ target: node.target, summary: node.failureSummary })),
    })), "admin login WCAG 2.1 A/AA violations").toEqual([]);
    expect(adminStorageState, "the preceding role test must capture an administrator session").toBeDefined();
    await page.context().addCookies(adminStorageState!.cookies);
    await page.goto(`${baseURL}/admin`);
    await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
    await dismissAdminTour(page);
    adminStorageState = await page.context().storageState();
    await page.goto(`${baseURL}/admin`);
    await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
    await expect(page.locator("h1")).toBeVisible();

    for (const route of [
      "/admin",
      "/admin/pages/new",
      "/admin/news/new",
      "/admin/media",
      "/admin/users",
      "/admin/permissions",
      "/admin/security",
      "/admin/audit",
      "/admin/backup",
      "/admin/control",
      "/admin/operations",
      "/admin/developer",
      "/admin/developer/resources",
      "/admin/reports",
    ]) {
      const response = await page.goto(`${baseURL}${route}`);
      expect(response?.status(), `${route} should be available to the Edition admin`).toBe(200);
      await expect(page.locator("h1").first()).toBeVisible();

      const a11y = await page.evaluate(async () => (window as any).axe.run(document, {
        runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
      }));
      expect(a11y.violations.map((violation: any) => ({
        id: violation.id,
        impact: violation.impact,
        description: violation.description,
        nodes: violation.nodes.map((node: any) => ({ target: node.target, summary: node.failureSummary })),
      })), `${route} WCAG 2.1 A/AA violations`).toEqual([]);

      const unnamedControls = await page.locator('a,button,input:not([type="hidden"]),select,textarea,[role="button"],[contenteditable="true"]').evaluateAll((elements) =>
        elements.filter((element) => {
          const html = element as HTMLElement;
          if (html.hidden || html.getClientRects().length === 0 || getComputedStyle(html).display === "none" || getComputedStyle(html).visibility === "hidden") return false;
          const labelledBy = html.getAttribute("aria-labelledby")?.split(/\s+/).map((id) => document.getElementById(id)?.textContent ?? "").join(" ");
          const label = (html.id ? document.querySelector(`label[for="${CSS.escape(html.id)}"]`)?.textContent : "") || html.closest("label")?.textContent;
          const value = html instanceof HTMLInputElement && ["submit", "button", "reset"].includes(html.type) ? html.value : "";
          const name = [html.getAttribute("aria-label"), labelledBy, label, value, html.textContent, html.getAttribute("title")]
            .some((part) => part?.trim());
          return !name;
        }).map((element) => {
          const html = element as HTMLElement;
          return `${html.tagName.toLowerCase()}${html.id ? `#${html.id}` : ""}${html.getAttribute("name") ? `[name=${html.getAttribute("name")}]` : ""}`;
        }),
      );
      expect(unnamedControls, `${route} visible controls must have accessible names`).toEqual([]);

      const mutations = await page.locator('form[method="post"]').evaluateAll((forms) =>
        forms.map((form) => {
          const element = form as HTMLFormElement;
          return {
            action: element.action || location.href,
            csrf: element.querySelector<HTMLInputElement>('input[name="_csrf"]')?.value ?? "",
          };
        }),
      );
      for (const mutation of mutations) {
        expect(mutation.csrf, `${route} POST form must contain a CSRF token`).not.toBe("");
        const target = new URL(mutation.action);
        expect(target.origin, `${route} mutation target must stay on this Edition`).toBe(new URL(baseURL).origin);
        const deniedMutation = await page.request.post(target.href, { form: {} });
        expect(deniedMutation.status(), `CSRF-less POST to ${target.pathname} must be denied`).toBe(403);
      }
    }

    await page.goto(`${baseURL}/admin/security`);
    await page.route(`${baseURL}/admin/auth/invites`, async (route) => {
      const request = route.request();
      expect(request.method()).toBe("POST");
      expect(request.headers()["x-csrf-token"]).toBeTruthy();
      expect(request.postData()).toContain("invitee@example.test");
      expect(request.postData()).toContain("role");
      await route.fulfill({
        status: 202,
        contentType: "application/json",
        body: JSON.stringify({ email: "invitee@example.test", url: `${baseURL}/auth/invites/accept?token=e2e-invite-token` }),
      });
    });
    const inviteForm = page.locator("form[data-admin-invite]");
    await inviteForm.getByLabel("E-posta Adresi").fill("invitee@example.test");
    await inviteForm.getByRole("button", { name: /Davet Bağlantısı Oluştur/ }).click();
    await expect(inviteForm.locator("[data-invite-status]")).toContainText("Davet bağlantısı oluşturuldu");
    await expect(inviteForm.locator("[data-invite-link]")).toHaveValue(`${baseURL}/auth/invites/accept?token=e2e-invite-token`);
    await page.unroute(`${baseURL}/admin/auth/invites`);
    await page.route(`${baseURL}/admin/auth/invites`, (route) => route.fulfill({ status: 503, body: "unavailable" }));
    await inviteForm.getByLabel("E-posta Adresi").fill("offline@example.test");
    await inviteForm.getByRole("button", { name: /Davet Bağlantısı Oluştur/ }).click();
    await expect(inviteForm.locator("[data-invite-status]")).toContainText("Davet oluşturulamadı");
    await page.unroute(`${baseURL}/admin/auth/invites`);

    const testBanAddress = "198.51.100.222";
    const banForm = page.locator('form[action="/admin/security/bans"]');
    await banForm.getByLabel("IP adresi veya CIDR").fill(testBanAddress);
    await banForm.getByLabel(/Süre/).fill("1h");
    await banForm.getByRole("button", { name: "IP engelle" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/security`);
    await expect(page.getByText(`${testBanAddress}/32`, { exact: true })).toBeVisible();
    const unbanForm = page.locator('form[action="/admin/security/bans/remove"]');
    await unbanForm.getByLabel("Engeli kaldırılacak adres veya CIDR").fill(testBanAddress);
    await unbanForm.getByRole("button", { name: "Engeli kaldır" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/security`);
    await expect(page.getByText(testBanAddress, { exact: true })).toHaveCount(0);

    await page.goto(`${baseURL}/admin/backup`);
    const backupForm = page.locator('form[action="/admin/backup"]');
    await backupForm.getByLabel("Devam etmek için parolanızı doğrulayın").fill(adminPassword);
    const downloadResponse = page.waitForResponse((response) => response.url() === `${baseURL}/admin/backup` && response.request().method() === "POST");
    const backupDownload = page.waitForEvent("download", { timeout: 10_000 });
    void backupDownload.catch(() => {});
    await backupForm.getByRole("button", { name: "Yedeği oluştur ve indir" }).click();
    const backupResponse = await downloadResponse;
    expect(backupResponse.status()).toBe(200);
    const download = await backupDownload;
    expect(backupResponse.headers()["content-type"]).toContain("application/gzip");
    expect(backupResponse.headers()["cache-control"]).toContain("no-store");
    expect(download.suggestedFilename()).toMatch(/^hydrogo-backup-.*\.tar\.gz$/);

    await page.goto(`${baseURL}/admin/account`);
    const localeForm = page.locator('form[action="/admin/account/locale"]');
    await localeForm.locator('[name="locale"]').selectOption("en");
    await localeForm.getByRole("button", { name: /Dili kaydet|Save language/i }).click();
    await expect(page).toHaveURL(/\/admin\/account\?locale_saved=1$/);
    await page.goto(`${baseURL}/admin/backup`);
    await expect(page.getByRole("heading", { name: "Full site backup" })).toBeVisible();
    await expect(page.getByLabel("Confirm your password to continue")).toBeVisible();
    await page.goto(`${baseURL}/admin/account`);
    const restoreLocaleForm = page.locator('form[action="/admin/account/locale"]');
    await restoreLocaleForm.locator('[name="locale"]').selectOption("tr");
    await restoreLocaleForm.getByRole("button", { name: /Dili kaydet|Save language/i }).click();
    await expect(page).toHaveURL(/\/admin\/account\?locale_saved=1$/);

    const usersResponse = await page.goto(`${baseURL}/admin/users`);
    expect(usersResponse?.status()).toBe(200);
    const editorRow = page.locator(".studio-table tbody tr").filter({ has: page.getByText("editor@example.test", { exact: true }) });
    const permissionForm = editorRow.locator('form[action$="/permissions"]');
    const permissionToken = await permissionForm.locator('[name="_csrf"]').inputValue();
    const permissionAction = await permissionForm.getAttribute("action");
    expect(permissionAction).toBeTruthy();
    const denyPermission = await page.request.post(new URL(permissionAction!, baseURL).href, {
      form: { _csrf: permissionToken, "perm_admin.content.read": "deny", "perm_admin.content.write": "deny", "perm_admin.media.manage": "deny" },
    });
    expect(denyPermission.status()).toBe(204);
    await page.reload();
    await expect(editorRow.locator('[name="perm_admin.content.write"]')).toHaveValue("deny");

    const restrictedContext = await browser.newContext();
    const restrictedEditor = await restrictedContext.newPage();
    await restrictedEditor.goto(`${baseURL}/login`);
    await restrictedEditor.getByLabel("Email address").fill("editor@example.test");
    await restrictedEditor.getByLabel("Password").fill(adminPassword);
    await restrictedEditor.getByRole("button", { name: "Sign in" }).click();
    await expect(restrictedEditor).toHaveURL(/\/admin(?:\?.*)?$/);
    const deniedCreatePage = await restrictedEditor.goto(`${baseURL}/admin/pages/new`);
    expect(deniedCreatePage?.status()).toBe(403);
    expect((await restrictedEditor.goto(`${baseURL}/admin/pages/import`))?.status()).toBe(403);
    expect((await restrictedEditor.goto(`${baseURL}/admin/media`))?.status()).toBe(403);
    await expect(restrictedEditor.locator('form[action="/admin/pages"]')).toHaveCount(0);
    expect((await restrictedEditor.goto(`${baseURL}/admin/pages`))?.status()).toBe(403);
    const csrfResponse = await restrictedEditor.request.get(`${baseURL}/csrf-token`);
    expect(csrfResponse.status()).toBe(200);
    const { token: restrictedCsrfToken } = await csrfResponse.json();
    const deniedCreateMutation = await restrictedEditor.request.post(`${baseURL}/admin/pages`, {
      form: { _csrf: restrictedCsrfToken, slug: "permission-denied-page", title: "Must not be created" },
    });
    expect(deniedCreateMutation.status()).toBe(403);
    const adminPageListAfterDeny = await page.request.get(`${baseURL}/admin/pages`);
    expect(adminPageListAfterDeny.status()).toBe(200);
    const adminPageListHTML = await adminPageListAfterDeny.text();
    expect(adminPageListHTML).not.toContain("permission-denied-page");
    expect(adminPageListHTML).not.toContain("Must not be created");
    const editPageURL = `${baseURL}/admin/pages/${createdPageID}`;
    const deniedUpdateMutation = await restrictedEditor.request.post(editPageURL, {
      form: { _csrf: restrictedCsrfToken, title: "Must not update the page" },
    });
    expect(deniedUpdateMutation.status()).toBe(403);
    const persistedPageResponse = await page.goto(editPageURL);
    expect(persistedPageResponse?.status()).toBe(200);
    await expect(page.locator('[name="title"]')).toHaveValue("Editor session page");
    const deniedPublishMutation = await restrictedEditor.request.post(`${editPageURL}/publish`, {
      form: { _csrf: restrictedCsrfToken, version: "1" },
    });
    expect(deniedPublishMutation.status()).toBe(403);
    const deniedTranslateMutation = await restrictedEditor.request.post(`${editPageURL}/translate`, {
      form: { _csrf: restrictedCsrfToken, locale: "en", title: "Denied translation", body: "Denied body" },
    });
    expect(deniedTranslateMutation.status()).toBe(403);
    const deniedRestoreMutation = await restrictedEditor.request.post(`${editPageURL}/revisions/1/restore`, {
      form: { _csrf: restrictedCsrfToken },
    });
    expect(deniedRestoreMutation.status()).toBe(403);
    const deniedPreviewLink = await restrictedEditor.request.post(`${editPageURL}/preview-link`, {
      form: { _csrf: restrictedCsrfToken },
    });
    expect(deniedPreviewLink.status()).toBe(403);
    const apiHeaders = { "X-CSRF-Token": restrictedCsrfToken };
    const deniedAPICreate = await restrictedEditor.request.post(`${baseURL}/api/pages`, {
      headers: apiHeaders,
      data: { locale: "en", slug: "denied-api-page", title: "Denied API page", body: "" },
    });
    expect(deniedAPICreate.status()).toBe(403);
    const deniedAPIUpdate = await restrictedEditor.request.put(`${baseURL}/api/pages/${createdPageID}`, {
      headers: apiHeaders,
      data: { locale: "en", slug: "role-editor-page", title: "Denied API update", body: "", version: 1 },
    });
    expect(deniedAPIUpdate.status()).toBe(403);
    const deniedAPIPublish = await restrictedEditor.request.post(`${baseURL}/api/pages/${createdPageID}/publish?version=1`, {
      headers: apiHeaders,
    });
    expect(deniedAPIPublish.status()).toBe(403);
    const deniedImportMutation = await restrictedEditor.request.post(`${baseURL}/admin/pages/import`, {
      form: { _csrf: restrictedCsrfToken },
    });
    expect(deniedImportMutation.status()).toBe(403);
    const deniedBulkExport = await restrictedEditor.request.post(`${baseURL}/admin/pages/bulk`, {
      form: { _csrf: restrictedCsrfToken, action: "export", ids: createdPageID },
    });
    expect(deniedBulkExport.status()).toBe(403);
    expect(deniedBulkExport.headers()["content-disposition"]).toBeUndefined();
    const pageAfterDeniedPublish = await page.request.get(`${baseURL}/api/pages/${createdPageID}`);
    expect(pageAfterDeniedPublish.status()).toBe(200);
    expect(await pageAfterDeniedPublish.json()).toMatchObject({ status: "draft", title: "Editor session page" });
    const finalAdminPageList = await page.request.get(`${baseURL}/admin/pages`);
    const finalAdminPageListHTML = await finalAdminPageList.text();
    expect(finalAdminPageListHTML).not.toContain("denied-api-page");
    expect(finalAdminPageListHTML).not.toContain("Denied translation");
    const deniedSearch = await restrictedEditor.request.get(`${baseURL}/admin/api/search?q=Hydro`);
    expect(deniedSearch.status()).toBe(200);
    const deniedSearchBody = await deniedSearch.json();
    expect(deniedSearchBody.items.map((item: { badge: string }) => item.badge)).not.toContain("Sayfa");
    expect(deniedSearchBody.items.map((item: { badge: string }) => item.badge)).not.toContain("Haber");

    const grantToken = await editorRow.locator('form[action$="/permissions"] [name="_csrf"]').inputValue();
    const permissionUpdate = await page.request.post(new URL(permissionAction!, baseURL).href, {
      form: { _csrf: grantToken, "perm_admin.content.read": "allow", "perm_admin.content.write": "allow", "perm_admin.media.manage": "inherit" },
    });
    expect(await permissionUpdate.text()).toBe("");
    expect(permissionUpdate.status()).toBe(204);
    await page.reload();
    await expect(editorRow.locator('[name="perm_admin.content.write"]')).toHaveValue("allow");
    expect((await restrictedEditor.request.get(`${baseURL}/auth/me`)).status()).toBe(401);
    await restrictedContext.close();
    const grantedContext = await browser.newContext();
    const grantedEditor = await grantedContext.newPage();
    await grantedEditor.goto(`${baseURL}/login`);
    await grantedEditor.getByLabel("Email address").fill("editor@example.test");
    await grantedEditor.getByLabel("Password").fill(adminPassword);
    await grantedEditor.getByRole("button", { name: "Sign in" }).click();
    await expect(grantedEditor).toHaveURL(/\/admin(?:\?.*)?$/);
    await grantedEditor.goto(`${baseURL}/admin/pages`);
    await grantedEditor.locator('.studio-search').click();
    await expect(grantedEditor.locator('.studio-search-dialog .search-result-item').filter({ hasText: "Yeni Sayfa Ekle" })).toBeVisible();
    await expect(grantedEditor.locator('.studio-search-dialog .search-result-item').filter({ hasText: "Yeni Haber Yaz" })).toBeVisible();
    await expect(grantedEditor.locator('.studio-search-dialog .search-result-item').filter({ hasText: "Medya Yükle" })).toBeVisible();
    await grantedContext.close();

    const viewerCapabilityRow = page.locator(".studio-table tbody tr").filter({ has: page.getByText("viewer@example.test", { exact: true }) });
    const viewerPermissionForm = viewerCapabilityRow.locator('form[action$="/permissions"]');
    const viewerPermissionAction = await viewerPermissionForm.getAttribute("action");
    expect(viewerPermissionAction).toBeTruthy();
    const viewerPermissionToken = await viewerPermissionForm.locator('[name="_csrf"]').inputValue();
    const allowViewerContent = await page.request.post(new URL(viewerPermissionAction!, baseURL).href, {
      form: { _csrf: viewerPermissionToken, "perm_admin.content.read": "allow", "perm_admin.content.write": "allow" },
    });
    expect(allowViewerContent.status()).toBe(204);

    const elevatedViewerContext = await browser.newContext();
    const elevatedViewer = await elevatedViewerContext.newPage();
    await elevatedViewer.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await elevatedViewer.goto(`${baseURL}/login`);
    await elevatedViewer.getByLabel("Email address").fill("viewer@example.test");
    await elevatedViewer.getByLabel("Password").fill(adminPassword);
    await elevatedViewer.getByRole("button", { name: "Sign in" }).click();
    await expect(elevatedViewer).toHaveURL(/\/admin(?:\?.*)?$/);
    expect((await elevatedViewer.goto(`${baseURL}/admin/pages/new`))?.status()).toBe(200);
    await expectAxeClean(elevatedViewer, "Viewer Pages create form");
    await expect(elevatedViewer.locator('body[data-studio-role="viewer"]')).toBeVisible();
    const viewerCreateForm = elevatedViewer.locator('form[action="/admin/pages"]');
    await viewerCreateForm.locator('[name="slug"]').fill("viewer-capability-page");
    await viewerCreateForm.locator('[name="title"]').fill("Viewer capability page");
    await viewerCreateForm.locator(".richtext-editor [contenteditable=true]").fill("Created with explicit content permissions.");
    await viewerCreateForm.getByRole("button", { name: "Taslak oluştur" }).click();
    await expect(elevatedViewer).toHaveURL(/\/admin\/pages\/[^/?]+$/);
    const viewerPageID = new URL(elevatedViewer.url()).pathname.split("/").at(-1)!;
    const viewerCreatedAPI = await page.request.get(`${baseURL}/api/pages/${viewerPageID}`);
    expect(viewerCreatedAPI.status()).toBe(200);
    expect(await viewerCreatedAPI.json()).toMatchObject({ status: "draft", title: "Viewer capability page" });
    expect((await elevatedViewer.goto(`${baseURL}/admin/news/new`))?.status(), "Viewer content.write override grants the News editor route").toBe(200);
    await expectAxeClean(elevatedViewer, "Viewer News create form");
    await elevatedViewer.locator('#news-form [name="title"]').fill("Viewer capability news");
    await elevatedViewer.locator('#news-form [name="slug"]').fill("viewer-capability-news");
    await elevatedViewer.locator('#news-form .richtext-editor [contenteditable="true"]').fill("Created with an explicit content.write override.");
    await elevatedViewer.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(elevatedViewer).toHaveURL(/\/admin\/news\/[^/?]+$/);
    const viewerNewsID = new URL(elevatedViewer.url()).pathname.split("/").at(-1)!;
    await expectAxeClean(elevatedViewer, "Viewer News edit form");
    const viewerCreatedNewsAPI = await page.request.get(`${baseURL}/api/news/${viewerNewsID}`);
    expect(viewerCreatedNewsAPI.status()).toBe(200);
    expect(await viewerCreatedNewsAPI.json()).toMatchObject({ status: "draft", title: "Viewer capability news" });
    const viewerNewsCSRFResponse = await elevatedViewer.request.get(`${baseURL}/csrf-token`);
    expect(viewerNewsCSRFResponse.status()).toBe(200);
    const { token: viewerNewsCSRF } = await viewerNewsCSRFResponse.json();
    expect((await elevatedViewerContext.cookies(baseURL)).find((cookie) => cookie.name === "hydro_csrf")?.value).toBe(viewerNewsCSRF);
    const viewerNewsAPICreate = await elevatedViewer.request.post(`${baseURL}/api/news`, {
      headers: { "X-CSRF-Token": viewerNewsCSRF },
      data: {
        locale: "tr",
        slug: "viewer-capability-news-api",
        title: "Viewer capability API news",
        body: "Created with explicit content.write through the News API.",
      },
    });
    expect(viewerNewsAPICreate.status(), "Viewer content.write override grants News API create with valid CSRF").toBe(200);
    const viewerCreatedNewsAPIResponse = await viewerNewsAPICreate.json();
    const viewerNewsAPIReadback = await page.request.get(`${baseURL}/api/news/${viewerCreatedNewsAPIResponse.id}`);
    expect(viewerNewsAPIReadback.status()).toBe(200);
    expect(await viewerNewsAPIReadback.json()).toMatchObject({ status: "draft", title: "Viewer capability API news" });

    await page.goto(`${baseURL}/admin/users`);
    const refreshedViewerRow = page.locator(".studio-table tbody tr").filter({ has: page.getByText("viewer@example.test", { exact: true }) });
    const revokeViewerPermissionForm = refreshedViewerRow.locator('form[action$="/permissions"]');
    const revokeViewerPermissionAction = await revokeViewerPermissionForm.getAttribute("action");
    const revokeViewerPermissionToken = await revokeViewerPermissionForm.locator('[name="_csrf"]').inputValue();
    const revokeViewerContent = await page.request.post(new URL(revokeViewerPermissionAction!, baseURL).href, {
      form: { _csrf: revokeViewerPermissionToken, "perm_admin.content.read": "inherit", "perm_admin.content.write": "inherit" },
    });
    expect(revokeViewerContent.status()).toBe(204);
    expect((await elevatedViewer.request.get(`${baseURL}/auth/me`)).status()).toBe(401);
    await elevatedViewerContext.close();

    const roleForm = editorRow.locator('form[action$="/role"]');
    const roleToken = await roleForm.locator('[name="_csrf"]').inputValue();
    const roleAction = await roleForm.getAttribute("action");
    expect(roleAction).toBeTruthy();
    const roleUpdate = await page.request.post(new URL(roleAction!, baseURL).href, {
      form: { _csrf: roleToken, role: "viewer" },
    });
    expect(roleUpdate.status()).toBe(204);
    await page.reload();
    await expect(editorRow.locator('[name="role"]')).toHaveValue("viewer");

    const restoreRoleForm = editorRow.locator('form[action$="/role"]');
    const restoreRoleToken = await restoreRoleForm.locator('[name="_csrf"]').inputValue();
    const restoreRoleAction = await restoreRoleForm.getAttribute("action");
    const restoreRole = await page.request.post(new URL(restoreRoleAction!, baseURL).href, {
      form: { _csrf: restoreRoleToken, role: "editor" },
    });
    expect(restoreRole.status()).toBe(204);
    await page.reload();
    await expect(editorRow.locator('[name="role"]')).toHaveValue("editor");

    const editorResponse = await page.goto(`${baseURL}/admin/pages/new`);
    await expect(page.getByRole("group", { name: "Sayfa düzeni" })).toBeVisible();
    await expect(page.getByText("Boş sayfa")).toBeVisible();
    await expect(page.getByText("Hizmet tanıtımı")).toBeVisible();
    const picker = page.locator(".ds-layout-picker");
    await expect(picker.locator("input[type=radio][value=blank]")).toBeChecked();
    const pickerLayout = await page.locator(".ds-layout-picker__options").evaluate((element) => {
      const picker = getComputedStyle(element);
      const fieldset = getComputedStyle(element.parentElement!);
      const finalOption = getComputedStyle(element.lastElementChild!);
      return { columns: picker.gridTemplateColumns, span: fieldset.gridColumnEnd, finalOptionSpan: finalOption.gridColumnEnd };
    });
    expect(pickerLayout.columns.split(" ").length).toBe(2);
    expect(pickerLayout.span).toBe("-1");
    expect(pickerLayout.finalOptionSpan).toBe("auto");
    const editorHTML = await editorResponse?.text() ?? "";
    const inlineLayoutCSS = [...editorHTML.matchAll(/<style(?:\s[^>]*)?>([\s\S]*?)<\/style>/g)]
      .map((match) => match[1])
      .find((style) => style.includes(".ds-layout-picker"));
    expect(inlineLayoutCSS, "Admin shell must include the layout picker styles").toBeTruthy();
    const styleHash = `sha256-${Buffer.from(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(inlineLayoutCSS!))).toString("base64")}`;
    expect(editorResponse?.headers()["content-security-policy"]).toContain(`'${styleHash}'`);
    await page.getByLabel("Başarı hikâyesi").check();
    await expect(page.locator(".ds-layout-picker input[value=case_study]")).toBeChecked();
    await page.setViewportSize({ width: 390, height: 844 });
    const mobileLayout = await page.locator(".ds-layout-picker__options").evaluate((element) =>
      getComputedStyle(element).gridTemplateColumns,
    );
    expect(mobileLayout.split(" ").length).toBe(1);

    await page.setViewportSize({ width: 1280, height: 900 });
    await page.getByLabel("Boş sayfa").check();
    const createPage = page.locator('form[action="/admin/pages"]');
    await expect.poll(() => page.evaluate(() => typeof (window as any).HydroInitQuill)).toBe("function");
    await expect(createPage.locator(".richtext-editor [contenteditable=true]")).toBeVisible();
    await createPage.locator('[name="slug"]').fill("e2e-editorial-page");
    await createPage.locator('[name="title"]').fill("E2E editorial draft");
    await createPage.locator(".richtext-editor [contenteditable=true]").fill("Draft body for the editorial browser journey.");
    await createPage.getByRole("button", { name: "Taslak oluştur" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);

    const editPage = page.locator('form[data-autosave-resource^="page:"]');
    const faqBlock = editPage.locator("[data-block-row]").first();
    await faqBlock.locator('[name="block_type_0"]').selectOption("faq");
    await expect(faqBlock.locator(".faq-visual-builder")).toBeVisible();
    await faqBlock.locator(".faq-q-input").first().fill("Teslimat ne kadar sürer?");
    await faqBlock.locator(".faq-a-input").first().fill("Siparişiniz iki iş gününde ulaşır.");
    await faqBlock.locator(".faq-add-card-btn").click();
    await faqBlock.locator(".faq-q-input").nth(1).fill("İade yapabilir miyim?");
    await faqBlock.locator(".faq-a-input").nth(1).fill("Evet, 14 gün içinde iade edebilirsiniz.");
    await editPage.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+\?saved=1$/);
    const savedFAQBlock = page.locator('form[data-autosave-resource^="page:"] [data-block-row]').first();
    await expect(savedFAQBlock.locator('[name="block_type_0"]')).toHaveValue("faq");
    await expect(savedFAQBlock.locator(".faq-q-input").nth(1)).toHaveValue("İade yapabilir miyim?");

    await editPage.locator('[name="title"]').fill("E2E editorial published");
    await editPage.locator(".richtext-editor [contenteditable=true]").fill("Published body from the editorial browser journey.");
    await editPage.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+\?saved=1$/);
    await expect(page.locator('form[data-autosave-resource^="page:"] [name="title"]')).toHaveValue("E2E editorial published");

    const originalPagePath = new URL(page.url()).pathname;
    await page.goto(`${baseURL}${originalPagePath}?revision=1`);
    const revisionPanel = page.locator("[data-revision-diff]");
    await expect(revisionPanel).toBeVisible();
    await revisionPanel.locator("summary").click();
    await expect(revisionPanel.getByRole("button", { name: "Bu sürümü geri yükle" })).toBeVisible();
    page.once("dialog", (dialog) => dialog.accept());
    await revisionPanel.getByRole("button", { name: "Bu sürümü geri yükle" }).click();
    await expect(page).toHaveURL(`${baseURL}${originalPagePath}`);
    await expect(page.locator('form[data-autosave-resource^="page:"] [name="title"]')).toHaveValue("E2E editorial draft");

    const restoredEdit = page.locator('form[data-autosave-resource^="page:"]');
    const restoredFAQBlock = restoredEdit.locator("[data-block-row]").first();
    await restoredFAQBlock.locator('[name="block_type_0"]').selectOption("faq");
    await expect(restoredFAQBlock.locator(".faq-visual-builder")).toBeVisible();
    await restoredFAQBlock.locator(".faq-q-input").first().fill("Teslimat ne kadar sürer?");
    await restoredFAQBlock.locator(".faq-a-input").first().fill("Siparişiniz iki iş gününde ulaşır.");
    await restoredFAQBlock.locator(".faq-add-card-btn").click();
    await restoredFAQBlock.locator(".faq-q-input").nth(1).fill("İade yapabilir miyim?");
    await restoredFAQBlock.locator(".faq-a-input").nth(1).fill("Evet, 14 gün içinde iade edebilirsiniz.");
    await restoredEdit.locator('[name="title"]').fill("E2E editorial published");
    await restoredEdit.locator(".richtext-editor [contenteditable=true]").fill("Published body from the editorial browser journey.");
    await restoredEdit.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+\?saved=1$/);

    await page.goto(`${baseURL}${originalPagePath}?compare=en`);
    const translation = page.locator("[data-translation-editor]");
    await expect(translation).toBeVisible();
    await translation.getByLabel("Başlık çevirisi").fill("E2E editorial translated");
    await translation.getByLabel("Metin çevirisi").fill("Translated body from the editorial browser journey.");
    await translation.getByRole("button", { name: "Çeviriyi kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);
    await expect(page.locator('form[data-autosave-resource^="page:"] [name="title"]')).toHaveValue("E2E editorial translated");

    await page.goto(`${baseURL}${originalPagePath}`);
    await page.locator('form[action$="/publish"] button[type="submit"]').click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);
    const publishedPage = await page.request.get(`${baseURL}/public/pages/tr/e2e-editorial-page/html`);
    expect(publishedPage.status()).toBe(200);
    await expect(publishedPage).toBeOK();
    expect(await publishedPage.text()).toContain("E2E editorial published");
    expect(await publishedPage.text()).toContain("Teslimat ne kadar sürer?");
    expect(await publishedPage.text()).toContain("Evet, 14 gün içinde iade edebilirsiniz.");

    const mediaName = "hydro-picker-e2e.png";
    await page.goto(`${baseURL}/admin/media`);
    await page.locator("#media-file-input").setInputFiles({
      name: mediaName,
      mimeType: "image/png",
      buffer: Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg==", "base64"),
    });
    const uploadResponsePromise = page.waitForResponse((response) => response.url() === `${baseURL}/admin/media` && response.request().method() === "POST");
    await page.locator('#media-upload-form button[type="submit"]').click();
    const uploadResponse = await uploadResponsePromise;
    expect(uploadResponse.status()).toBe(303);
    await expect(page.locator(`.media-card[data-filename="${mediaName}"]`)).toBeVisible();

    await page.goto(`${baseURL}/admin/news/new`);
    const coverField = page.locator('#news-form .media-select-field');
    await coverField.getByRole("button", { name: "Kütüphaneden seç" }).click();
    const mediaModal = page.locator("#admin-media-picker-modal");
    await expect(mediaModal).toBeVisible();
    const mediaChoice = mediaModal.getByRole("button", { name: mediaName });
    await expect(mediaChoice).toBeVisible();
    await mediaChoice.click();
    const coverID = page.locator('#news-form [name="cover_media_id"]');
    await expect(coverID).toHaveValue(/.+/);
    const selectedMediaID = await coverID.inputValue();
    await expect(coverField.locator(".media-select-preview img")).toHaveAttribute("src", `/media/${selectedMediaID}`);
    await coverField.getByRole("button", { name: "Kaldır" }).click();
    await expect(coverID).toHaveValue("");
    await coverField.getByRole("button", { name: "Kütüphaneden seç" }).click();
    await mediaModal.getByRole("button", { name: mediaName }).click();
    await expect(coverID).toHaveValue(selectedMediaID);

    await page.locator('#news-form [name="title"]').fill("E2E media picker story");
    await page.locator('#news-form [name="slug"]').fill("e2e-media-picker-story");
    await page.locator('#news-form .richtext-editor [contenteditable="true"]').fill("Created from the media picker browser flow.");
    await page.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/news\/[^/?]+$/);
    await expect(page.locator('#news-edit-form [name="cover_media_id"]')).toHaveValue(selectedMediaID);

    const previewDialog = page.locator("[data-device-preview-dialog]");
    const previewFrameResponse = page.waitForResponse((response) => response.request().method() === "GET" && new URL(response.url()).searchParams.has("preview_token"));
    await page.locator("[data-device-preview-open]").click();
    const previewResponse = await previewFrameResponse;
    expect(previewResponse.status()).toBe(200);
    expect(previewResponse.headers()["cache-control"]).toContain("no-store");
    expect(previewResponse.headers()["x-robots-tag"]).toContain("noindex");
    expect(previewResponse.headers()["referrer-policy"]).toBe("no-referrer");
    expect(previewResponse.headers()["x-frame-options"]).toBe("SAMEORIGIN");
    expect(previewResponse.headers()["content-security-policy"]).toContain("frame-ancestors 'self'");
    expect(await previewResponse.text()).toContain("E2E media picker story");
    await expect(previewDialog.locator("[data-device-preview-status]")).toHaveText("Güncel taslak · betikler devre dışı");
    await expect(previewDialog.locator("iframe")).toHaveAttribute("src", /preview_token=/);
    await expect(previewDialog.locator("iframe").contentFrame().locator("body")).toContainText("E2E media picker story");
    await previewDialog.getByRole("button", { name: /Mobil/ }).click();
    await expect(previewDialog).toHaveAttribute("data-device-size", "mobile");
    await previewDialog.getByRole("button", { name: "Önizlemeyi kapat" }).click();

    const scheduledNewsPath = new URL(page.url()).pathname;
    const scheduleInput = page.getByLabel("Yayın tarihi ve saati");
    await expect(scheduleInput).toBeVisible();
    const scheduledAt = new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString().slice(0, 16);
    await scheduleInput.fill(scheduledAt);
    const scheduleResponsePromise = page.waitForResponse((response) => response.url() === `${baseURL}${scheduledNewsPath}/schedule` && response.request().method() === "POST");
    await page.getByRole("button", { name: "Zamanla" }).click();
    expect((await scheduleResponsePromise).status()).toBe(303);
    await expect(page).toHaveURL(new RegExp(`${scheduledNewsPath}$`));
    const scheduledStory = await page.request.get(`${baseURL}/public/news/tr/e2e-media-picker-story/html`);
    expect(scheduledStory.status(), "scheduled news must stay private until its publication time").toBe(404);
    expect(viewerStorageState, "the preceding role test must capture a viewer session").toBeDefined();
    const workflowViewerContext = await browser.newContext({ storageState: viewerStorageState });
    const workflowViewer = await workflowViewerContext.newPage();
    const viewerCSRFResponse = await workflowViewer.request.get(`${baseURL}/csrf-token`);
    expect(viewerCSRFResponse.status()).toBe(200);
    const { token: viewerCSRFToken } = await viewerCSRFResponse.json();
    const scheduledVersion = await page.locator('#news-publish-form [name="version"]').inputValue();
    for (const action of ["publish", "retract"]) {
      const deniedTransition = await workflowViewer.request.post(`${baseURL}${scheduledNewsPath}/${action}`, {
        form: { _csrf: viewerCSRFToken, version: scheduledVersion },
      });
      expect(deniedTransition.status(), `viewer with valid CSRF must not ${action} news`).toBe(403);
    }
    expect((await page.request.get(`${baseURL}/public/news/tr/e2e-media-picker-story/html`)).status()).toBe(404);
    await workflowViewerContext.close();
    const publishResponsePromise = page.waitForResponse((response) => response.url() === `${baseURL}${scheduledNewsPath}/publish` && response.request().method() === "POST");
    await page.getByRole("button", { name: "Yayına Al" }).click();
    expect((await publishResponsePromise).status()).toBe(303);
    const publishedStory = await page.request.get(`${baseURL}/public/news/tr/e2e-media-picker-story/html`);
    expect(publishedStory.status(), "manually publishing a scheduled news item must make it public").toBe(200);
    const retractResponsePromise = page.waitForResponse((response) => response.url() === `${baseURL}${scheduledNewsPath}/retract` && response.request().method() === "POST");
    await page.getByRole("button", { name: "Yayından Çek" }).click();
    expect((await retractResponsePromise).status()).toBe(303);
    const retractedStory = await page.request.get(`${baseURL}/public/news/tr/e2e-media-picker-story/html`);
    expect(retractedStory.status(), "retracted news must no longer be public").toBe(404);

    await page.goto(`${baseURL}/admin/news/new`);
    await page.locator('#news-form [name="title"]').fill("E2E worker scheduled story");
    await page.locator('#news-form [name="slug"]').fill("e2e-worker-scheduled-story");
    await page.locator('#news-form .richtext-editor [contenteditable="true"]').fill("Published by the running HydroGo news worker.");
    await page.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/news\/[^/?]+$/);
    const workerNewsPath = new URL(page.url()).pathname;
    const workerScheduleInput = page.getByLabel("Yayın tarihi ve saati");
    const workerScheduleAt = new Date();
    workerScheduleAt.setMinutes(workerScheduleAt.getMinutes() + 1, 0, 0);
    await workerScheduleInput.fill(workerScheduleAt.toISOString().slice(0, 16));
    const workerScheduleResponse = page.waitForResponse((response) => response.url() === `${baseURL}${workerNewsPath}/schedule` && response.request().method() === "POST");
    await page.getByRole("button", { name: "Zamanla" }).click();
    expect((await workerScheduleResponse).status()).toBe(303);
    const workerStoryURL = `${baseURL}/public/news/tr/e2e-worker-scheduled-story/html`;
    if (workerScheduleAt.getTime() - Date.now() > 2_000) {
      expect((await page.request.get(workerStoryURL)).status()).toBe(404);
    }
    await expect.poll(async () => (await page.request.get(workerStoryURL)).status(), {
      timeout: 75_000,
      intervals: [1_000],
      message: "the live news worker should publish the scheduled item when its time arrives",
    }).toBe(200);
    expect(await (await page.request.get(workerStoryURL)).text()).toContain("Published by the running HydroGo news worker.");

    await page.goto(`${baseURL}/admin/pages/new`);
    const conflictCreate = page.locator('form[action="/admin/pages"]');
    await conflictCreate.locator('[name="slug"]').fill("e2e-conflict-page");
    await conflictCreate.locator('[name="title"]').fill("Conflict draft");
    await conflictCreate.locator(".richtext-editor [contenteditable=true]").fill("Initial conflict content.");
    await conflictCreate.getByRole("button", { name: "Taslak oluştur" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);

    const staleEditor = await page.context().newPage();
    await staleEditor.goto(page.url());
    await dismissAdminTour(staleEditor);
    const currentEditor = page.locator('form[data-autosave-resource^="page:"]');
    await currentEditor.locator('[name="title"]').fill("Current page version");
    await currentEditor.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\?saved=1$/);

    const staleForm = staleEditor.locator('form[data-autosave-resource^="page:"]');
    await staleForm.locator('[name="title"]').fill("Stale page version");
    const staleSave = staleEditor.waitForResponse((response) => response.request().method() === "POST" && response.status() === 409);
    await staleForm.getByRole("button", { name: "Taslağı kaydet" }).click();
    const conflictResponse = await staleSave;
    expect(await conflictResponse.text()).toContain("Bu içerik başka biri tarafından güncellendi. Sayfayı yenileyip tekrar deneyin.");
    await staleEditor.goto(`${baseURL}${new URL(page.url()).pathname}`);
    await expect(staleEditor.locator('form[data-autosave-resource^="page:"] [name="title"]')).toHaveValue("Current page version");
    await staleEditor.close();

    expect(viewerStorageState, "role-boundary test must capture the authenticated viewer session").toBeDefined();
    const viewerContext = await browser.newContext({ storageState: viewerStorageState });
    const viewerPage = await viewerContext.newPage();
    expect((await viewerPage.request.get(`${baseURL}/auth/me`)).status()).toBe(200);

    await page.goto(`${baseURL}/admin/users`);
    const viewerRow = page.locator(".studio-table tbody tr").filter({ hasText: "viewer@example.test" });
    const disableForm = viewerRow.locator('form[action$="/disable"]');
    const disableToken = await disableForm.locator('[name="_csrf"]').inputValue();
    const disableAction = await disableForm.getAttribute("action");
    expect(disableAction).toBeTruthy();
    const disabled = await page.request.post(new URL(disableAction!, baseURL).href, { form: { _csrf: disableToken } });
    expect(disabled.status()).toBe(204);
    expect((await viewerPage.request.get(`${baseURL}/auth/me`)).status()).toBe(401);
    await page.reload();
    await expect(viewerRow).toContainText("Pasif");
    await viewerContext.close();
  });

  test("admin routes fit mobile and tablet viewports", async ({ browser }) => {
    const context = await browser.newContext(adminStorageState ? { storageState: adminStorageState } : undefined);
    const page = await context.newPage();
    if (!adminStorageState) {
      await page.goto(`${baseURL}/login`);
      await page.getByLabel("Email address").fill("admin@example.test");
      await page.getByLabel("Password").fill(adminPassword);
      await page.getByRole("button", { name: "Sign in" }).click();
      await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
      await dismissAdminTour(page);
    }
    for (const viewport of [{ name: "mobile", width: 390, height: 844 }, { name: "tablet", width: 768, height: 1024 }]) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      if (viewport.width === 390) {
        await page.goto(`${baseURL}/admin`);
        const menuToggle = page.getByRole("button", { name: "Menüyü aç" });
        await expect(menuToggle).toHaveAttribute("aria-expanded", "false");
        await menuToggle.click();
        await expect(menuToggle).toHaveAttribute("aria-expanded", "true");
        await expect(page.locator("body")).toHaveClass(/studio-mobile-nav-open/);
        await page.keyboard.press("Escape");
        await expect(menuToggle).toHaveAttribute("aria-expanded", "false");
        await expect(page.locator("body")).not.toHaveClass(/studio-mobile-nav-open/);
        await menuToggle.click();
        await page.locator('.studio-sidebar a[href="/admin/pages"]').click();
        await expect(page).toHaveURL(`${baseURL}/admin/pages`);
        await expect(page.locator("body")).not.toHaveClass(/studio-mobile-nav-open/);
      }
      for (const route of [
        "/admin",
        "/admin/pages/new",
        "/admin/news/new",
        "/admin/media",
        "/admin/users",
        "/admin/permissions",
        "/admin/security",
        "/admin/audit",
        "/admin/backup",
        "/admin/control",
        "/admin/operations",
        "/admin/developer",
        "/admin/developer/resources",
        "/admin/reports",
        "/admin/account",
      ]) {
        const response = await page.goto(`${baseURL}${route}`, { waitUntil: "domcontentloaded" });
        expect(response?.status(), `${route} must render at ${viewport.width}px`).toBe(200);
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
        expect(overflow, `${route} must not overflow at ${viewport.width}px`).toBeLessThanOrEqual(1);
        const smallHeaderTargets = await page.locator(".studio-mobile-menu-toggle, .topbar-right > button").evaluateAll((targets) => targets
          .filter((target) => target.getClientRects().length > 0)
          .filter((target) => target.getBoundingClientRect().height < 44)
          .map((target) => `${target.getAttribute("aria-label") || target.textContent?.trim()}: ${Math.round(target.getBoundingClientRect().height)}px`));
        expect(smallHeaderTargets, `${route} header controls must be at least 44px high`).toEqual([]);
        const smallActions = await page.locator(".studio-card button:not(.ql-toolbar button), .dashboard-card button:not(.ql-toolbar button), .studio-form button:not(.ql-toolbar button), .studio-card .button, .dashboard-card .button, .studio-form .button, .studio-actions button, .studio-actions .button").evaluateAll((targets) => targets
          .filter((target) => target.getClientRects().length > 0)
          .filter((target) => target.getBoundingClientRect().height < 44)
          .map((target) => `${target.textContent?.trim() || target.getAttribute("aria-label") || target.tagName}: ${Math.round(target.getBoundingClientRect().height)}px`));
        expect(smallActions, `${route} main actions must be at least 44px high`).toEqual([]);
        if (route === "/admin/media") {
          const mediaTargets = page.locator(".media-thumb-overlay .media-overlay-btn");
          expect(await mediaTargets.count(), "media cards must expose quick actions at narrow viewports").toBeGreaterThan(0);
          const smallMediaTargets = await mediaTargets.evaluateAll((targets) => targets
            .filter((target) => target.getClientRects().length > 0)
            .filter((target) => target.getBoundingClientRect().width < 44 || target.getBoundingClientRect().height < 44)
            .map((target) => `${target.getAttribute("title") || target.tagName}: ${Math.round(target.getBoundingClientRect().width)}×${Math.round(target.getBoundingClientRect().height)}px`));
          expect(smallMediaTargets, "media card quick actions must be at least 44×44px").toEqual([]);
        }
        const smallInfoTargets = await page.locator(".studio-info-tip").evaluateAll((targets) => targets
          .filter((target) => target.getClientRects().length > 0)
          .filter((target) => target.getBoundingClientRect().width < 24 || target.getBoundingClientRect().height < 24)
          .map((target) => `${target.getAttribute("aria-label") || "info"}: ${Math.round(target.getBoundingClientRect().width)}×${Math.round(target.getBoundingClientRect().height)}px`));
        expect(smallInfoTargets, `${route} info targets must be at least 24×24px`).toEqual([]);
      }
    }
    await context.close();
  });

  test("real invitation link creates a usable Edition editor account", async ({ browser }) => {
    test.setTimeout(120_000);
    const dialect = process.env.HYDRO_E2E_DATABASE_DIALECT ?? "sqlite";
    const database = process.env.HYDRO_E2E_DATABASE_DSN ?? path.join(tempDir, "edition.sqlite");
    const token = runCommand("go", ["run", "./tests/e2e/support/seedinvite", "--database-dialect", dialect, "--database-dsn", database, "--email", "invited-editor@example.test"], { cwd: repoRoot, stdio: ["ignore", "pipe", "pipe"] }).trim();
    const context = await browser.newContext();
    const page = await context.newPage();
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    const inviteURL = `${baseURL}/auth/invites/accept?token=${token}`;
    const response = await page.goto(inviteURL);
    expect(response?.status()).toBe(200);
    await expect(page.getByRole("heading", { name: "Set up your account" })).toBeVisible();
    await expect(page).toHaveURL(`${baseURL}/auth/invites/accept`);
    expect(response?.headers()["cache-control"]).toContain("no-store");
    expect(response?.headers()["referrer-policy"]).toBe("no-referrer");
    const a11y = await page.evaluate(async () => (window as any).axe.run(document, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
    }));
    expect(a11y.violations.map((violation: any) => violation.id)).toEqual([]);

    await page.getByLabel("Password", { exact: true }).fill("passwordpassword");
    await page.getByLabel("Confirm password").fill("passwordpassword");
    await page.getByRole("button", { name: "Accept invitation" }).click();
    await expect(page.getByRole("alert")).toContainText("less predictable password");

    await page.getByLabel("Password", { exact: true }).fill("Lime-Train7!Harbor");
    await page.getByLabel("Confirm password").fill("Lime-Train7!HarborDifferent");
    await page.getByRole("button", { name: "Accept invitation" }).click();
    await expect(page).toHaveURL(`${baseURL}/auth/invites/accept`);
    await expect(page.getByRole("alert")).toContainText("passwords do not match");

    await page.getByLabel("Password", { exact: true }).fill("Lime-Train7!Harbor");
    await page.getByLabel("Confirm password").fill("Lime-Train7!Harbor");
    await page.getByRole("button", { name: "Accept invitation" }).click();
    await expect(page).toHaveURL(`${baseURL}/login?invite_accepted=1`);
    await expect(page.getByRole("status")).toContainText("Your account is ready");
    await page.getByLabel("Email address").fill("invited-editor@example.test");
    await page.getByLabel("Password").fill("Lime-Train7!Harbor");
    const loginResponse = page.waitForResponse((response) => response.url() === `${baseURL}/auth/login` && response.request().method() === "POST");
    await page.getByRole("button", { name: "Sign in" }).click();
    let loginResult = await loginResponse;
    if (loginResult.status() === 429) {
      const retryAfterSeconds = Number(loginResult.headers()["retry-after"] ?? "60");
      await page.waitForTimeout((retryAfterSeconds + 1) * 1000);
      const retryResponse = page.waitForResponse((response) => response.url() === `${baseURL}/auth/login` && response.request().method() === "POST");
      await page.getByRole("button", { name: "Sign in" }).click();
      loginResult = await retryResponse;
    }
    expect(loginResult.status(), "invited editor login must authenticate").toBe(200);
    await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
    await expect(page.locator('body[data-studio-role="editor"]')).toBeVisible();
    await context.close();
  });

  test("public Admin confirmation states remain accessible in both color modes", async ({ browser }) => {
    test.setTimeout(180_000);
    const context = await browser.newContext();
    const page = await context.newPage();
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    const colorModes = ["light", "dark"];
    const confirmations = [
      { query: "invite_accepted=1", en: "Your account is ready", tr: "Hesabın hazır" },
      { query: "password_reset=1", en: "Your password has been updated", tr: "Parolan güncellendi" },
    ];
    for (const mode of colorModes) {
      for (const confirmation of confirmations) {
        for (const [locale, expected] of [["en", confirmation.en], ["tr", confirmation.tr]] as const) {
          await page.goto(`${baseURL}/login`);
          await page.evaluate((colorMode) => localStorage.setItem("hydro-ui-theme-v2", `workbench-${colorMode}`), mode);
          const response = await page.goto(`${baseURL}/login?${confirmation.query}&lang=${locale}`);
          expect(response?.status(), `${confirmation.query} must render in ${mode}/${locale}`).toBe(200);
          await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
          await expect(page.getByRole("status")).toContainText(expected);
          const result = await page.evaluate(async () => (window as any).axe.run(document, {
            runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
          }));
          expect(result.violations.map((violation: any) => ({
            id: violation.id,
            impact: violation.impact,
            nodes: violation.nodes.map((node: any) => ({ target: node.target, summary: node.failureSummary })),
          })), `${confirmation.query} WCAG 2.1 A/AA violations in ${mode}/${locale}`).toEqual([]);
        }
      }
    }
    await context.close();
  });

  test("failed Admin login gives accessible localized feedback in both color modes", async ({ browser }) => {
    test.setTimeout(180_000);
    const context = await browser.newContext();
    const page = await context.newPage();
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    for (const mode of ["light", "dark"]) {
      for (const locale of ["en", "tr"] as const) {
        await page.goto(`${baseURL}/login`);
        await page.evaluate((colorMode) => localStorage.setItem("hydro-ui-theme-v2", `workbench-${colorMode}`), mode);
        await page.goto(`${baseURL}/login?lang=${locale}`);
        await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
        await page.getByLabel(locale === "en" ? "Email address" : "E-posta Adresi").fill(`unknown-${mode}-${locale}@example.test`);
        await page.getByLabel(locale === "en" ? "Password" : "Şifre").fill("incorrect-password");
        const loginResponse = page.waitForResponse((response) => response.url() === `${baseURL}/auth/login` && response.request().method() === "POST");
        await page.getByRole("button", { name: locale === "en" ? "Sign in" : "Giriş Yap" }).click();
        expect((await loginResponse).status(), `invalid login must return 401 in ${mode}/${locale}`).toBe(401);
        const expected = locale === "en"
          ? "Email or password is incorrect. Check your details and try again."
          : "E-posta veya şifre hatalı. Lütfen bilgilerinizi kontrol edin.";
        await expect(page.locator("#login-status")).toHaveAttribute("role", "alert");
        await expect(page.getByRole("alert")).toHaveText(expected);
        const result = await page.evaluate(async () => (window as any).axe.run(document, {
          runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
        }));
        expect(result.violations.map((violation: any) => ({
          id: violation.id,
          impact: violation.impact,
          nodes: violation.nodes.map((node: any) => ({ target: node.target, summary: node.failureSummary })),
        })), `failed login WCAG 2.1 A/AA violations in ${mode}/${locale}`).toEqual([]);
      }
    }
    await context.close();
  });

  test("admin routes meet WCAG 2.1 A/AA in every Studio theme and color mode", async ({ browser }) => {
    test.setTimeout(180_000);
    expect(adminStorageState, "the preceding admin login test must create an authenticated state").toBeTruthy();
    const context = await browser.newContext({ storageState: adminStorageState });
    const page = await context.newPage();
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await page.goto(`${baseURL}/admin`);
    await dismissAdminTour(page);

    const routes = [
      "/admin",
      "/admin/pages/new",
      "/admin/news/new",
      "/admin/media",
      "/admin/users",
      "/admin/permissions",
      "/admin/security",
      "/admin/audit",
      "/admin/backup",
      "/admin/control",
      "/admin/operations",
      "/admin/developer",
      "/admin/developer/resources",
      "/admin/reports",
    ];
    const themes = ["hydrogo", "shadcn", "nova", "aurora"];
    for (const theme of themes) {
      for (const mode of ["light", "dark"]) {
        await page.evaluate(([family, colorMode]) => localStorage.setItem("hydro-ui-theme-v2", `${family}-${colorMode}`), [theme, mode]);
        for (const route of routes) {
          const response = await page.goto(`${baseURL}${route}`);
          expect(response?.status(), `${route} must render with ${theme}-${mode}`).toBe(200);
          await expect(page.locator("h1").first()).toBeVisible();
          await expect(page.locator("html")).toHaveAttribute("data-ui-theme", theme);
          await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
          const result = await page.evaluate(async () => (window as any).axe.run(document, {
            runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
          }));
          expect(result.violations.map((violation: any) => ({
            id: violation.id,
            impact: violation.impact,
            nodes: violation.nodes.map((node: any) => ({ target: node.target, summary: node.failureSummary })),
          })), `${route} WCAG 2.1 A/AA violations in ${theme}-${mode}`).toEqual([]);
        }
      }
    }
    await context.close();
  });


});

async function startAdminServer(): Promise<void> {
  server = spawnServer("go", serverArgs, { cwd: repoRoot, stdio: "ignore", env: serverEnv });
  const deadline = Date.now() + 45_000;
  while (Date.now() < deadline) {
    if (server.exitCode !== null) throw new Error(`HydroGo server exited with ${server.exitCode}`);
    try {
      const response = await fetch(`${baseURL}/healthz`);
      if (response.ok) return;
    } catch {
      // Wait while the Go command compiles and starts the local test server.
    }
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error("HydroGo server did not become healthy within 45 seconds");
}

async function unusedPort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const probe = net.createServer();
    probe.once("error", reject);
    probe.listen(0, "127.0.0.1", () => {
      const address = probe.address();
      if (!address || typeof address === "string") return reject(new Error("could not resolve an ephemeral port"));
      probe.close((error) => error ? reject(error) : resolve(address.port));
    });
  });
}

async function dismissAdminTour(page: import("@playwright/test").Page): Promise<void> {
  const dismissed = await page.evaluate(() => localStorage.getItem("hydro-onboarding-dismissed") === "1" || localStorage.getItem("hydro-onboarding-dismissed:admin") === "1");
  if (dismissed) return;
  const tour = page.locator("#admin-onboarding-dialog");
  await expect(tour).toBeVisible();
  await tour.locator("[data-onboarding-dismiss]").click();
}
