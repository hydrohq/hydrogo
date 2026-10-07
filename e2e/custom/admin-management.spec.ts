import { test, expect } from "@playwright/test";
import { spawn, execFileSync, type ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";

const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe("single Edition administrator management", () => {
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-admin-management-"));
    const database = path.join(tempDir, "edition.sqlite");
    execFileSync("go", ["run", "./cmd/hydro", "migrate", "--database-dialect", "sqlite", "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
    execFileSync("go", ["run", "./cmd/hydro", "create-admin", "--email", "admin@example.test", "--password-stdin", "--database-dialect", "sqlite", "--database-dsn", database], { cwd: repoRoot, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawn("go", ["run", "./cmd/hydro", "serve", "--dev", "--database-dialect", "sqlite", "--database-dsn", database, "--address", `127.0.0.1:${port}`], { cwd: repoRoot, stdio: "ignore" });
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
    if (server && server.exitCode === null) {
      server.kill("SIGTERM");
      await new Promise<void>((resolve) => {
        const timeout = setTimeout(resolve, 5_000);
        server?.once("exit", () => { clearTimeout(timeout); resolve(); });
      });
    }
    if (tempDir) await rm(tempDir, { recursive: true, force: true });
  });

  test("admin login opens the Edition management areas and rejects anonymous access", async ({ page }) => {
    const denied = await page.request.get(`${baseURL}/admin/users`, { maxRedirects: 0 });
    expect([401, 302, 303]).toContain(denied.status());
    if (denied.status() !== 401) expect(denied.headers().location).toContain("/login");

    await page.goto(`${baseURL}/login`);
    const loginResponse = await page.goto(`${baseURL}/login`);
    const loginHTML = await loginResponse?.text() ?? "";
    const csp = loginResponse?.headers()["content-security-policy"] ?? "";
    for (const [, kind, body] of loginHTML.matchAll(/<(script|style)(?:\s[^>]*)?>([\s\S]*?)<\/\1>/g)) {
      const hash = `sha256-${Buffer.from(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(body))).toString("base64")}`;
      expect(csp).toContain(`'${hash}'`);
      expect(csp).toContain(kind === "script" ? "script-src" : "style-src");
    }
    await page.getByLabel("Email address").fill("admin@example.test");
    await page.getByLabel("Password").fill(adminPassword);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
    await page.goto(`${baseURL}/admin`);
    await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);
    await expect(page.locator("h1")).toBeVisible();

    for (const route of [
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
    ]) {
      const response = await page.goto(`${baseURL}${route}`);
      expect(response?.status(), `${route} should be available to the Edition admin`).toBe(200);
      await expect(page.locator("h1").first()).toBeVisible();
    }
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
