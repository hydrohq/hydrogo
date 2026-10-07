import { test, expect, type ChildProcess } from "@playwright/test";
import { spawn, execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";

const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
const masterKey = "12345678901234567890123456789012";
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe("Admin Studio MFA self-service", () => {
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-admin-mfa-"));
    const database = path.join(tempDir, "edition.sqlite");
    const binary = path.join(tempDir, "hydro");
    execFileSync("go", ["run", "./cmd/hydro", "migrate", "--database-dialect", "sqlite", "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
    execFileSync("go", ["run", "./cmd/hydro", "create-admin", "--email", "admin@example.test", "--password-stdin", "--database-dialect", "sqlite", "--database-dsn", database], { cwd: repoRoot, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    execFileSync("go", ["build", "-o", binary, "./cmd/hydro"], { cwd: repoRoot, stdio: "pipe" });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawn(binary, ["serve", "--dev", "--database-dialect", "sqlite", "--database-dsn", database, "--address", `127.0.0.1:${port}`], { cwd: repoRoot, stdio: "ignore", env: { ...process.env, HYDRO_MFA_MASTER_KEY: masterKey, HYDRO_MEDIA_DIR: path.join(tempDir, "media") } });
    const deadline = Date.now() + 45_000;
    while (Date.now() < deadline) {
      if (server.exitCode !== null) throw new Error(`HydroGo server exited with ${server.exitCode}`);
      try {
        if ((await fetch(`${baseURL}/healthz`)).ok) return;
      } catch {
        // Wait for the isolated SQLite server to start.
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
    test.setTimeout(120_000);
    const context = await browser.newContext();
    const page = await context.newPage();
    const browserIssues: string[] = [];
    let watchCSP = true;
    page.on("console", (message) => { if (watchCSP && message.type() === "error" && message.text().includes("Content Security Policy")) browserIssues.push(message.text()); });
    page.on("pageerror", (error) => browserIssues.push(error.stack || error.message));
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
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
    await page.locator("[data-mfa-settings]").screenshot({ path: path.join(repoRoot, "e2e/artifacts/admin-mfa-mobile.png") });
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
    expect((await page.request.get(`${baseURL}/auth/me`)).status()).toBe(401);
    const loginMFA = page.locator("#login-mfa-code");
    await expect(loginMFA).toBeEnabled();
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
  });
});

function generateTOTP(secret: string): string {
  return execFileSync("go", ["run", "./tests/e2e/support/totp-code"], { cwd: repoRoot, input: secret, encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] }).trim();
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
