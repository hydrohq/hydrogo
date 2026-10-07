import { test, expect } from "@playwright/test";
import { execFileSync, type ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { spawnServer, stopServer } from "./server-process";

const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe("dynamic Admin collections", () => {
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-dynamic-collections-"));
    const editionRoot = path.join(tempDir, "edition");
    const hydroBinary = path.join(tempDir, process.platform === "win32" ? "hydro.exe" : "hydro");
    execFileSync("go", ["build", "-o", hydroBinary, "./cmd/hydro"], { cwd: repoRoot, stdio: "pipe" });
    execFileSync(hydroBinary, ["project", "new", "edition", "--root", tempDir, "--preset", "compact-site", "--no-demo", "--with-collections"], { cwd: repoRoot, stdio: "pipe" });
    execFileSync(hydroBinary, ["resource", "create", "--name", "sync_articles", "--field", "title:text", "--out", "."], { cwd: editionRoot, stdio: "pipe" });
    execFileSync(hydroBinary, ["migrate", "--config", "config.yaml"], { cwd: editionRoot, stdio: "pipe" });
    execFileSync(hydroBinary, ["collections", "sync", "--config", "config.yaml", "--generated", "generated", "--confirm-admin-crud"], { cwd: editionRoot, stdio: "pipe" });
    execFileSync(hydroBinary, ["create-admin", "--config", "config.yaml", "--email", "admin@example.test", "--password-stdin"], { cwd: editionRoot, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
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

  test("defines a collection and gets a schema-driven Admin CRUD surface", async ({ page }) => {
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
    await page.getByRole("link", { name: `Edit ${importedID}` }).click();
    const importedEdit = page.locator(`form[action="/admin/collections/sync_articles/records/${importedID}"]`);
    await importedEdit.getByLabel("Title").fill("Updated Blueprint record");
    await importedEdit.getByRole("button", { name: "Save record" }).click();
    await expect(page.locator("table")).toContainText("Updated Blueprint record");
    await page.getByRole("link", { name: `Edit ${importedID}` }).click();
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

    await page.getByRole("link", { name: `Edit ${recordID}` }).click();
    await expectNoA11yViolations(page);
    const editRecord = page.locator(`form[action="/admin/collections/e2e_catalog/records/${recordID}"]`);
    await editRecord.getByLabel("Product name").fill("Updated sample product");
    await editRecord.getByRole("button", { name: "Save record" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections/e2e_catalog`);
    await expect(page.locator("table")).toContainText("Updated sample product");

    await page.getByRole("link", { name: `Edit ${recordID}` }).click();
    await page.locator(`form[action="/admin/collections/e2e_catalog/records/${recordID}/delete"]`).getByRole("button", { name: "Delete record" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/collections/e2e_catalog`);
    await expect(page.getByText("No records are stored in this collection.")).toBeVisible();

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
