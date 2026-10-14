import { test, expect, type Page } from "@playwright/test";
import type { ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { runCommand, spawnServer, stopServer } from "./server-process";

// Drives the templ-rendered page and news admin screens end to end: create,
// edit, publish and schedule through the real forms, then check the lists and
// the public site. Every admin page must load without console errors (CSP
// violations, blocked resources, script errors).
const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe("page and news admin screens", () => {
  test.beforeAll(async () => {
    test.setTimeout(180_000);
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-pages-news-"));
    const databaseDSN = path.join(tempDir, "site.sqlite");
    const hydroBinary = path.join(tempDir, process.platform === "win32" ? "hydro.exe" : "hydro");
    runCommand("go", ["build", "-o", hydroBinary, "./cmd/hydro"], { cwd: repoRoot, stdio: "pipe" });
    const db = ["--database-dialect", "sqlite", "--database-dsn", databaseDSN];
    runCommand(hydroBinary, ["migrate", ...db], { cwd: tempDir, stdio: "pipe" });
    runCommand(hydroBinary, ["create-admin", "--email", "admin@example.test", "--password-stdin", ...db], { cwd: tempDir, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawnServer(hydroBinary, ["serve", "--dev", "--address", `127.0.0.1:${port}`, ...db], {
      cwd: tempDir,
      stdio: "ignore",
      env: { ...process.env, HYDRO_PUBLIC_ORIGIN: baseURL, HYDRO_MEDIA_DIR: path.join(tempDir, "media"), HYDRO_REDIS_URL: "" },
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

  test("pages and news are created, edited, published and scheduled through the admin forms", async ({ page }) => {
    test.setTimeout(150_000);
    const consoleErrors = watchConsole(page);
    await page.goto(`${baseURL}/login`);
    await page.locator('input[name="email"]').fill("admin@example.test");
    await page.locator('input[name="password"]').fill(adminPassword);
    await page.locator('button[type="submit"]').first().click();
    await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
    // Skip the onboarding tour like a returning admin. Set once instead of with
    // addInitScript: an init script also runs in the editors' script-less
    // preview frames, where Chrome reports it as blocked.
    await page.evaluate(() => {
      localStorage.setItem("hydro-onboarding-dismissed", "1");
      localStorage.setItem("hydro-onboarding-dismissed:admin", "1");
    });

    // Page: create a draft through /admin/pages/new.
    await page.goto(`${baseURL}/admin/pages/new`);
    const createPage = page.locator('form[action="/admin/pages"]');
    await createPage.locator('[name="slug"]').fill("e2e-about");
    await createPage.locator('[name="title"]').fill("E2E about page");
    await createPage.locator(".richtext-editor [contenteditable=true]").fill("Written by the admin pages test.");
    await createPage.locator('[name="tags"]').fill("e2e, about");
    await createPage.locator('.page-editor-submit-bar button[type="submit"]').click();
    await expect(page).toHaveURL(/\/admin\/pages\/[0-9a-f]+$/);
    const pagePath = new URL(page.url()).pathname;

    // Edit it: the block editor and the draft form are on the page.
    const editPage = page.locator(`form[action="${pagePath}"]`);
    await expect(editPage.locator('[data-block-editor] [data-block-row]')).toHaveCount(1);
    await expect(editPage.locator('[name="tags"]')).toHaveValue("about, e2e");
    await editPage.locator('[name="title"]').fill("E2E about page, edited");
    await editPage.locator('[name="block_type_0"]').selectOption("cta");
    await editPage.locator('[name="block_title_0"]').fill("Talk to us");
    await editPage.locator('.page-editor-submit-bar button[type="submit"]').click();
    await expect(page).toHaveURL(new RegExp(`${pagePath}\\?saved=1$`));
    await expect(editPage.locator('[name="title"]')).toHaveValue("E2E about page, edited");
    await expect(editPage.locator('[name="block_title_0"]')).toHaveValue("Talk to us");

    // Publish it and find it in the list and on the public site.
    await page.locator(`form[action="${pagePath}/publish"] button[type="submit"]`).click();
    await expect(page).toHaveURL(new RegExp(`${pagePath}$`));
    await page.goto(`${baseURL}/admin/pages`);
    const pageRow = page.locator("#admin-content-list tr", { hasText: "e2e-about" });
    await expect(pageRow).toHaveAttribute("data-status-row", "published");
    const publicPage = await page.request.get(`${baseURL}/public/pages/tr/e2e-about/html`);
    expect(publicPage.status()).toBe(200);
    const publicPageHTML = await publicPage.text();
    expect(publicPageHTML).toContain("E2E about page, edited");
    expect(publicPageHTML).toContain("Talk to us");

    // News: create two drafts through /admin/news/new; the header title follows
    // the title field.
    const createNews = async (title: string, slug: string): Promise<string> => {
      await page.goto(`${baseURL}/admin/news/new`);
      const form = page.locator("#news-form");
      await form.locator('[name="title"]').fill(title);
      await expect(page.locator(".editor-live-title")).toHaveText(title);
      await form.locator('[name="slug"]').fill(slug);
      await form.locator(".richtext-editor [contenteditable=true]").fill(`Body of ${title}.`);
      await page.locator('button[form="news-form"][type="submit"]').click();
      await expect(page).toHaveURL(/\/admin\/news\/[0-9a-f]+$/);
      return new URL(page.url()).pathname;
    };
    const publishedPath = await createNews("E2E launch story", "e2e-launch");
    const scheduledPath = await createNews("E2E future story", "e2e-future");

    // Edit the first story, then publish it.
    await page.goto(`${baseURL}${publishedPath}`);
    const editNews = page.locator("#news-edit-form");
    await editNews.locator('[name="title"]').fill("E2E launch story, edited");
    await expect(page.locator(".editor-live-title")).toHaveText("E2E launch story, edited");
    await editNews.locator('[name="author_name"]').fill("E2E Desk");
    await page.locator('button[form="news-edit-form"][type="submit"]').click();
    await expect(page).toHaveURL(new RegExp(`${publishedPath}\\?saved=1$`));
    await expect(editNews.locator('[name="author_name"]')).toHaveValue("E2E Desk");
    await page.locator('button[form="news-publish-form"]').click();
    await expect(page).toHaveURL(new RegExp(`${publishedPath}$`));
    await expect(page.locator(".editor-side-status strong").first()).toHaveText("published");
    await expect(page.locator("button[data-copy-text]")).toHaveAttribute("data-copy-text", `${baseURL}/public/news/tr/e2e-launch/html`);
    // Schedule the second story a year ahead.
    await page.goto(`${baseURL}${scheduledPath}`);
    const nextYear = new Date().getUTCFullYear() + 1;
    await page.locator("#news-scheduled-at").fill(`${nextYear}-06-01T09:30`);
    await page.locator('button[form="news-schedule-form"]').click();
    await expect(page).toHaveURL(new RegExp(`${scheduledPath}$`));
    await expect(page.locator(".editor-side-status strong").first()).toHaveText("scheduled");

    await page.goto(`${baseURL}/admin/news`);
    await expect(page.locator("#admin-content-list tr", { hasText: "e2e-launch" })).toHaveAttribute("data-status-row", "published");
    await expect(page.locator("#admin-content-list tr", { hasText: "e2e-future" })).toHaveAttribute("data-status-row", "scheduled");

    const publicNews = await page.request.get(`${baseURL}/public/news/tr/e2e-launch/html`);
    expect(publicNews.status()).toBe(200);
    expect(await publicNews.text()).toContain("E2E launch story, edited");
    const publicList = await page.request.get(`${baseURL}/api/news?locale=tr`);
    const slugs = (await publicList.json()).map((item: { slug: string }) => item.slug);
    expect(slugs).toContain("e2e-launch");
    expect(slugs).not.toContain("e2e-future");

    expect(consoleErrors).toEqual([]);
  });
});

function watchConsole(page: Page): string[] {
  const errors: string[] = [];
  page.on("console", (message) => {
    // Playwright's trace recorder injects scripts into every frame; Chrome
    // reports them as blocked in the editors' script-less preview frames.
    // The frames themselves never contain scripts.
    if (message.type() === "error" && !/^Blocked script execution in 'about:(srcdoc|blank)'/.test(message.text())) {
      errors.push(`${page.url()}: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => errors.push(`${page.url()}: ${error.message}`));
  return errors;
}

async function unusedPort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const probe = net.createServer();
    probe.once("error", reject);
    probe.listen(0, "127.0.0.1", () => {
      const address = probe.address();
      probe.close(() => (typeof address === "object" && address ? resolve(address.port) : reject(new Error("no port"))));
    });
  });
}
