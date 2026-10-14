import { test, expect, type Page } from "@playwright/test";
import type { ChildProcess } from "node:child_process";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { runCommand, spawnServer, stopServer } from "./server-process";

// End-to-end check of content types (docs/decisions/0002-content-types.md):
// a YAML file becomes admin screens and a public read API. The server runs
// with two site locales so localized fields get locale tabs.
const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

const faqYAML = `id: faq
label: FAQ
title_field: question
public: true
drafts: true
revisions: true
fields:
  - {id: question, label: Question, kind: text, required: true, list: true, searchable: true}
  - {id: answer, label: Answer, kind: textarea, searchable: true}
  - {id: category, label: Category, kind: select, options: [general, billing], list: true, filterable: true}
`;

const teamYAML = `id: team
label: Team
title_field: name
public: true
revisions: true
fields:
  - {id: name, label: Name, kind: text, required: true, list: true}
  - {id: role, label: Role, kind: text, list: true, localized: true}
  - {id: bio, label: Bio, kind: textarea, localized: true}
`;

test.describe("content types", () => {
  test.beforeAll(async () => {
    // Building the binary alone can take longer than the default hook timeout.
    test.setTimeout(180_000);
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-content-types-"));
    const typesDir = path.join(tempDir, "content-types");
    await mkdir(typesDir);
    await writeFile(path.join(typesDir, "faq.yaml"), faqYAML);
    await writeFile(path.join(typesDir, "team.yaml"), teamYAML);
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
      env: { ...process.env, HYDRO_CONTENT_TYPES_DIR: typesDir, HYDRO_PUBLIC_ORIGIN: baseURL, HYDRO_MEDIA_DIR: path.join(tempDir, "media"), HYDRO_REDIS_URL: "", HYDRO_I18N_ENABLED: "true", HYDRO_DEFAULT_LOCALE: "en", HYDRO_LOCALES: "en,de" },
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

  test("a YAML content type gets admin CRUD and a public API", async ({ page }) => {
    test.setTimeout(120_000);
    await signIn(page);

    const createFAQ = async (question: string, answer: string, category: string) => {
      await page.goto(`${baseURL}/admin/resources/faq/new`);
      await page.getByLabel("Question").fill(question);
      await page.getByLabel("Answer").fill(answer);
      await page.getByLabel("Category").selectOption(category);
      await page.locator('form.studio-form button[type="submit"]').last().click();
      await expect(page).toHaveURL(`${baseURL}/admin/resources/faq`);
    };
    await createFAQ("How long does shipping take?", "Two to three working days.", "billing");
    await createFAQ("Where is your office?", "In the city centre.", "general");
    const table = page.locator("table");
    await expect(table).toContainText("How long does shipping take?");
    await expect(table).toContainText("Where is your office?");

    // Search matches the question and the answer, case-insensitively.
    // Locators avoid UI strings: the admin language depends on the browser.
    const search = page.locator("#resource-search");
    const apply = page.locator('.admin-list-toolbar button[type="submit"]');
    await search.fill("WORKING DAYS");
    await apply.click();
    await expect(page).toHaveURL(/search=WORKING\+DAYS/);
    await expect(table).toContainText("How long does shipping take?");
    await expect(table).not.toContainText("Where is your office?");
    await search.fill("");
    await page.locator("#filter-category").selectOption("general");
    await apply.click();
    await expect(table).toContainText("Where is your office?");
    await expect(table).not.toContainText("How long does shipping take?");

    const a11y = await page.evaluate(async () => (window as any).axe.run(document.querySelector(".studio-main"), {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa"] },
    }));
    expect(a11y.violations.map((violation: any) => violation.id)).toEqual([]);

    // New records are drafts: the public API does not return them.
    const drafts = await page.request.get(`${baseURL}/api/v1/content/faq`);
    expect(drafts.status()).toBe(200);
    expect((await drafts.json()).items).toHaveLength(0);

    await page.goto(`${baseURL}/admin/resources/faq?search=shipping`);
    await page.getByRole("link", { name: "How long does shipping take?" }).click();
    const status = page.locator("input#status");
    const workflowButton = (target: string) => page.locator(`form[action$="/workflow"]:has(input[name="target"][value="${target}"]) button`);
    await expect(status).toHaveValue("draft");
    await workflowButton("published").click();
    await expect(status).toHaveValue("published");

    const list = await page.request.get(`${baseURL}/api/v1/content/faq?sort=-question`);
    expect(list.status()).toBe(200);
    const body = await list.json();
    expect(body.items).toHaveLength(1);
    expect(body.items[0]).toMatchObject({ question: "How long does shipping take?", category: "billing", status: "published", version: 2 });
    const filtered = await page.request.get(`${baseURL}/api/v1/content/faq?filter[category]=general`);
    expect((await filtered.json()).items).toHaveLength(0);

    await workflowButton("draft").click();
    await expect(status).toHaveValue("draft");
    // Every save is kept as a revision: create, publish, move to draft.
    await expect(page.locator('.revision-list form[action*="/revisions/"][method="get"]')).toHaveCount(3);
    const unpublished = await page.request.get(`${baseURL}/api/v1/content/faq`);
    expect((await unpublished.json()).items).toHaveLength(0);

    const unknown = await page.request.get(`${baseURL}/api/v1/content/does_not_exist`);
    expect(unknown.status()).toBe(404);
  });

  test("localized fields are edited per locale and served with ?locale=", async ({ page }) => {
    test.setTimeout(120_000);
    await signIn(page);
    await page.goto(`${baseURL}/admin/resources/team/new`);
    await page.getByLabel("Name").fill("Ada Example");
    await page.getByLabel("Role").fill("Engineer");
    await page.getByLabel("Bio").fill("Writes the compiler.");
    await page.locator('form.studio-form button[type="submit"]').last().click();
    await expect(page).toHaveURL(`${baseURL}/admin/resources/team?locale=en`);

    await page.getByRole("link", { name: "Ada Example" }).click();
    // The locale tabs are GET forms; DE switches the form to German.
    await page.locator('nav.locale-tabs form:has(input[name="locale"][value="de"]) button').click();
    await expect(page).toHaveURL(/locale=de/);
    const role = page.getByLabel("Role");
    await expect(role).toHaveValue("");
    await role.fill("Ingenieurin");
    await page.locator('form[action*="/locale?locale=de"] button[type="submit"]').last().click();
    await expect(page).toHaveURL(/\/admin\/resources\/team\/[^/?]+\?locale=de$/);
    await expect(page.getByLabel("Role")).toHaveValue("Ingenieurin");

    const german = await page.request.get(`${baseURL}/api/v1/content/team?locale=de`);
    expect(german.status()).toBe(200);
    const germanBody = await german.json();
    expect(germanBody.locale).toBe("de");
    // Bio has no German value and falls back to English.
    expect(germanBody.items[0]).toMatchObject({ name: "Ada Example", role: "Ingenieurin", bio: "Writes the compiler.", locale: "de" });
    const english = await (await page.request.get(`${baseURL}/api/v1/content/team/${germanBody.items[0].id}`)).json();
    expect(english).toMatchObject({ role: "Engineer", locale: "en" });
    expect((await page.request.get(`${baseURL}/api/v1/content/team?locale=fr`)).status()).toBe(400);
  });
});

async function signIn(page: Page): Promise<void> {
  await page.addInitScript(() => {
    localStorage.setItem("hydro-onboarding-dismissed", "1");
    localStorage.setItem("hydro-onboarding-dismissed:admin", "1");
  });
  await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
  await page.goto(`${baseURL}/login`);
  await page.locator('input[name="email"]').fill("admin@example.test");
  await page.locator('input[name="password"]').fill(adminPassword);
  await page.locator('button[type="submit"]').first().click();
  await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
  await expect(page.locator('.studio-sidebar a[href="/admin/resources"]')).toBeVisible();
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
