import { test, expect } from "@playwright/test";
import { spawn, execFileSync, type ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";

const repoRoot = path.resolve(process.cwd(), "..");
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe("generated Edition Admin CRUD", () => {
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-admin-crud-"));
    const dialect = process.env.HYDRO_E2E_DATABASE_DIALECT ?? "sqlite";
    const database = process.env.HYDRO_E2E_DATABASE_DSN ?? path.join(tempDir, "edition.sqlite");
    execFileSync("go", ["run", "./examples/project-edition/cmd/server", "migrate", "--dialect", dialect, "--dsn", database], { cwd: repoRoot, stdio: "pipe" });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawn("go", ["run", "./examples/project-edition/cmd/server", "serve", "--dialect", dialect, "--dsn", database, "--address", `127.0.0.1:${port}`], { cwd: repoRoot, stdio: "ignore" });
    const deadline = Date.now() + 30_000;
    while (Date.now() < deadline) {
      if (server.exitCode !== null) throw new Error(`generated Edition server exited with ${server.exitCode}`);
      try {
        const response = await fetch(`${baseURL}/healthz`);
        if (response.ok) return;
      } catch {
        // The Go command may still be compiling; keep polling until the bounded deadline.
      }
      await new Promise((resolve) => setTimeout(resolve, 200));
    }
    throw new Error("generated Edition server did not become healthy within 30 seconds");
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

  test("finds a generated collection and creates, edits, and deletes a record", async ({ page, request }) => {
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await page.goto(`${baseURL}/admin/resources`);
    await expect(page.getByRole("heading", { name: "Kaynaklar" })).toBeVisible();
    await expect(page.locator(".studio-sidebar")).toBeVisible();
    await expect(page.getByRole("link", { name: "HYDROGO Studio" })).toBeVisible();
    await expect(page.locator(".studio-sidebar")).toContainText("İÇERİK");
    const onboarding = page.locator("#admin-onboarding-dialog");
    await expect(onboarding).toBeVisible();
    await onboarding.getByRole("button", { name: "Bir daha gösterme" }).click();
    await expect(onboarding).toBeHidden();
    await expectNoA11yViolations(page);
    await page.context().addCookies([{ name: "hydro_admin_locale", value: "en", url: baseURL }]);
    await page.reload();
    await expect(page.getByRole("heading", { name: "Generated resources" })).toBeVisible();
    await expect(page.locator(".studio-sidebar")).toContainText("CONTENT");
    await page.context().clearCookies();
    await page.reload();
    await expect(page.getByRole("heading", { name: "Kaynaklar" })).toBeVisible();
    await page.getByRole("navigation").getByRole("link", { name: "Projects" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/project(?:\?.*)?$/);
    await expect(page.locator("h1", { hasText: "Projects" })).toBeVisible();
    await expect(page.getByRole("navigation").getByRole("link", { name: "Projects" })).toBeVisible();
    await expectNoA11yViolations(page);
    for (const family of ["hydrogo", "shadcn", "nova", "aurora"]) {
      for (const mode of ["light", "dark"]) {
        await page.evaluate(([theme, colorMode]) => localStorage.setItem("hydro-ui-theme-v2", `${theme}-${colorMode}`), [family, mode]);
        await page.reload();
        await expect(page.locator("html")).toHaveAttribute("data-ui-theme", family);
        await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
        await expectNoA11yViolations(page);
      }
    }
    await page.evaluate(() => localStorage.setItem("hydro-ui-theme-v2", "hydrogo-light"));
    await page.reload();
    await expect(page.locator("#filter-status option").first()).toHaveText("Tümü");
    for (const viewport of [{ width: 390, height: 844 }, { width: 768, height: 1024 }]) {
      await page.setViewportSize(viewport);
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
      expect(overflow).toBeLessThanOrEqual(1);
      await expect(page.locator(".table-wrap")).toBeVisible();
    }
    await page.setViewportSize({ width: 1280, height: 720 });
    const newRecord = page.getByRole("link", { name: "Yeni kayıt" });
    await newRecord.focus();
    await page.keyboard.press("Enter");
    const create = page.locator('form[action="/admin/resources/project/new"]');
    await expect(create.getByLabel("Title")).toBeVisible();
    await expect(create.locator('[name="id"]')).toHaveAttribute("type", "hidden");
    await expect(create.locator('label[for="resource-id"]')).toHaveCount(0);
    await expect(create.getByLabel("Project code")).toBeVisible();
    await create.locator('button[type="submit"]').click();
    await expect(create.getByLabel("Project code")).toBeFocused();
    expect(await create.getByLabel("Project code").evaluate((input: HTMLInputElement) => input.validity.valueMissing)).toBe(true);
    await create.locator('[name="title"]').fill("HydroGo E2E project");
    await create.locator('[name="project_code"]').fill("E2E-001");
    await create.locator('[name="status"]').selectOption("active");
    await create.locator('button[type="submit"]').click();
    await expect(page).toHaveURL(/\/admin\/resources\/project(?:\?.*)?$/);
    const recordLink = page.getByRole("link", { name: "HydroGo E2E project" });
    await expect(recordLink).toBeVisible();
    await page.screenshot({ path: test.info().outputPath("generated-list-desktop.png") });

    await page.getByLabel("Status").selectOption("active");
    await page.getByRole("button", { name: "Uygula" }).click();
    await expect(page.getByRole("link", { name: "HydroGo E2E project" })).toBeVisible();
    await page.getByLabel("Ara", { exact: true }).fill("no matching project");
    await page.getByRole("button", { name: "Uygula" }).click();
    await expect(page.getByText("Bu koleksiyonda henüz kayıt bulunmuyor.")).toBeVisible();
    await page.getByLabel("Ara", { exact: true }).fill("");
    await page.getByLabel("Status").selectOption("");
    await page.getByRole("button", { name: "Uygula" }).click();
    await expect(page.getByRole("link", { name: "HydroGo E2E project" })).toBeVisible();
    const recordPath = await recordLink.getAttribute("href");
    expect(recordPath).toMatch(/^\/admin\/resources\/project\//);

    await recordLink.click();
    await expect(page.locator('[name="title"]')).toHaveValue("HydroGo E2E project");
    await page.locator('[name="title"]').fill("HydroGo E2E updated");
    await page.locator(`form[action="${recordPath}"] button[type="submit"]`).click();
    await expect(page).toHaveURL(/\/admin\/resources\/project(?:\?.*)?$/);
    await expect(page.getByRole("link", { name: "HydroGo E2E updated" })).toBeVisible();

    await page.getByRole("link", { name: "HydroGo E2E updated" }).click();
    await page.locator(`form[action="${recordPath}/delete"] button[type="submit"]`).click();
    await expect(page).toHaveURL(/\/admin\/resources\/project(?:\?.*)?$/);
    await expect(page.getByText("HydroGo E2E updated")).toHaveCount(0);
    const deleted = await request.get(`${baseURL}${recordPath!.replace("/admin/resources", "/api/v1")}`);
    expect(deleted.status()).toBe(404);

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`${baseURL}/admin/resources/project/new`);
    await expect(page.getByLabel("Title")).toBeVisible();
    await page.screenshot({ path: test.info().outputPath("generated-form-mobile.png") });
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
    expect(overflow).toBeLessThanOrEqual(1);
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
    nodes: violation.nodes.map((node: any) => ({
      target: node.target,
      failureSummary: node.failureSummary,
    })),
  }))).toEqual([]);
}
