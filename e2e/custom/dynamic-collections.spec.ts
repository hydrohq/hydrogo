import { test, expect } from "@playwright/test";
import type { ChildProcess } from "node:child_process";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { runCommand, spawnServer, stopServer } from "./server-process";

const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe("dynamic Admin collections", () => {
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-dynamic-collections-"));
    const editionRoot = path.join(tempDir, "edition");
    const databaseDialect = process.env.HYDRO_E2E_DATABASE_DIALECT ?? "sqlite";
    const databaseDSN = process.env.HYDRO_E2E_DATABASE_DSN ?? path.join(editionRoot, "data", "edition.sqlite");
    if (databaseDialect === "postgres" && !process.env.HYDRO_E2E_DATABASE_DSN) {
      throw new Error("HYDRO_E2E_DATABASE_DSN is required for a PostgreSQL Dynamic Collections run");
    }
    const hydroBinary = path.join(tempDir, process.platform === "win32" ? "hydro.exe" : "hydro");
    runCommand("go", ["build", "-o", hydroBinary, "./cmd/hydro"], { cwd: repoRoot, stdio: "pipe" });
    runCommand(hydroBinary, ["project", "new", "edition", "--root", tempDir, "--preset", "compact-site", "--database", databaseDialect, "--no-demo", "--with-collections"], { cwd: repoRoot, stdio: "pipe" });
    if (databaseDialect === "postgres") {
      const configPath = path.join(editionRoot, "config.yaml");
      const config = await readFile(configPath, "utf8");
      const updatedConfig = config.replace(/^  dsn: .*$/m, `  dsn: ${JSON.stringify(databaseDSN)}`);
      if (updatedConfig === config) throw new Error("generated Edition config is missing its database DSN");
      await writeFile(configPath, updatedConfig);
    }
    runCommand(hydroBinary, ["resource", "create", "--name", "sync_articles", "--field", "title:text", "--out", "."], { cwd: editionRoot, stdio: "pipe" });
    runCommand(hydroBinary, ["migrate", "--config", "config.yaml"], { cwd: editionRoot, stdio: "pipe" });
    runCommand(hydroBinary, ["collections", "sync", "--config", "config.yaml", "--generated", "generated", "--confirm-admin-crud"], { cwd: editionRoot, stdio: "pipe" });
    runCommand(hydroBinary, ["create-admin", "--config", "config.yaml", "--email", "admin@example.test", "--password-stdin"], { cwd: editionRoot, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    runCommand("go", ["run", "./tests/e2e/support/seedroles", "--database-dialect", databaseDialect, "--database-dsn", databaseDSN], { cwd: repoRoot, stdio: "pipe" });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawnServer(hydroBinary, ["serve", "--dev", "--config", "config.yaml", "--address", `127.0.0.1:${port}`], {
      cwd: editionRoot,
      stdio: "ignore",
      env: { ...process.env, HYDRO_PUBLIC_ORIGIN: baseURL, HYDRO_MEDIA_DIR: path.join(tempDir, "media") },
    });
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

  test("defines a collection and gets a schema-driven Admin CRUD surface", async ({ page, browser }) => {
    test.setTimeout(240_000);
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await page.goto(`${baseURL}/login`);
    await page.getByLabel("Email address").fill("admin@example.test");
    await page.getByLabel("Password").fill(adminPassword);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
    const collectionsLink = page.locator('.studio-sidebar a[href="/admin/collections"]');
    await expect(collectionsLink).toBeVisible();
    await expect(collectionsLink).toContainText("Koleksiyonlar");

    const index = await page.goto(`${baseURL}/admin/collections`);
    expect(index?.status()).toBe(200);
    await expect(page.locator('.studio-sidebar a[href="/admin/collections"]')).toHaveAttribute("aria-current", "page");
    await expect(page.getByRole("heading", { name: "Collections and schemas" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Sync articles" })).toHaveAttribute("href", "/admin/collections/sync_articles");
    const cardBorder = await page.locator(".studio-card").first().evaluate((card) => getComputedStyle(card).borderTopStyle);
    expect(cardBorder).toBe("solid");
    const a11y = await page.evaluate(async () => (window as any).axe.run(document, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
    }));
    expect(a11y.violations.map((violation: any) => ({ id: violation.id, targets: violation.nodes.map((node: any) => node.target) }))).toEqual([]);

    await page.getByRole("link", { name: "Sync articles" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections/sync_articles`);
    await expect(page.getByRole("heading", { name: "Sync articles" }).first()).toBeVisible();
    const importedRecord = page.locator('form[action="/admin/collections/sync_articles/records"]');
    await importedRecord.getByLabel("Title").fill("Blueprint generated record");
    await importedRecord.getByRole("button", { name: "Save record" }).click();
    await expect(page.locator("table")).toContainText("Blueprint generated record");
    const importedResponse = await page.request.get(`${baseURL}/api/v1/collections/sync_articles/records`);
    expect(importedResponse.status()).toBe(200);
    const importedRecords = await importedResponse.json();
    expect(importedRecords.records).toHaveLength(1);
    const importedID = importedRecords.records[0].id as string;
    await page.locator("table tbody tr").filter({ hasText: "Blueprint generated record" }).getByRole("link", { name: /^Edit record:/ }).click();
    const importedEdit = page.locator(`form[action="/admin/collections/sync_articles/records/${importedID}"]`);
    await importedEdit.getByLabel("Title").fill("Updated Blueprint record");
    await importedEdit.getByRole("button", { name: "Save record" }).click();
    await expect(page.locator("table")).toContainText("Updated Blueprint record");
    await page.locator("table tbody tr").filter({ hasText: "Updated Blueprint record" }).getByRole("link", { name: /^Edit record:/ }).click();
    await page.locator(`form[action="/admin/collections/sync_articles/records/${importedID}/delete"]`).getByRole("button", { name: "Delete record" }).click();
    await expect(page.getByText("No records are stored in this collection.")).toBeVisible();
    await page.goto(`${baseURL}/admin/collections`);
    const deleteImported = page.locator('form[action="/admin/collections/sync_articles/delete"]');
    await deleteImported.getByLabel("Delete this schema and its records").check();
    await deleteImported.getByRole("button", { name: "Delete schema" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections`);

    const createSchema = page.locator('form[action="/admin/collections"]');
    await createSchema.getByLabel("Collection ID").fill("e2e_catalog");
    await createSchema.getByLabel("Name", { exact: true }).fill("E2E Catalog");
    await createSchema.getByLabel("Description").fill("Generated panel acceptance fixture.");
    const fields = createSchema.locator("[data-collection-field-row]");
    const sku = fields.nth(0);
    await sku.locator('[name="field_id_0"]').fill("sku");
    await sku.locator('[name="field_label_0"]').fill("Stock code");
    await sku.locator('[name="field_kind_0"]').selectOption("text");
    await sku.getByLabel("Required").check();
    await sku.getByLabel("Unique").check();
    await createSchema.getByRole("button", { name: "Add field" }).click();
    await expect(fields).toHaveCount(9);
    const title = fields.nth(8);
    await title.locator('[name="field_id_8"]').fill("title");
    await title.locator('[name="field_label_8"]').fill("Product name");
    await title.locator('[name="field_kind_8"]').selectOption("text");
    await title.getByLabel("Required").check();

    const noCSRF = await page.request.post(`${baseURL}/admin/collections`, { form: { id: "blocked", label: "Blocked" } });
    expect(noCSRF.status()).toBe(403);
    await createSchema.getByRole("button", { name: "Create collection" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections/e2e_catalog`);
    await expect(page.getByRole("heading", { name: "E2E Catalog" }).first()).toBeVisible();
    await expect(page.getByLabel("Stock code")).toHaveAttribute("required", "");
    await expect(page.getByLabel("Product name")).toHaveAttribute("required", "");
    await expectNoA11yViolations(page);
    await page.screenshot({ path: test.info().outputPath("dynamic-collection-admin-desktop.png") });

    const createRecord = page.locator('form[action="/admin/collections/e2e_catalog/records"]');
    const textFieldWidths = await createRecord.locator("label input[type=text]").evaluateAll((inputs) =>
      inputs.map((input) => (input as HTMLInputElement).getBoundingClientRect().width),
    );
    expect(Math.max(...textFieldWidths) - Math.min(...textFieldWidths)).toBeLessThanOrEqual(1);
    await createRecord.getByLabel("Stock code").fill("SKU-E2E-001");
    await createRecord.getByLabel("Product name").fill("HydroGo sample product");
    await createRecord.getByRole("button", { name: "Save record" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections/e2e_catalog`);
    await expect(page.locator("table")).toContainText("SKU-E2E-001");
    await expect(page.locator("table")).toContainText("HydroGo sample product");
    const recordsResponse = await page.request.get(`${baseURL}/api/v1/collections/e2e_catalog/records`);
    expect(recordsResponse.status()).toBe(200);
    const records = await recordsResponse.json();
    expect(records.records).toHaveLength(1);
    const recordID = records.records[0].id as string;

    const themeRoutes = [
      "/admin/collections",
      "/admin/collections/e2e_catalog",
      `/admin/collections/e2e_catalog/records/${recordID}`,
    ];
    for (const theme of ["hydrogo"]) {
      for (const mode of ["light", "dark"]) {
        await page.evaluate((colorMode) => localStorage.setItem("hydro-theme", colorMode), mode);
        for (const route of themeRoutes) {
          const response = await page.goto(`${baseURL}${route}`);
          expect(response?.status(), `${route} should render in ${theme}-${mode}`).toBe(200);
          await expect(page.locator("html")).toHaveAttribute("data-ui-theme", theme);
          await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
          await expectNoA11yViolations(page);
        }
      }
    }
    await page.goto(`${baseURL}/admin/collections/e2e_catalog`);

    await page.locator("table tbody tr").filter({ hasText: "SKU-E2E-001" }).getByRole("link", { name: /^Edit record:/ }).click();
    await expectNoA11yViolations(page);
    const editRecord = page.locator(`form[action="/admin/collections/e2e_catalog/records/${recordID}"]`);
    await editRecord.getByLabel("Product name").fill("Updated sample product");
    await editRecord.getByRole("button", { name: "Save record" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections/e2e_catalog`);
    await expect(page.locator("table")).toContainText("Updated sample product");

    await page.locator("table tbody tr").filter({ hasText: "SKU-E2E-001" }).getByRole("link", { name: /^Edit record:/ }).click();
    await page.locator(`form[action="/admin/collections/e2e_catalog/records/${recordID}/delete"]`).getByRole("button", { name: "Delete record" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections/e2e_catalog`);
    await expect(page.getByText("No records are stored in this collection.")).toBeVisible();

    const csrfResponse = await page.request.get(`${baseURL}/csrf-token`);
    const { token } = await csrfResponse.json();
    for (let index = 0; index < 51; index++) {
      const response = await page.request.post(`${baseURL}/admin/collections/e2e_catalog/records`, {
        form: { _csrf: token, sku: `PAGE-${String(index).padStart(3, "0")}`, title: `Page record ${index}` },
        maxRedirects: 0,
      });
      expect(response.status(), `create pagination fixture record ${index}`).toBe(303);
    }
    await page.goto(`${baseURL}/admin/collections/e2e_catalog`);
    const recordRows = page.locator("section.studio-card table tbody tr");
    await expect(recordRows).toHaveCount(50);
    await expect(page.locator("table")).toContainText("Stock code: PAGE-");
    await expect(page.locator("table")).toContainText("Product name: Page record");
    await expect(page.locator("table")).not.toContainText('{"sku":');
    await page.screenshot({ path: test.info().outputPath("collection-records-page-1-desktop.png") });
    const pagination = page.getByRole("navigation", { name: "Record pages" });
    await expect(pagination).toContainText("Page 1");
    await expect(pagination.getByText("Previous")).toHaveAttribute("aria-disabled", "true");
    await expectNoA11yViolations(page);
    await pagination.getByRole("link", { name: "Next" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections/e2e_catalog?offset=50`);
    await expect(recordRows).toHaveCount(1);
    await expect(pagination).toContainText("Page 2");
    await expect(pagination.getByText("Next")).toHaveAttribute("aria-disabled", "true");
    await expectNoA11yViolations(page);
    for (const offset of ["-1", "1", "10001", "invalid"]) {
      const invalidPage = await page.request.get(`${baseURL}/admin/collections/e2e_catalog?offset=${offset}`);
      expect(invalidPage.status(), `reject invalid page offset ${offset}`).toBe(400);
    }
    await pagination.getByRole("link", { name: "Previous" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections/e2e_catalog`);
    await expect(recordRows).toHaveCount(50);

    await page.setViewportSize({ width: 390, height: 844 });
    const mobileNavOpen = await page.locator("body").evaluate((body) => body.classList.contains("studio-mobile-nav-open"));
    expect(mobileNavOpen).toBe(false);
    const sidebar = page.locator(".studio-sidebar");
    expect(await sidebar.evaluate((element) => getComputedStyle(element).boxShadow)).toBe("none");
    await page.getByRole("button", { name: "Menüyü aç" }).click();
    await expect(page.locator("body")).toHaveClass(/studio-mobile-nav-open/);
    expect(await sidebar.evaluate((element) => getComputedStyle(element).boxShadow)).not.toBe("none");
    await page.keyboard.press("Escape");
    await expect(page.locator("body")).not.toHaveClass(/studio-mobile-nav-open/);
    expect(await sidebar.evaluate((element) => getComputedStyle(element).boxShadow)).toBe("none");
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
    expect(overflow).toBeLessThanOrEqual(1);
    await page.screenshot({ path: test.info().outputPath("dynamic-collection-admin-mobile.png") });

    for (const role of ["editor", "viewer"]) {
      const restricted = await browser.newContext();
      const restrictedPage = await restricted.newPage();
      await restrictedPage.goto(`${baseURL}/login`);
      await restrictedPage.getByLabel("Email address").fill(`${role}@example.test`);
      await restrictedPage.getByLabel("Password").fill(adminPassword);
      await restrictedPage.getByRole("button", { name: "Sign in" }).click();
      await expect(restrictedPage).toHaveURL(/\/admin(?:\?.*)?$/);
      expect((await restrictedPage.goto(`${baseURL}/admin/pages`))?.status(), `${role} can open the permitted content shell`).toBe(200);
      await expect(restrictedPage.locator('.studio-sidebar a[href="/admin/collections"]'), `${role} must not see the admin-only Collections navigation`).toHaveCount(0);
      await restrictedPage.locator(".studio-search").click();
      await expect(restrictedPage.locator(".studio-search-dialog .search-result-item").filter({ hasText: "Koleksiyonlar" }), `${role} must not find the admin-only Collections command`).toHaveCount(0);
      expect((await restrictedPage.request.get(`${baseURL}/admin/collections`)).status()).toBe(403);
      expect((await restrictedPage.request.get(`${baseURL}/admin/collections/e2e_catalog`)).status()).toBe(403);
      expect((await restrictedPage.request.get(`${baseURL}/api/v1/collections`)).status()).toBe(403);
      expect((await restrictedPage.request.get(`${baseURL}/api/v1/collections/e2e_catalog/records`)).status()).toBe(403);
      expect((await restrictedPage.request.get(`${baseURL}/admin/collections/schema-builder.js`)).status()).toBe(403);
      const csrfResponse = await restrictedPage.request.get(`${baseURL}/csrf-token`);
      expect(csrfResponse.status()).toBe(200);
      const { token } = await csrfResponse.json();
      const deniedSchemaCreate = await restrictedPage.request.post(`${baseURL}/admin/collections`, {
        form: { _csrf: token, id: `denied_${role}`, label: "Must not be created" },
      });
      expect(deniedSchemaCreate.status(), `${role} with a valid CSRF token cannot create Collections schemas`).toBe(403);
      const deniedRecordCreate = await restrictedPage.request.post(`${baseURL}/api/v1/collections/e2e_catalog/records`, {
        headers: { "X-CSRF-Token": token },
        data: { sku: `DENIED-${role}`, title: `Denied ${role} record` },
      });
      expect(deniedRecordCreate.status(), `${role} with a valid CSRF token cannot create dynamic collection records`).toBe(403);
      const deniedSchemaDelete = await restrictedPage.request.post(`${baseURL}/admin/collections/e2e_catalog/delete`, {
        form: { _csrf: token, confirm_schema: "e2e_catalog" },
      });
      expect(deniedSchemaDelete.status(), `${role} with a valid CSRF token cannot delete dynamic collection schemas`).toBe(403);
      const deniedAPISchemaDelete = await restrictedPage.request.delete(`${baseURL}/api/v1/collections/e2e_catalog`, {
        headers: { "X-CSRF-Token": token },
      });
      expect(deniedAPISchemaDelete.status(), `${role} with a valid CSRF token cannot delete dynamic collection schemas through the API`).toBe(403);
      await restricted.close();
    }

    const retainedSchema = await page.request.get(`${baseURL}/api/v1/collections/e2e_catalog`);
    expect(retainedSchema.status(), "denied Editor and Viewer schema mutations must preserve the live collection").toBe(200);
    const retainedRecords = await page.request.get(`${baseURL}/api/v1/collections/e2e_catalog/records?limit=100`);
    expect(retainedRecords.status()).toBe(200);
    const retainedRecordData = await retainedRecords.json();
    expect(retainedRecordData.records).toHaveLength(51);
    expect(JSON.stringify(retainedRecordData.records)).not.toContain("DENIED-editor");
    expect(JSON.stringify(retainedRecordData.records)).not.toContain("DENIED-viewer");

    await page.goto(`${baseURL}/admin/collections`);
    const deleteSchema = page.locator('form[action="/admin/collections/e2e_catalog/delete"]');
    await deleteSchema.getByLabel("Delete this schema and its records").check();
    await deleteSchema.getByRole("button", { name: "Delete schema" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections`);
    await expect(page.getByText("No collections have been created.")).toBeVisible();
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

async function expectNoA11yViolations(page: import("@playwright/test").Page): Promise<void> {
  const result = await page.evaluate(async () => (window as any).axe.run(document, {
    runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
  }));
  expect(result.violations.map((violation: any) => ({
    id: violation.id,
    targets: violation.nodes.map((node: any) => node.target),
  }))).toEqual([]);
}
