import { test, expect } from "@playwright/test";
import type { ChildProcess } from "node:child_process";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { runCommand, spawnServer, stopServer } from "./server-process";

// End-to-end check of content types (docs/decisions/0002-content-types.md):
// a YAML file becomes admin screens and a public read API.
const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

const faqYAML = `id: faq
label: FAQ
title_field: question
public: true
fields:
  - {id: question, label: Question, kind: text, required: true, list: true}
  - {id: answer, label: Answer, kind: textarea}
  - {id: category, label: Category, kind: select, options: [general, billing], list: true}
`;

test.describe("content types", () => {
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-content-types-"));
    const typesDir = path.join(tempDir, "content-types");
    await mkdir(typesDir);
    await writeFile(path.join(typesDir, "faq.yaml"), faqYAML);
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
      env: { ...process.env, HYDRO_CONTENT_TYPES_DIR: typesDir, HYDRO_PUBLIC_ORIGIN: baseURL, HYDRO_MEDIA_DIR: path.join(tempDir, "media"), HYDRO_REDIS_URL: "" },
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
  }, 120_000);

  test.afterAll(async () => {
    await stopServer(server);
    if (tempDir) await rm(tempDir, { recursive: true, force: true });
  });

  test("a YAML content type gets admin CRUD and a public API", async ({ page }) => {
    test.setTimeout(120_000);
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

    await page.goto(`${baseURL}/admin/resources/faq/new`);
    await page.getByLabel("Question").fill("How long does shipping take?");
    await page.getByLabel("Answer").fill("Two to three working days.");
    await page.getByLabel("Category").selectOption("billing");
    await page.locator('form button[type="submit"]').last().click();
    await expect(page).toHaveURL(`${baseURL}/admin/resources/faq`);
    await expect(page.locator("table")).toContainText("How long does shipping take?");

    const a11y = await page.evaluate(async () => (window as any).axe.run(document.querySelector(".studio-main"), {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa"] },
    }));
    expect(a11y.violations.map((violation: any) => violation.id)).toEqual([]);

    const list = await page.request.get(`${baseURL}/api/v1/content/faq`);
    expect(list.status()).toBe(200);
    const body = await list.json();
    expect(body.items).toHaveLength(1);
    expect(body.items[0]).toMatchObject({ question: "How long does shipping take?", category: "billing", version: 1 });

    const unknown = await page.request.get(`${baseURL}/api/v1/content/does_not_exist`);
    expect(unknown.status()).toBe(404);
  });
});

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
