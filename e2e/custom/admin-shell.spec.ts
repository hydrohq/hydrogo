import { test, expect, type Page } from "@playwright/test";
import type { ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { runCommand, spawnServer, stopServer } from "./server-process";

// The admin frame (internal/ui/shell): templ-rendered, role-filtered sidebar
// and Alpine.js (CSP build) for the user menu, notifications, the light/dark
// switch and the mobile drawer. Every page must load without console or CSP
// errors.
const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe.configure({ mode: "serial" });

test.describe("admin shell", () => {
  test.beforeAll(async () => {
    test.setTimeout(180_000);
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-admin-shell-"));
    const databaseDSN = path.join(tempDir, "site.sqlite");
    const hydroBinary = path.join(tempDir, process.platform === "win32" ? "hydro.exe" : "hydro");
    runCommand("go", ["build", "-o", hydroBinary, "./cmd/hydro"], { cwd: repoRoot, stdio: "pipe" });
    const db = ["--database-dialect", "sqlite", "--database-dsn", databaseDSN];
    runCommand(hydroBinary, ["migrate", ...db], { cwd: tempDir, stdio: "pipe" });
    runCommand(hydroBinary, ["create-admin", "--email", "admin@example.test", "--password-stdin", ...db], { cwd: tempDir, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    // editor@example.test and viewer@example.test, same password.
    runCommand("go", ["run", "./tests/e2e/support/seedroles", "--database-dsn", databaseDSN], { cwd: repoRoot, stdio: "pipe" });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawnServer(hydroBinary, ["serve", "--dev", "--address", `127.0.0.1:${port}`, ...db], {
      cwd: tempDir,
      stdio: "ignore",
      env: {
        ...process.env,
        HYDRO_CONTENT_TYPES_DIR: path.join(repoRoot, "content-types"),
        HYDRO_PUBLIC_ORIGIN: baseURL,
        HYDRO_MEDIA_DIR: path.join(tempDir, "media"),
        HYDRO_REDIS_URL: "",
      },
    });
    const deadline = Date.now() + 45_000;
    while (Date.now() < deadline) {
      if (server.exitCode !== null) throw new Error(`HydroGo server exited with ${server.exitCode}`);
      try {
        if ((await fetch(`${baseURL}/healthz`)).ok) return;
      } catch {
        // still starting
      }
      await new Promise((resolve) => setTimeout(resolve, 200));
    }
    throw new Error("HydroGo server did not become healthy within 45 seconds");
  });

  test.afterAll(async () => {
    await stopServer(server);
    if (tempDir) await rm(tempDir, { recursive: true, force: true });
  });

  test("admin: user menu, notifications, theme, sidebar navigation", async ({ page }) => {
    test.setTimeout(120_000);
    const problems = watchForErrors(page);
    await login(page, "admin@example.test");

    // Sidebar: the full admin navigation, content types listed under their entry.
    const sidebar = page.locator(".studio-sidebar nav");
    for (const href of ["/admin", "/admin/reports", "/admin/news", "/admin/pages", "/admin/resources", "/admin/resources/faq", "/admin/members", "/admin/users", "/admin/media", "/admin/developer"]) {
      await expect(sidebar.locator(`a[href="${href}"]`)).toBeVisible();
    }
    await expect(sidebar.locator('a[href="/admin"]')).toHaveAttribute("aria-current", "page");

    // User menu: click opens, Escape closes and returns focus, outside click closes.
    const userButton = page.locator("#user-menu-btn");
    const userMenu = page.locator("#user-dropdown");
    await expect(userMenu).toBeHidden();
    await userButton.click();
    await expect(userMenu).toBeVisible();
    await expect(userButton).toHaveAttribute("aria-expanded", "true");
    await expect(userMenu).toContainText("admin@example.test");
    await expect(userMenu.locator('a[href="/admin/permissions"]')).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(userMenu).toBeHidden();
    await expect(userButton).toHaveAttribute("aria-expanded", "false");
    await expect(userButton).toBeFocused();
    await userButton.press("ArrowDown");
    await expect(userMenu).toBeVisible();
    await expect(userMenu.locator('a[href="/admin/account"]')).toBeFocused();
    await page.locator(".studio-heading h1").click();
    await expect(userMenu).toBeHidden();

    // Notifications panel.
    const bell = page.locator(".studio-notification-bell");
    const panel = page.locator("#studio-notifications-panel");
    await bell.click();
    await expect(panel).toBeVisible();
    await expect(bell).toHaveAttribute("aria-expanded", "true");
    await expect(panel.locator('a[href="/admin/notifications"]')).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(panel).toBeHidden();
    await expect(bell).toBeFocused();

    // Light/dark switch persists across reloads.
    const html = page.locator("html");
    const before = await html.getAttribute("data-theme");
    const after = before === "dark" ? "light" : "dark";
    const toggle = page.locator("#theme-toggle-btn");
    await expect(toggle).toHaveAccessibleName(/tema; değiştirmek için tıklayın/);
    await toggle.click();
    await expect(html).toHaveAttribute("data-theme", after);
    await expect(page.locator(".studio-toast").last()).toContainText("tema uygulandı.");
    await page.reload();
    await expect(html).toHaveAttribute("data-theme", after);
    await waitForAlpine(page);
    await page.locator("#theme-toggle-btn").click();
    await expect(html).toHaveAttribute("data-theme", before ?? "light");

    // Sidebar links navigate through the studio router.
    await sidebar.locator('a[href="/admin/pages"]').click();
    await expect(page).toHaveURL(`${baseURL}/admin/pages`);
    await expect(page.locator(".studio-heading h1")).toHaveText("Sayfalar");
    await expect(sidebar.locator('a[href="/admin/pages"]')).toHaveClass(/is-active/);
    await sidebar.locator('a[href="/admin/resources/faq"]').click();
    await expect(page).toHaveURL(`${baseURL}/admin/resources/faq`);
    await expect(sidebar.locator('a[href="/admin/resources/faq"]')).toHaveClass(/is-active/);
    await sidebar.locator('a[href="/admin/news"]').click();
    await expect(page).toHaveURL(`${baseURL}/admin/news`);
    // The topbar is re-rendered on every navigation; its components still work.
    await page.locator("#user-menu-btn").click();
    await expect(page.locator("#user-dropdown")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.locator("#user-dropdown")).toBeHidden();

    expect(problems, problems.join("\n")).toEqual([]);
  });

  test("mobile drawer opens, closes and follows links at 390px", async ({ page }) => {
    test.setTimeout(90_000);
    const problems = watchForErrors(page);
    await page.setViewportSize({ width: 390, height: 844 });
    await login(page, "admin@example.test");
    const toggle = page.getByRole("button", { name: "Menüyü aç" });
    const body = page.locator("body");
    await expect(toggle).toHaveAttribute("aria-expanded", "false");
    await toggle.click();
    await expect(toggle).toHaveAttribute("aria-expanded", "true");
    await expect(body).toHaveClass(/studio-mobile-nav-open/);
    await expect(page.locator(".studio-sidebar a[href='/admin/media']")).toBeInViewport();
    await page.keyboard.press("Escape");
    await expect(body).not.toHaveClass(/studio-mobile-nav-open/);
    await expect(toggle).toHaveAttribute("aria-expanded", "false");
    await expect(toggle).toBeFocused();
    await toggle.click();
    await page.locator(".studio-mobile-backdrop").click({ position: { x: 370, y: 400 } });
    await expect(body).not.toHaveClass(/studio-mobile-nav-open/);
    await toggle.click();
    await page.locator(".studio-sidebar a[href='/admin/media']").click();
    await expect(page).toHaveURL(`${baseURL}/admin/media`);
    await expect(body).not.toHaveClass(/studio-mobile-nav-open/);
    // The re-rendered toggle still drives the shared drawer state.
    await page.getByRole("button", { name: "Menüyü aç" }).click();
    await expect(body).toHaveClass(/studio-mobile-nav-open/);
    expect(problems, problems.join("\n")).toEqual([]);
  });

  test("editor sees only the navigation its role allows", async ({ page }) => {
    test.setTimeout(90_000);
    const problems = watchForErrors(page);
    await login(page, "editor@example.test");
    await expect(page.locator("body")).toHaveAttribute("data-studio-role", "editor");
    const hrefs = await page.locator(".studio-sidebar nav a").evaluateAll((links) => links.map((link) => link.getAttribute("href")));
    expect(hrefs).toEqual(["/admin", "/admin/news", "/admin/pages", "/admin/media"]);
    for (const label of ["GENEL", "İÇERİK", "SİSTEM"]) {
      await expect(page.locator(".studio-sidebar .sidebar-section-label", { hasText: label })).toBeVisible();
    }
    await expect(page.locator(".studio-sidebar .sidebar-section-label", { hasText: "KULLANICILAR" })).toHaveCount(0);
    await page.locator("#user-menu-btn").click();
    await expect(page.locator("#user-dropdown")).toContainText("Editör Hesabı");
    await expect(page.locator('#user-dropdown a[href="/admin/permissions"]')).toHaveCount(0);
    for (const href of hrefs) {
      const response = await page.goto(`${baseURL}${href}`);
      expect(response?.status(), `editor navigation link ${href} must open`).toBe(200);
    }
    expect(problems, problems.join("\n")).toEqual([]);
  });
});

async function login(page: Page, email: string) {
  await page.addInitScript(() => {
    localStorage.setItem("hydro-onboarding-dismissed", "1");
    for (const role of ["admin", "editor", "viewer"]) localStorage.setItem(`hydro-onboarding-dismissed:${role}`, "1");
  });
  await page.goto(`${baseURL}/login`);
  await page.locator('input[name="email"]').fill(email);
  await page.locator('input[name="password"]').fill(adminPassword);
  await page.locator('button[type="submit"]').first().click();
  await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
  await waitForAlpine(page);
}

// Alpine starts after the deferred scripts; interacting earlier would click a
// button with no handler yet.
async function waitForAlpine(page: Page) {
  await page.waitForFunction(() => Boolean((document.querySelector(".user-menu-wrap") as any)?._x_dataStack));
}

// Collects JavaScript errors, console errors and CSP violations.
function watchForErrors(page: Page): string[] {
  const problems: string[] = [];
  page.on("pageerror", (error) => problems.push(`pageerror: ${error.message}`));
  page.on("console", (message) => {
    if (message.type() === "error") problems.push(`console: ${message.text()}`);
  });
  page.on("response", (response) => {
    if (response.url().startsWith(baseURL) && response.status() >= 400 && !response.url().includes("/favicon")) problems.push(`HTTP ${response.status()} ${response.url()}`);
  });
  void page.addInitScript(() => {
    document.addEventListener("securitypolicyviolation", (event) => console.error(`CSP violation: ${event.violatedDirective} ${event.blockedURI}`));
  });
  return problems;
}

async function unusedPort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const listener = net.createServer();
    listener.unref();
    listener.on("error", reject);
    listener.listen(0, "127.0.0.1", () => {
      const address = listener.address();
      listener.close(() => (typeof address === "object" && address ? resolve(address.port) : reject(new Error("no port"))));
    });
  });
}
