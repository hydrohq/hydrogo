import { test, expect, type BrowserContext } from "@playwright/test";
import { execFileSync, type ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { spawnServer, stopServer } from "./server-process";

const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
const backupSigningSeed = "ab".repeat(32);
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";
let viewerStorageState: Awaited<ReturnType<BrowserContext["storageState"]>> | undefined;
let adminStorageState: Awaited<ReturnType<BrowserContext["storageState"]>> | undefined;

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
    execFileSync("go", ["run", "./cmd/hydro", "migrate", "--database-dialect", dialect, "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
    execFileSync("go", ["run", "./cmd/hydro", "create-admin", "--email", "admin@example.test", "--password-stdin", "--database-dialect", dialect, "--database-dsn", database], { cwd: repoRoot, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    execFileSync("go", ["run", "./tests/e2e/support/seedroles", "--database-dialect", dialect, "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawnServer("go", ["run", "./cmd/hydro", "serve", "--dev", "--database-dialect", dialect, "--database-dsn", database, "--address", `127.0.0.1:${port}`], { cwd: repoRoot, stdio: "ignore", env: { ...process.env, HYDRO_PUBLIC_ORIGIN: baseURL, HYDRO_MEDIA_DIR: path.join(tempDir, "media"), HYDRO_BACKUP_SIGNING_SEED: backupSigningSeed, HYDRO_PREVIEW_SIGNING_KEY: "cd".repeat(32) } });
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
  }, 60_000);

  test.afterAll(async () => {
    await stopServer(server);
    if (tempDir) await rm(tempDir, { recursive: true, force: true });
  });

  test("editor and viewer sessions enforce the Edition role boundary", async ({ browser }) => {
    let createdPageID = "";
    const adminContext = await browser.newContext();
    const adminPage = await adminContext.newPage();
    await adminPage.goto(`${baseURL}/login`);
    await adminPage.getByLabel("Email address").fill("admin@example.test");
    await adminPage.getByLabel("Password").fill(adminPassword);
    await adminPage.getByRole("button", { name: "Sign in" }).click();
    await expect(adminPage).toHaveURL(/\/admin(?:\?.*)?$/);
    const adminNavPaths = await adminPage.locator('.studio-sidebar a[href^="/admin"]').evaluateAll((links) =>
      [...new Set(links.map((link) => new URL((link as HTMLAnchorElement).href).pathname))],
    );
    expect(adminNavPaths.length).toBeGreaterThan(0);
    for (const path of adminNavPaths) {
      const response = await adminPage.goto(`${baseURL}${path}`);
      expect(response?.status(), `admin navigation link ${path} must open`).toBe(200);
    }
    await adminPage.goto(`${baseURL}/admin/users`);
    const managementTargets: Record<string, { id: string; role: string; permissions: string; disable: string }> = {};
    for (const email of ["editor@example.test", "viewer@example.test"]) {
      const row = adminPage.locator(".studio-table tbody tr").filter({ has: adminPage.getByText(email, { exact: true }) });
      const roleAction = await row.locator('form[action$="/role"]').getAttribute("action");
      const permissionAction = await row.locator('form[action$="/permissions"]').getAttribute("action");
      const disableAction = await row.locator('form[action$="/disable"]').getAttribute("action");
      expect(roleAction).toBeTruthy();
      expect(permissionAction).toBeTruthy();
      expect(disableAction).toBeTruthy();
      const id = new URL(roleAction!, baseURL).pathname.split("/").at(-2)!;
      expect(new URL(permissionAction!, baseURL).pathname.split("/").at(-2)).toBe(id);
      expect(new URL(disableAction!, baseURL).pathname.split("/").at(-2)).toBe(id);
      managementTargets[email] = { id, role: roleAction!, permissions: permissionAction!, disable: disableAction! };
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
        const createForm = page.locator('form[action="/admin/pages"]');
        await page.goto(`${baseURL}${account.allowedPath}`);
        await createForm.locator('[name="slug"]').fill("role-editor-page");
        await createForm.locator('[name="title"]').fill("Editor session page");
        await createForm.locator(".richtext-editor [contenteditable=true]").fill("Created by an authenticated Edition editor.");
        await createForm.getByRole("button", { name: "Taslak oluştur" }).click();
        await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);
        createdPageID = new URL(page.url()).pathname.split("/").at(-1) || "";
      } else {
        expect(allowed?.status()).toBe(403);
        expect((await page.goto(`${baseURL}/admin/media`))?.status()).toBe(403);
        expect(createdPageID).not.toBe("");
        expect((await page.goto(`${baseURL}/admin/pages`))?.status()).toBe(200);
        await expect(page.locator('.studio-sidebar a[href="/admin/pages"]')).toBeVisible();
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
        await expect(page.locator(`a[href="/admin/pages/${createdPageID}"]`)).toBeVisible();
        expect((await page.goto(`${baseURL}/admin/pages/${createdPageID}`))?.status()).toBe(200);
        await expect(page.locator('form[action^="/admin/pages/"]')).toHaveCount(0);
        await expect(page.locator('iframe.readonly-content-preview')).toBeVisible();
        expect((await page.request.get(`${baseURL}/api/pages/${createdPageID}`)).status()).toBe(200);
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
    const backupDownload = page.waitForEvent("download");
    await backupForm.getByRole("button", { name: "Yedeği oluştur ve indir" }).click();
    const [backupResponse, download] = await Promise.all([downloadResponse, backupDownload]);
    expect(backupResponse.status()).toBe(200);
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
    const token = execFileSync("go", ["run", "./tests/e2e/support/seedinvite", "--database-dialect", dialect, "--database-dsn", database, "--email", "invited-editor@example.test"], { cwd: repoRoot, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
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
