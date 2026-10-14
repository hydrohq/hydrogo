import { test, expect, type Page } from "@playwright/test";
import type { ChildProcess } from "node:child_process";
import { mkdtemp, rm, writeFile } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { runCommand, spawnServer, stopServer } from "./server-process";

// End-to-end check of the templ-rendered site and media admin screens: their
// behaviour now lives in static/admin/js, so these flows also prove the
// scripts load under the strict CSP.
const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
const fixture = path.join(repoRoot, "e2e/custom/fixtures/tiny.png");
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe.configure({ mode: "serial" });

test.describe("site and media admin", () => {
  test.beforeAll(async () => {
    test.setTimeout(180_000);
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-site-media-"));
    const config = path.join(tempDir, "hydro.yaml");
    await writeFile(config, "modules: [core, auth, members, pages, media, site, news, collections, forms]\n");
    const hydroBinary = path.join(tempDir, process.platform === "win32" ? "hydro.exe" : "hydro");
    runCommand("go", ["build", "-o", hydroBinary, "./cmd/hydro"], { cwd: repoRoot, stdio: "pipe" });
    const flags = ["--config", config, "--database-dialect", "sqlite", "--database-dsn", path.join(tempDir, "site.sqlite")];
    runCommand(hydroBinary, ["migrate", ...flags], { cwd: tempDir, stdio: "pipe" });
    runCommand(hydroBinary, ["create-admin", "--email", "admin@example.test", "--password-stdin", ...flags], { cwd: tempDir, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawnServer(hydroBinary, ["serve", "--dev", "--address", `127.0.0.1:${port}`, ...flags], {
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

  test("media, navigation, form builder and inbox work end to end", async ({ page, browser }) => {
    test.setTimeout(120_000);
    const problems = watchForErrors(page);
    await login(page);

    // Media: upload a PNG, then edit its tags and focal point.
    await page.goto(`${baseURL}/admin/media`);
    await page.locator("#media-file-input").setInputFiles(fixture);
    await page.locator('#media-upload-form button[type="submit"]').click();
    const card = page.locator('.media-card[data-filename="tiny.png"]');
    await expect(card).toBeVisible({ timeout: 15_000 });
    await card.locator(".media-meta-disclosure summary").click();
    await card.locator('input[name="tags"]').fill("e2e, hero");
    await card.locator('input[name="focal_x"]').fill("0.25");
    await card.locator(".media-meta-form button[type=submit]").click();
    await expect(page).toHaveURL(`${baseURL}/admin/media`);
    await page.goto(`${baseURL}/admin/media?tag=e2e`);
    await expect(page.locator(".media-card")).toHaveCount(1);
    await expect(page.locator('.media-card[data-filename="tiny.png"] input[name="focal_x"]')).toHaveValue("0.25");
    await expect(page.locator('.media-card[data-filename="tiny.png"] img.media-focal-preview')).toHaveAttribute("style", /object-position:\s*25(?:\.0+)?%/);
    // Client-side search comes from media-library.js.
    await page.goto(`${baseURL}/admin/media`);
    await page.locator("#media-search").fill("no-such-file");
    await expect(page.locator("#media-count-badge")).toHaveText("0 asset");

    // Navigation: add two links, move the second above the first, save.
    await page.goto(`${baseURL}/admin/site/navigation/editor?locale=tr`);
    for (const [label, href] of [["Birinci", "/birinci"], ["Ikinci", "/ikinci"]]) {
      await page.locator("#navigation-add").click();
      const row = page.locator("#navigation-tree [data-navigation-row]").last();
      await row.locator('input[name="label"]').fill(label);
      await row.locator('input[name="href"]').fill(href);
    }
    const rows = page.locator("#navigation-tree [data-navigation-row]");
    await expect(rows).toHaveCount(2);
    // Each added row got unique, labelled controls.
    await expect(rows.nth(1).getByLabel("Bağlantı adı")).toHaveValue("Ikinci");
    await rows.nth(1).locator("[data-move-up]").click();
    await expect(rows.nth(0).locator('input[name="label"]')).toHaveValue("Ikinci");
    await expect(rows.nth(0).locator('input[name="position"]')).toHaveValue("0");
    await page.getByRole("button", { name: "Menüyü kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/site\/navigation\/editor\?locale=tr&saved=1$/);
    expect(await inputValues(page, '#navigation-tree input[name="label"]')).toEqual(["Ikinci", "Birinci"]);

    // Form builder: replace the default fields with two new ones and save.
    await page.goto(`${baseURL}/admin/site/forms`);
    const builder = page.locator("[data-form-builder]");
    await builder.locator('input[name="title"]').fill("Bize yazın");
    while ((await builder.locator("[data-form-field]").count()) > 0) {
      await builder.locator("[data-form-remove-field]").first().click();
    }
    const fields = [
      { id: "email", label: "E-posta", type: "email", required: true },
      { id: "topic", label: "Konu", type: "text", required: false },
    ];
    for (const field of fields) {
      await builder.locator("[data-form-add-field]").click();
      const row = builder.locator("[data-form-field]").last();
      await row.locator('input[name="field_id"]').fill(field.id);
      await row.locator('input[name="field_label"]').fill(field.label);
      await row.locator('select[name="field_type"]').selectOption(field.type);
      await row.locator('input[name="field_required"]').setChecked(field.required);
    }
    await builder.getByRole("button", { name: "Formu kaydet" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/site/forms?id=contact`);
    expect(await inputValues(page, '[data-form-builder] input[name="field_id"]')).toEqual(["email", "topic"]);
    await expect(page.locator('[data-form-builder] input[name="title"]')).toHaveValue("Bize yazın");

    // Public site: an anonymous visitor submits the form.
    // Like the public form script, it sends the CSRF cookie it was given back
    // as the X-CSRF-Token header.
    const visitor = await browser.newContext();
    await visitor.request.get(`${baseURL}/login`);
    const csrf = (await visitor.cookies(baseURL)).find((cookie) => cookie.name === "hydro_csrf")?.value ?? "";
    expect(csrf).not.toBe("");
    const submitted = await visitor.request.post(`${baseURL}/forms/contact/submit`, {
      form: { email: "visitor@example.test", topic: "Fiyat teklifi", _csrf: csrf, _submission_token: "e2e-submission-0001" },
      headers: { Origin: baseURL, "X-CSRF-Token": csrf },
    });
    expect(submitted.status()).toBe(202);
    await visitor.close();

    // Inbox: the submission is listed and searchable.
    await page.goto(`${baseURL}/admin/site/inbox/contact`);
    await expect(page.locator("table")).toContainText("visitor@example.test");
    await page.locator("#inbox-search").fill("nobody");
    await page.getByRole("button", { name: "Ara", exact: true }).click();
    await expect(page.locator("table")).not.toContainText("visitor@example.test");

    // Translations search and appearance contrast are driven by static scripts.
    await page.goto(`${baseURL}/admin/translations`);
    await page.locator("#translation-search").fill("zzz-no-such-key");
    await expect(page.locator("[data-translation-row]:visible")).toHaveCount(0);
    await page.goto(`${baseURL}/admin/site/appearance`);
    await expect(page.locator("#appearance-contrast")).toHaveAttribute("data-state", /^(low|aa|aaa)$/);

    expect(problems, problems.join("\n")).toEqual([]);
  });
});

function watchForErrors(page: Page): string[] {
  const problems: string[] = [];
  page.on("pageerror", (error) => problems.push(`pageerror: ${error.message}`));
  page.on("console", (message) => {
    if (message.type() === "error" && /Content Security Policy|Refused to/i.test(message.text())) problems.push(`console: ${message.text()}`);
  });
  return problems;
}

async function inputValues(page: Page, selector: string): Promise<string[]> {
  return page.locator(selector).evaluateAll((inputs) => inputs.map((input) => (input as HTMLInputElement).value));
}

async function login(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem("hydro-onboarding-dismissed", "1");
    localStorage.setItem("hydro-onboarding-dismissed:admin", "1");
  });
  await page.goto(`${baseURL}/login`);
  await page.locator('input[name="email"]').fill("admin@example.test");
  await page.locator('input[name="password"]').fill(adminPassword);
  await page.locator('button[type="submit"]').first().click();
  await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
}

async function unusedPort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const listener = net.createServer();
    listener.once("error", reject);
    listener.listen(0, "127.0.0.1", () => {
      const address = listener.address();
      listener.close(() => (typeof address === "object" && address ? resolve(address.port) : reject(new Error("no port"))));
    });
  });
}
