import { test, expect } from "@playwright/test";
import { spawn, type ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { runCommand } from "./server-process";

const repoRoot = path.resolve(process.cwd(), "..");
const adminPassword = "Lime-Train7!Harbor";
const resetPassword = "Sage-Bridge8!Canyon";
const backupSigningSeed = "ab".repeat(32);
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe("administrator-issued password reset", () => {
  test.beforeAll(async () => {
    // Building and migrating with the Go toolchain can outlast the default hook timeout.
    test.setTimeout(180_000);
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-admin-password-reset-"));
    const database = path.join(tempDir, "edition.sqlite");
    const binary = path.join(tempDir, process.platform === "win32" ? "hydro.exe" : "hydro");
    runCommand("go", ["run", "./cmd/hydro", "migrate", "--database-dialect", "sqlite", "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
    runCommand("go", ["run", "./cmd/hydro", "create-admin", "--email", "admin@example.test", "--password-stdin", "--database-dialect", "sqlite", "--database-dsn", database], { cwd: repoRoot, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    runCommand("go", ["run", "./tests/e2e/support/seedroles", "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
    runCommand("go", ["build", "-o", binary, "./cmd/hydro"], { cwd: repoRoot, stdio: "pipe" });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawn(binary, ["serve", "--dev", "--database-dialect", "sqlite", "--database-dsn", database, "--address", `127.0.0.1:${port}`], { cwd: repoRoot, stdio: "ignore", env: { ...process.env, HYDRO_PUBLIC_ORIGIN: baseURL, HYDRO_MEDIA_DIR: path.join(tempDir, "media"), HYDRO_BACKUP_SIGNING_SEED: backupSigningSeed } });
    const deadline = Date.now() + 45_000;
    while (Date.now() < deadline) {
      if (server.exitCode !== null) throw new Error(`HydroGo server exited with ${server.exitCode}`);
      try {
        if ((await fetch(`${baseURL}/healthz`)).ok) return;
      } catch {
        // Wait while the server starts against the isolated SQLite fixture.
      }
      await new Promise((resolve) => setTimeout(resolve, 200));
    }
    throw new Error("HydroGo server did not become healthy within 45 seconds");
  });

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

  test("admin reset link completes password recovery and replaces the old credential", async ({ browser }) => {
    test.setTimeout(60_000);
    const context = await browser.newContext();
    const adminPage = await context.newPage();
    await adminPage.goto(`${baseURL}/login`);
    await adminPage.getByLabel("Email address").fill("admin@example.test");
    await adminPage.getByLabel("Password").fill(adminPassword);
    await adminPage.getByRole("button", { name: "Sign in" }).click();
    await expect(adminPage).toHaveURL(/\/admin(?:\?.*)?$/);
    await adminPage.goto(`${baseURL}/admin/users`);

    const editorRow = adminPage.locator(".studio-table tbody tr").filter({ has: adminPage.getByText("editor@example.test", { exact: true }) });
    const resetButton = editorRow.locator("[data-password-reset-link]");
    await expect(resetButton).toBeVisible();
    const resetResponsePromise = adminPage.waitForResponse((response) => response.request().method() === "POST" && response.url().includes("/password-reset-link"));
    await resetButton.click();
    const resetResponse = await resetResponsePromise;
    expect(resetResponse.status()).toBe(200);
    const { url: resetURL } = await resetResponse.json();
    const parsedResetURL = new URL(resetURL);
    expect(parsedResetURL.origin).toBe(baseURL);
    expect(parsedResetURL.pathname).toBe("/password-reset");
    expect(parsedResetURL.searchParams.get("token")).toMatch(/^[a-f0-9]{64}$/);

    const resetContext = await browser.newContext();
    const resetPage = await resetContext.newPage();
    await resetPage.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    const resetPageResponse = await resetPage.goto(resetURL);
    expect(resetPageResponse?.status()).toBe(200);
    expect(resetPageResponse?.headers()["cache-control"]).toContain("no-store");
    expect(resetPageResponse?.headers()["referrer-policy"]).toBe("no-referrer");
    await expect(resetPage.getByRole("heading", { name: "Choose a new password" })).toBeVisible();
    await expect(resetPage).toHaveURL(`${baseURL}/password-reset`);
    const a11y = await resetPage.evaluate(async () => (window as any).axe.run(document, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
    }));
    expect(a11y.violations.map((violation: any) => violation.id)).toEqual([]);

    await resetPage.getByLabel("New password", { exact: true }).fill("passwordpassword");
    await resetPage.getByLabel("Confirm new password").fill("passwordpassword");
    await resetPage.getByRole("button", { name: "Reset password" }).click();
    await expect(resetPage.getByRole("alert")).toContainText("less predictable password");
    await expect(resetPage).toHaveURL(`${baseURL}/password-reset`);

    await resetPage.getByLabel("New password", { exact: true }).fill(resetPassword);
    await resetPage.getByLabel("Confirm new password").fill(`${resetPassword}Different`);
    await resetPage.getByRole("button", { name: "Reset password" }).click();
    await expect(resetPage.getByRole("alert")).toContainText("passwords do not match");
    await expect(resetPage).toHaveURL(`${baseURL}/password-reset`);

    await resetPage.getByLabel("New password", { exact: true }).fill(resetPassword);
    await resetPage.getByLabel("Confirm new password").fill(resetPassword);
    await resetPage.getByRole("button", { name: "Reset password" }).click();
    await expect(resetPage).toHaveURL(`${baseURL}/login?password_reset=1`);
    await expect(resetPage.getByRole("status")).toContainText("password has been updated");

    const loginPage = await resetContext.newPage();
    await loginPage.goto(`${baseURL}/login`);
    await loginPage.getByLabel("Email address").fill("editor@example.test");
    await loginPage.getByLabel("Password").fill(adminPassword);
    await loginPage.getByRole("button", { name: "Sign in" }).click();
    await expect(loginPage.getByRole("alert")).toContainText("incorrect");
    await loginPage.getByLabel("Password").fill(resetPassword);
    await loginPage.getByRole("button", { name: "Sign in" }).click();
    await expect(loginPage).toHaveURL(/\/admin(?:\?.*)?$/);
    await expect(loginPage.locator('body[data-studio-role="editor"]')).toBeVisible();
    await resetContext.close();
    await context.close();
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
