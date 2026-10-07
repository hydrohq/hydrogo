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
    execFileSync("go", ["run", "./tests/e2e/support/seedroles", "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
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

  test("editor and viewer sessions enforce the Edition role boundary", async ({ browser }) => {
    for (const account of [
      { email: "editor@example.test", expectedRole: "editor", allowedPath: "/admin/pages/new" },
      { email: "viewer@example.test", expectedRole: "viewer", allowedPath: "/admin/pages/new" },
    ]) {
      const context = await browser.newContext();
      const page = await context.newPage();
      await page.goto(`${baseURL}/login`);
      await page.getByLabel("Email address").fill(account.email);
      await page.getByLabel("Password").fill(adminPassword);
      await page.getByRole("button", { name: "Sign in" }).click();
      await expect(page).toHaveURL(/\/admin(?:\?.*)?$/);

      const allowed = await page.goto(`${baseURL}${account.allowedPath}`);
      if (account.expectedRole === "editor") {
        expect(allowed?.status()).toBe(200);
        await expect(page.locator(`body[data-studio-role="${account.expectedRole}"]`)).toBeVisible();
        await expect(page.locator('.studio-sidebar a[href="/admin/pages"]')).toBeVisible();
        const createForm = page.locator('form[action="/admin/pages"]');
        await createForm.locator('[name="slug"]').fill("role-editor-page");
        await createForm.locator('[name="title"]').fill("Editor session page");
        await createForm.locator(".richtext-editor [contenteditable=true]").fill("Created by an authenticated Edition editor.");
        await createForm.getByRole("button", { name: "Taslak oluştur" }).click();
        await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);
      } else {
        expect(allowed?.status()).toBe(403);
      }

      if (account.expectedRole === "editor") {
        await expect(page.locator('.studio-sidebar a[href="/admin/users"]')).toHaveCount(0);
      }
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
      ]) {
        const denied = await page.goto(`${baseURL}${route}`);
        expect(denied?.status(), `${account.expectedRole} must be denied ${route}`).toBe(403);
      }
      const mutation = await page.request.post(`${baseURL}/admin/pages`, { form: { slug: "role-denied", title: "Role denied" } });
      expect(mutation.status()).toBe(403);
      await context.close();
    }
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

      const mutations = await page.locator('form[method="post"]').evaluateAll((forms) =>
        forms.map((form) => {
          const element = form as HTMLFormElement;
          return {
            action: element.action || location.href,
            csrf: element.querySelector<HTMLInputElement>('input[name="_csrf"]')?.value ?? "",
          };
        }),
      );
      for (const mutation of mutations) {
        expect(mutation.csrf, `${route} POST form must contain a CSRF token`).not.toBe("");
        const target = new URL(mutation.action);
        expect(target.origin, `${route} mutation target must stay on this Edition`).toBe(new URL(baseURL).origin);
        const deniedMutation = await page.request.post(target.href, { form: {} });
        expect(deniedMutation.status(), `CSRF-less POST to ${target.pathname} must be denied`).toBe(403);
      }
    }

    const editorResponse = await page.goto(`${baseURL}/admin/pages/new`);
    await expect(page.getByRole("group", { name: "Sayfa düzeni" })).toBeVisible();
    await expect(page.getByText("Boş sayfa")).toBeVisible();
    await expect(page.getByText("Hizmet tanıtımı")).toBeVisible();
    const picker = page.locator(".ds-layout-picker");
    await expect(picker.locator("input[type=radio][value=blank]")).toBeChecked();
    const pickerLayout = await page.locator(".ds-layout-picker__options").evaluate((element) => {
      const picker = getComputedStyle(element);
      const fieldset = getComputedStyle(element.parentElement!);
      const finalOption = getComputedStyle(element.lastElementChild!);
      return { columns: picker.gridTemplateColumns, span: fieldset.gridColumnEnd, finalOptionSpan: finalOption.gridColumnEnd };
    });
    expect(pickerLayout.columns.split(" ").length).toBe(2);
    expect(pickerLayout.span).toBe("-1");
    expect(pickerLayout.finalOptionSpan).toBe("auto");
    const editorHTML = await editorResponse?.text() ?? "";
    const inlineLayoutCSS = [...editorHTML.matchAll(/<style(?:\s[^>]*)?>([\s\S]*?)<\/style>/g)]
      .map((match) => match[1])
      .find((style) => style.includes(".ds-layout-picker"));
    expect(inlineLayoutCSS, "Admin shell must include the layout picker styles").toBeTruthy();
    const styleHash = `sha256-${Buffer.from(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(inlineLayoutCSS!))).toString("base64")}`;
    expect(editorResponse?.headers()["content-security-policy"]).toContain(`'${styleHash}'`);
    await page.getByLabel("Başarı hikâyesi").check();
    await expect(page.locator(".ds-layout-picker input[value=case_study]")).toBeChecked();
    await page.setViewportSize({ width: 390, height: 844 });
    const mobileLayout = await page.locator(".ds-layout-picker__options").evaluate((element) =>
      getComputedStyle(element).gridTemplateColumns,
    );
    expect(mobileLayout.split(" ").length).toBe(1);

    await page.setViewportSize({ width: 1280, height: 900 });
    await page.getByLabel("Boş sayfa").check();
    const createPage = page.locator('form[action="/admin/pages"]');
    await expect.poll(() => page.evaluate(() => typeof (window as any).HydroInitQuill)).toBe("function");
    await expect(createPage.locator(".richtext-editor [contenteditable=true]")).toBeVisible();
    await createPage.locator('[name="slug"]').fill("e2e-editorial-page");
    await createPage.locator('[name="title"]').fill("E2E editorial draft");
    await createPage.locator(".richtext-editor [contenteditable=true]").fill("Draft body for the editorial browser journey.");
    await createPage.getByRole("button", { name: "Taslak oluştur" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);

    const editPage = page.locator('form[data-autosave-resource^="page:"]');
    await editPage.locator('[name="title"]').fill("E2E editorial published");
    await editPage.locator(".richtext-editor [contenteditable=true]").fill("Published body from the editorial browser journey.");
    await editPage.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+\?saved=1$/);
    await expect(page.locator('form[data-autosave-resource^="page:"] [name="title"]')).toHaveValue("E2E editorial published");

    const originalPagePath = new URL(page.url()).pathname;
    await page.goto(`${baseURL}${originalPagePath}?revision=1`);
    const revisionPanel = page.locator("[data-revision-diff]");
    await expect(revisionPanel).toBeVisible();
    await revisionPanel.locator("summary").click();
    await expect(revisionPanel.getByRole("button", { name: "Bu sürümü geri yükle" })).toBeVisible();
    page.once("dialog", (dialog) => dialog.accept());
    await revisionPanel.getByRole("button", { name: "Bu sürümü geri yükle" }).click();
    await expect(page).toHaveURL(`${baseURL}${originalPagePath}`);
    await expect(page.locator('form[data-autosave-resource^="page:"] [name="title"]')).toHaveValue("E2E editorial draft");

    const restoredEdit = page.locator('form[data-autosave-resource^="page:"]');
    await restoredEdit.locator('[name="title"]').fill("E2E editorial published");
    await restoredEdit.locator(".richtext-editor [contenteditable=true]").fill("Published body from the editorial browser journey.");
    await restoredEdit.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+\?saved=1$/);

    await page.goto(`${baseURL}${originalPagePath}?compare=en`);
    const translation = page.locator("[data-translation-editor]");
    await expect(translation).toBeVisible();
    await translation.getByLabel("Başlık çevirisi").fill("E2E editorial translated");
    await translation.getByLabel("Metin çevirisi").fill("Translated body from the editorial browser journey.");
    await translation.getByRole("button", { name: "Çeviriyi kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);
    await expect(page.locator('form[data-autosave-resource^="page:"] [name="title"]')).toHaveValue("E2E editorial translated");

    await page.goto(`${baseURL}${originalPagePath}`);
    await page.locator('form[action$="/publish"] button[type="submit"]').click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);
    const publishedPage = await page.request.get(`${baseURL}/public/pages/tr/e2e-editorial-page/html`);
    expect(publishedPage.status()).toBe(200);
    await expect(publishedPage).toBeOK();
    expect(await publishedPage.text()).toContain("E2E editorial published");
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
