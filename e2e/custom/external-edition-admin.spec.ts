import { test, expect, type ChildProcess } from "@playwright/test";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { spawnServer, stopServer } from "./server-process";

const repoRoot = path.resolve(process.cwd(), "..");
const editionRoot = path.join(repoRoot, "tests", "external-edition");
let server: ChildProcess | undefined;
let serverArgs: string[] = [];
let tempDir = "";
let baseURL = "";

test.describe("independent Edition generated Admin", () => {
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-external-edition-"));
    const dialect = process.env.HYDRO_E2E_DATABASE_DIALECT ?? "sqlite";
    const dsn = process.env.HYDRO_E2E_DATABASE_DSN ?? path.join(tempDir, "edition.sqlite");
    if (dialect === "postgres" && !process.env.HYDRO_E2E_DATABASE_DSN) {
      throw new Error("HYDRO_E2E_DATABASE_DSN is required for the PostgreSQL tenant-isolation run");
    }
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    serverArgs = ["run", "./cmd/testserver", "--address", `127.0.0.1:${port}`, "--dialect", dialect, "--dsn", dsn];
    await startServer();
  }, 60_000);

  test.afterAll(async () => {
    await stopServer(server);
    if (tempDir) await rm(tempDir, { recursive: true, force: true });
  });

  test("renders and runs generated CRUD with Edition role and tenant boundaries", async ({ browser }) => {
    const admin = await browser.newContext({ extraHTTPHeaders: { "X-Test-Role": "admin", "X-Test-Tenant": "tenant-a" } });
    const page = await admin.newPage();
    await page.goto(`${baseURL}/admin/resources`);
    await expect(page.getByRole("heading", { name: "Kaynaklar" })).toBeVisible();
    const onboarding = page.locator("#admin-onboarding-dialog");
    await expect(onboarding).toBeVisible();
    await onboarding.getByRole("button", { name: "Bir daha gösterme" }).click();
    await expect(onboarding).toBeHidden();
    await page.getByRole("navigation").getByRole("link", { name: "Articles" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    await expect(page.getByRole("heading", { name: "Articles" })).toBeVisible();

    await page.getByRole("link", { name: "Yeni kayıt" }).click();
    const form = page.locator('form[action="/admin/resources/article/new"]');
    await expect(form.getByLabel("Title")).toBeVisible();
    await form.getByLabel("Title").fill("External Edition tenant A record");
    await form.locator('button[type="submit"]').click();
    await expect(page.getByRole("link", { name: "External Edition tenant A record" })).toBeVisible();
    const recordPath = await page.getByRole("link", { name: "External Edition tenant A record" }).getAttribute("href");
    expect(recordPath).toMatch(/^\/admin\/resources\/article\//);

    await stopServer(server);
    await startServer(true);
    await page.reload();
    await expect(page.getByRole("link", { name: "External Edition tenant A record" })).toBeVisible();

    const tenantB = await browser.newContext({ extraHTTPHeaders: { "X-Test-Role": "admin", "X-Test-Tenant": "tenant-b" } });
    const tenantBPage = await tenantB.newPage();
    await tenantBPage.goto(`${baseURL}/admin/resources/article`);
    await expect(tenantBPage.getByRole("heading", { name: "Articles" })).toBeVisible();
    await expect(tenantBPage.getByText("External Edition tenant A record")).toHaveCount(0);
    const crossTenantRead = await tenantB.request.get(`${baseURL}${recordPath!.replace("/admin/resources", "/api/v1")}`);
    expect(crossTenantRead.status()).toBe(404);

    const viewer = await browser.newContext({ extraHTTPHeaders: { "X-Test-Role": "viewer", "X-Test-Tenant": "tenant-a" } });
    const viewerPage = await viewer.newPage();
    const viewerDenied = await viewerPage.goto(`${baseURL}/admin/resources/article`);
    expect(viewerDenied?.status()).toBe(403);
    const csrfDenied = await admin.request.post(`${baseURL}/admin/resources/article/new`, { form: { id: "csrf-denied", title: "Must not persist" } });
    expect(csrfDenied.status()).toBe(403);
    expect((await admin.request.get(`${baseURL}/api/v1/article/csrf-denied`)).status()).toBe(404);

    await page.getByRole("link", { name: "External Edition tenant A record" }).click();
    await expect(page.locator('[name="title"]')).toHaveValue("External Edition tenant A record");
    await page.locator('[name="title"]').fill("External Edition tenant A updated");
    await page.locator(`form[action="${recordPath}"] button[type="submit"]`).click();
    await expect(page.getByRole("link", { name: "External Edition tenant A updated" })).toBeVisible();
    await page.getByRole("link", { name: "External Edition tenant A updated" }).click();
    await page.locator(`form[action="${recordPath}/delete"] button[type="submit"]`).click();
    await expect(page.getByText("External Edition tenant A updated")).toHaveCount(0);
    expect((await admin.request.get(`${baseURL}${recordPath!.replace("/admin/resources", "/api/v1")}`)).status()).toBe(404);
    await Promise.all([admin.close(), tenantB.close(), viewer.close()]);
  });
});

async function startServer(skipMigrate = false): Promise<void> {
  server = spawnServer("go", skipMigrate ? [...serverArgs, "--skip-migrate"] : serverArgs, { cwd: editionRoot, stdio: "ignore" });
  const deadline = Date.now() + 45_000;
  while (Date.now() < deadline) {
    if (server.exitCode !== null) throw new Error(`external Edition server exited with ${server.exitCode}`);
    try {
      const response = await fetch(`${baseURL}/healthz`);
      if (response.ok) return;
    } catch {
      // Wait while the independent Go module builds and starts.
    }
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error("external Edition server did not become healthy within 45 seconds");
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
