import { test, expect, request, type ChildProcess } from "@playwright/test";
import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { runCommand } from "./server-process";

const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
const masterKey = "12345678901234567890123456789012";
const backupSigningSeed = "ab".repeat(32);
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe("Admin Studio MFA self-service", () => {
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-admin-mfa-"));
    const dialect = process.env.HYDRO_E2E_DATABASE_DIALECT ?? "sqlite";
    const database = process.env.HYDRO_E2E_DATABASE_DSN ?? path.join(tempDir, "edition.sqlite");
    if (dialect === "postgres" && !process.env.HYDRO_E2E_DATABASE_DSN) {
      throw new Error("HYDRO_E2E_DATABASE_DSN is required for the PostgreSQL Admin MFA run");
    }
    if (dialect === "postgres" && !process.env.HYDRO_PG_PRIVILEGED_ROLE) {
      throw new Error("HYDRO_PG_PRIVILEGED_ROLE must name the PostgreSQL backup operator role for the Admin MFA run");
    }
    const binary = path.join(tempDir, process.platform === "win32" ? "hydro.exe" : "hydro");
    const databaseArgs = ["--database-dialect", dialect, "--database-dsn", database];
    runCommand("go", ["run", "./cmd/hydro", "migrate", ...databaseArgs], { cwd: repoRoot, stdio: "pipe" });
    runCommand("go", ["run", "./cmd/hydro", "create-admin", "--email", "admin@example.test", "--password-stdin", ...databaseArgs], { cwd: repoRoot, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    runCommand("go", ["run", "./tests/e2e/support/seedroles", ...databaseArgs], { cwd: repoRoot, stdio: "pipe" });
    runCommand("go", ["build", "-o", binary, "./cmd/hydro"], { cwd: repoRoot, stdio: "pipe" });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawn(binary, ["serve", "--dev", ...databaseArgs, "--address", `127.0.0.1:${port}`], { cwd: repoRoot, stdio: "ignore", env: { ...process.env, HYDRO_MFA_MASTER_KEY: masterKey, HYDRO_BACKUP_SIGNING_SEED: backupSigningSeed, HYDRO_MEDIA_DIR: path.join(tempDir, "media") } });
    const deadline = Date.now() + 45_000;
    while (Date.now() < deadline) {
      if (server.exitCode !== null) throw new Error(`HydroGo server exited with ${server.exitCode}`);
      try {
        if ((await fetch(`${baseURL}/healthz`)).ok) return;
      } catch {
        // Wait for the isolated test server to start.
      }
      await new Promise((resolve) => setTimeout(resolve, 200));
    }
    throw new Error("HydroGo server did not become healthy within 45 seconds");
  }, 60_000);

  test.afterAll(async () => {
    if (server && server.exitCode === null) {
      server.kill("SIGTERM");
      await new Promise<void>((resolve) => {
        const timeout = setTimeout(resolve, 5_000);
        server?.once("exit", () => { clearTimeout(timeout); resolve(); });
      });
    }
    if (tempDir) await rm(tempDir, { recursive: true, force: true });
  });

  test("sets up, verifies and disables MFA from account settings", async ({ browser }) => {
    test.setTimeout(240_000);
    const context = await browser.newContext();
    const page = await context.newPage();
    const browserIssues: string[] = [];
    let watchCSP = true;
    page.on("console", (message) => { if (watchCSP && message.type() === "error" && message.text().includes("Content Security Policy")) browserIssues.push(message.text()); });
    page.on("pageerror", (error) => browserIssues.push(error.stack || error.message));
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await page.goto(`${baseURL}/login`);
    await page.evaluate(() => localStorage.setItem("hydro-ui-theme-v2", "workbench-dark"));
    await page.goto(`${baseURL}/login`);
    await page.getByLabel("Email address").fill("admin@example.test");
    await page.getByLabel("Password").fill(adminPassword);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
    const onboardingTour = page.locator("#admin-onboarding-dialog");
    await expect(onboardingTour).toBeVisible();
    await onboardingTour.locator("[data-onboarding-dismiss]").click();

    browserIssues.length = 0;
    const accountResponse = await page.goto(`${baseURL}/admin/account`);
    expect(accountResponse?.headers()["cache-control"]).toContain("no-store");
    expect(accountResponse?.headers()["content-security-policy"]).toContain("script-src 'self'");
    expect(accountResponse?.headers()["content-security-policy"]).toContain("style-src 'self'");
    const setup = page.getByRole("button", { name: "Authenticator uygulamasını kur" });
    await expect(setup).toBeVisible();
    const a11y = await page.evaluate(async () => (window as any).axe.run(document, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
    }));
    expect(a11y.violations.map((violation: any) => violation.id)).toEqual([]);
    await page.setViewportSize({ width: 390, height: 844 });
    const mobileLayout = await page.evaluate(() => ({ width: window.innerWidth, contentWidth: document.documentElement.scrollWidth }));
    expect(mobileLayout.contentWidth).toBeLessThanOrEqual(mobileLayout.width);
    const mobileA11y = await page.evaluate(async () => (window as any).axe.run(document, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
    }));
    expect(mobileA11y.violations.map((violation: any) => violation.id)).toEqual([]);
    await page.locator("[data-mfa-settings]").screenshot({ path: path.join(tempDir, "admin-mfa-mobile.png") });
    await page.setViewportSize({ width: 1280, height: 900 });

    const enrollmentResponse = page.waitForResponse((response) => response.request().method() === "POST" && response.url().endsWith("/auth/me/mfa/enroll"));
    await setup.click();
    const enrolled = await enrollmentResponse;
    expect(enrolled.status()).toBe(201);
    expect(enrolled.headers()["cache-control"]).toBe("no-store");
    expect(enrolled.headers()["referrer-policy"]).toBe("no-referrer");
    const enrollmentPayload = await enrolled.json();
    expect(enrollmentPayload.Secret).toMatch(/^[A-Z2-7]+$/);
    expect(enrollmentPayload.RecoveryCodes).toHaveLength(8);
    await expect(page.locator("[data-mfa-secret]")).toHaveValue(/^[A-Z2-7]+$/);
    const secret = await page.locator("[data-mfa-secret]").inputValue();
    expect(secret).toBe(enrollmentPayload.Secret);
    expect(secret).toMatch(/^[A-Z2-7]+$/);
    await expect(page.locator("[data-mfa-recovery-codes] li")).toHaveCount(8);
    await expect(page.locator("[data-mfa-provisioning]")).toHaveAttribute("href", /^otpauth:\/\//);

    await page.getByLabel("Bu kodları güvenli bir yere kaydettim.").check();
    const enableCode = generateTOTP(secret);
    await page.getByLabel("Doğrulama kodu").fill(enableCode);
    const enableResponse = page.waitForResponse((response) => response.request().method() === "POST" && response.url().endsWith("/auth/me/mfa/enable"));
    await page.getByRole("button", { name: "Doğrula ve etkinleştir" }).click();
    const enableResult = await enableResponse;
    expect(enableResult.request().postData() || "").toContain(enableCode);
    if (enableResult.status() !== 204) throw new Error(`MFA enable returned ${enableResult.status()}: ${await enableResult.text()}`);
    await expect(page.getByRole("button", { name: "MFA'yı kapat" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Authenticator uygulamasını kur" })).toHaveCount(0);

    await page.locator("#user-menu-btn").click();
    await page.locator("#studio-logout-btn").click();
    await expect(page).toHaveURL(/\/login\?return_to=%2Fadmin|\/login\?return_to=\/admin/);
    await page.getByLabel("Email address").fill("admin@example.test");
    await page.getByLabel("Password").fill(adminPassword);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.locator("#mfa-challenge")).toBeVisible();
    await expect(page.locator("#login-email")).toBeHidden();
    await expect(page.locator("#login-password")).toBeHidden();
    expect((await page.request.get(`${baseURL}/auth/me`)).status()).toBe(401);
    const loginMFA = page.locator("#login-mfa-code");
    await expect(loginMFA).toBeEnabled();
    const assertLoginAccessibility = async (state: string, mode: string) => {
      const themeTokens = await page.evaluate(() => ({
        mode: document.documentElement.dataset.theme,
        accent: getComputedStyle(document.documentElement).getPropertyValue("--brand-accent").trim(),
        inputBackground: getComputedStyle(document.querySelector("#login-email")!).backgroundColor,
      }));
      const result = await page.evaluate(async () => (window as any).axe.run(document, {
        runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
      }));
      expect(result.violations.map((violation: any) => ({
        id: violation.id,
        impact: violation.impact,
        nodes: violation.nodes.map((node: any) => ({ target: node.target, summary: node.failureSummary })),
      })), `${state} login WCAG 2.1 A/AA violations in ${mode}; tokens=${JSON.stringify(themeTokens)}`).toEqual([]);
    };
    await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
    await assertLoginAccessibility("MFA challenge", "dark");
    await page.locator("#theme-toggle-btn").click();
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
    await assertLoginAccessibility("MFA challenge", "light");
    const invalidMFAResponse = page.waitForResponse((response) => response.url() === `${baseURL}/auth/login` && response.request().method() === "POST");
    await loginMFA.fill("000000");
    await page.getByRole("button", { name: "Sign in" }).click();
    expect((await invalidMFAResponse).status()).toBe(401);
    await expect(page.getByRole("alert")).toHaveText("Code could not be verified. Try the current code in your authenticator app.");
    await assertLoginAccessibility("MFA invalid-code error", "light");
    await page.locator("#theme-toggle-btn").click();
    await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
    await assertLoginAccessibility("MFA invalid-code error", "dark");
    await loginMFA.fill(generateTOTP(secret));
    await page.getByRole("button", { name: "Sign in" }).click();
    await page.waitForURL((url) => url.pathname === "/admin");
    expect((await page.request.get(`${baseURL}/auth/me`)).status()).toBe(200);

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
      const response = await page.goto(`${baseURL}${route}`);
      expect(response?.status(), `MFA-authenticated administrator must access ${route}`).toBe(200);
      const csp = response?.headers()["content-security-policy"] || "";
      const authorized = new Set([...csp.matchAll(/'sha256-([^']+)'/g)].map((match) => match[1]));
      const inlineScripts = await page.locator("script:not([src])").evaluateAll((scripts) => scripts.map((script) => script.textContent || "").filter((source) => source.trim().length > 0));
      for (const source of inlineScripts) {
        const digest = createHash("sha256").update(source, "utf8").digest("base64");
        expect(authorized.has(digest), `CSP for ${route} must authorize rendered inline script ${digest}`).toBe(true);
      }
    }

    await page.goto(`${baseURL}/admin/security`);
    const banAddress = "198.51.100.223";
    const banForm = page.locator('form[action="/admin/security/bans"]');
    await banForm.getByLabel("IP adresi veya CIDR").fill(banAddress);
    await banForm.getByLabel(/Süre/).fill("1h");
    await banForm.getByRole("button", { name: "IP engelle" }).click();
    await expect(page.getByText(`${banAddress}/32`, { exact: true })).toBeVisible();
    const unbanForm = page.locator('form[action="/admin/security/bans/remove"]');
    await unbanForm.getByLabel("Engeli kaldırılacak adres veya CIDR").fill(banAddress);
    await unbanForm.getByRole("button", { name: "Engeli kaldır" }).click();
    await expect(page.getByText(banAddress, { exact: true })).toHaveCount(0);

    await page.goto(`${baseURL}/admin/pages/new`);
    await page.getByLabel("Boş sayfa").check();
    const contentForm = page.locator('form[action="/admin/pages"]');
    await contentForm.locator('[name="slug"]').fill("mfa-admin-mutation");
    await contentForm.locator('[name="title"]').fill("MFA verified content mutation");
    await contentForm.locator(".richtext-editor [contenteditable=true]").fill("Created in an MFA-authenticated admin session.");
    await contentForm.getByRole("button", { name: "Taslak oluştur" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);

    await page.goto(`${baseURL}/admin/backup`);
    const backupForm = page.locator('form[action="/admin/backup"]');
    await backupForm.getByLabel(/Devam etmek için parolanızı doğrulayın|Confirm your password to continue/i).fill(adminPassword);
    const backupResponse = page.waitForResponse((response) => response.url() === `${baseURL}/admin/backup` && response.request().method() === "POST");
    const backupDownload = page.waitForEvent("download");
    await backupForm.getByRole("button", { name: /Yedeği oluştur ve indir|Create and download backup/i }).click();
    const [backupResult, backupFile] = await Promise.all([backupResponse, backupDownload]);
    expect(backupResult.status()).toBe(200);
    expect(backupResult.headers()["cache-control"]).toContain("no-store");
    expect(backupFile.suggestedFilename()).toMatch(/^hydrogo-backup-.*\.tar\.gz$/);

    const editorContext = await browser.newContext();
    const editorPage = await editorContext.newPage();
    await editorPage.goto(`${baseURL}/login`);
    await editorPage.getByLabel("Email address").fill("editor@example.test");
    await editorPage.getByLabel("Password").fill(adminPassword);
    await editorPage.getByRole("button", { name: "Sign in" }).click();
    await expect(editorPage).toHaveURL(/\/admin(?:\?.*)?$/);
    expect((await editorPage.goto(`${baseURL}/admin/pages/new`))?.status()).toBe(200);

    await page.goto(`${baseURL}/admin/users`);
    const editorRow = page.locator(".studio-table tbody tr").filter({ has: page.getByText("editor@example.test", { exact: true }) });
    const editorPermissionForm = editorRow.locator('form[action$="/permissions"]');
    const denyToken = await editorPermissionForm.locator('[name="_csrf"]').inputValue();
    const permissionAction = await editorPermissionForm.getAttribute("action");
    expect(permissionAction).toBeTruthy();
    const deniedWrite = await page.request.post(new URL(permissionAction!, baseURL).href, {
      form: { _csrf: denyToken, "perm_admin.content.write": "deny" },
    });
    expect(deniedWrite.status()).toBe(204);
    expect((await editorPage.request.get(`${baseURL}/auth/me`)).status()).toBe(401);
    await page.reload();
    await expect(editorRow.locator('[name="perm_admin.content.write"]')).toHaveValue("deny");

    const grantForm = editorRow.locator('form[action$="/permissions"]');
    const grantToken = await grantForm.locator('[name="_csrf"]').inputValue();
    const grantWrite = await page.request.post(new URL(permissionAction!, baseURL).href, {
      form: { _csrf: grantToken, "perm_admin.content.write": "allow" },
    });
    expect(grantWrite.status()).toBe(204);
    await page.reload();
    await expect(editorRow.locator('[name="perm_admin.content.write"]')).toHaveValue("allow");
    await editorContext.close();

    const viewerContext = await browser.newContext();
    const viewerPage = await viewerContext.newPage();
    await viewerPage.goto(`${baseURL}/login`);
    await viewerPage.getByLabel("Email address").fill("viewer@example.test");
    await viewerPage.getByLabel("Password").fill(adminPassword);
    const viewerLoginResponse = viewerPage.waitForResponse((response) => response.url() === `${baseURL}/auth/login` && response.request().method() === "POST");
    await viewerPage.getByRole("button", { name: "Sign in" }).click();
    let viewerLogin = await viewerLoginResponse;
    if (viewerLogin.status() === 429) {
      const retryAfterSeconds = Number(viewerLogin.headers()["retry-after"] ?? "60");
      await viewerPage.waitForTimeout((retryAfterSeconds + 1) * 1000);
      const retryResponse = viewerPage.waitForResponse((response) => response.url() === `${baseURL}/auth/login` && response.request().method() === "POST");
      await viewerPage.getByRole("button", { name: "Sign in" }).click();
      viewerLogin = await retryResponse;
    }
    expect(viewerLogin.status(), "viewer login must recover after the configured IP rate-limit window").toBe(200);
    await expect(viewerPage).toHaveURL(/\/admin(?:\?.*)?$/);
    expect((await viewerPage.request.get(`${baseURL}/auth/me`)).status()).toBe(200);

    await page.goto(`${baseURL}/admin/users`);
    const viewerRow = page.locator(".studio-table tbody tr").filter({ has: page.getByText("viewer@example.test", { exact: true }) });
    const disableForm = viewerRow.locator('form[action$="/disable"]');
    const disableToken = await disableForm.locator('[name="_csrf"]').inputValue();
    const disableAction = await disableForm.getAttribute("action");
    expect(disableAction).toBeTruthy();
    const disableViewer = await page.request.post(new URL(disableAction!, baseURL).href, { form: { _csrf: disableToken } });
    expect(disableViewer.status()).toBe(204);
    expect((await viewerPage.request.get(`${baseURL}/auth/me`)).status()).toBe(401);
    await viewerContext.close();

    watchCSP = true;
    browserIssues.length = 0;
    await page.goto(`${baseURL}/admin/account`);
    await page.getByLabel("Güncel doğrulama veya kurtarma kodu").fill("000000");
    await page.getByRole("button", { name: "MFA'yı kapat" }).click();
    await expect(page.getByRole("alert")).toContainText("Kod geçersiz");
    await expect(page.getByRole("button", { name: "MFA'yı kapat" })).toBeVisible();

    await page.getByLabel("Güncel doğrulama veya kurtarma kodu").fill(generateTOTP(secret));
    const disableResponse = page.waitForResponse((response) => response.request().method() === "POST" && response.url().endsWith("/auth/me/mfa/disable"));
    await page.getByRole("button", { name: "MFA'yı kapat" }).click();
    expect((await disableResponse).status()).toBe(204);
    await expect(page.getByRole("button", { name: "Authenticator uygulamasını kur" })).toBeVisible();
    expect(browserIssues).toEqual([]);
    await context.close();

    const rateLimitContext = await browser.newContext();
    const rateLimitPage = await rateLimitContext.newPage();
    await rateLimitPage.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await rateLimitPage.goto(`${baseURL}/login`);
    await rateLimitPage.evaluate(() => localStorage.setItem("hydro-ui-theme-v2", "workbench-light"));
    await rateLimitPage.goto(`${baseURL}/login?lang=en`);
    await expect(rateLimitPage.locator("html")).toHaveAttribute("data-theme", "light");
    const exhaustLoginLimit = async () => {
      const limiterContext = await request.newContext();
      const csrfResponse = await limiterContext.get(`${baseURL}/csrf-token`);
      expect(csrfResponse.status()).toBe(200);
      const { token } = await csrfResponse.json();
      for (let attempt = 0; attempt < 6; attempt++) {
        const response = await limiterContext.post(`${baseURL}/auth/login`, {
          form: { _csrf: token, email: "admin@example.test", password: adminPassword },
          headers: { Accept: "application/json", "X-CSRF-Token": token },
        });
        if (response.status() === 429) {
          expect(response.headers()["retry-after"]).toBe("60");
          expect(response.headers()["cache-control"]).toContain("no-store");
          await limiterContext.dispose();
          return;
        }
        expect(response.status(), "valid credentials should consume the login limiter before it is exhausted").toBe(200);
      }
      await limiterContext.dispose();
      throw new Error("login limiter did not return HTTP 429 within six attempts");
    };
    await exhaustLoginLimit();
    const englishLimitResponse = rateLimitPage.waitForResponse((response) => response.url() === `${baseURL}/auth/login` && response.request().method() === "POST");
    await rateLimitPage.getByLabel("Email address").fill("admin@example.test");
    await rateLimitPage.getByLabel("Password").fill(adminPassword);
    await rateLimitPage.getByRole("button", { name: "Sign in" }).click();
    expect((await englishLimitResponse).status()).toBe(429);
    await expect(rateLimitPage.getByRole("alert")).toHaveText("Too many attempts. Please wait and try again.");
    const englishLimitA11y = await rateLimitPage.evaluate(async () => (window as any).axe.run(document, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
    }));
    expect(englishLimitA11y.violations.map((violation: any) => violation.id)).toEqual([]);

    await rateLimitPage.waitForTimeout(61_000);
    await rateLimitPage.evaluate(() => localStorage.setItem("hydro-ui-theme-v2", "workbench-dark"));
    await rateLimitPage.goto(`${baseURL}/login?lang=tr`);
    await expect(rateLimitPage.locator("html")).toHaveAttribute("data-theme", "dark");
    await exhaustLoginLimit();
    const turkishLimitResponse = rateLimitPage.waitForResponse((response) => response.url() === `${baseURL}/auth/login` && response.request().method() === "POST");
    await rateLimitPage.getByLabel("E-posta Adresi").fill("admin@example.test");
    await rateLimitPage.getByLabel("Şifre").fill(adminPassword);
    await rateLimitPage.getByRole("button", { name: "Giriş Yap" }).click();
    expect((await turkishLimitResponse).status()).toBe(429);
    await expect(rateLimitPage.getByRole("alert")).toHaveText("Çok fazla başarısız deneme yapıldı. Lütfen biraz bekleyin.");
    const turkishLimitA11y = await rateLimitPage.evaluate(async () => (window as any).axe.run(document, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
    }));
    expect(turkishLimitA11y.violations.map((violation: any) => violation.id)).toEqual([]);
    await rateLimitContext.close();
  });
});

function generateTOTP(secret: string): string {
  return runCommand("go", ["run", "./tests/e2e/support/totp-code"], { cwd: repoRoot, input: secret, stdio: ["pipe", "pipe", "pipe"] }).trim();
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
