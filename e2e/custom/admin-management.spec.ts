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
  test.describe.configure({ mode: "serial" });
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-admin-management-"));
    const database = path.join(tempDir, "edition.sqlite");
    execFileSync("go", ["run", "./cmd/hydro", "migrate", "--database-dialect", "sqlite", "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
    execFileSync("go", ["run", "./cmd/hydro", "create-admin", "--email", "admin@example.test", "--password-stdin", "--database-dialect", "sqlite", "--database-dsn", database], { cwd: repoRoot, input: `${adminPassword}\n`, stdio: ["pipe", "ignore", "pipe"] });
    execFileSync("go", ["run", "./tests/e2e/support/seedroles", "--database-dsn", database], { cwd: repoRoot, stdio: "pipe" });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawn("go", ["run", "./cmd/hydro", "serve", "--dev", "--database-dialect", "sqlite", "--database-dsn", database, "--address", `127.0.0.1:${port}`], { cwd: repoRoot, stdio: "ignore", env: { ...process.env, HYDRO_MEDIA_DIR: path.join(tempDir, "media") } });
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

  test("admin login opens the Edition management areas and rejects anonymous access", async ({ page, browser }) => {
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

      const unnamedControls = await page.locator('a,button,input:not([type="hidden"]),select,textarea,[role="button"],[contenteditable="true"]').evaluateAll((elements) =>
        elements.filter((element) => {
          const html = element as HTMLElement;
          if (html.hidden || html.getClientRects().length === 0 || getComputedStyle(html).display === "none" || getComputedStyle(html).visibility === "hidden") return false;
          const labelledBy = html.getAttribute("aria-labelledby")?.split(/\s+/).map((id) => document.getElementById(id)?.textContent ?? "").join(" ");
          const label = (html.id ? document.querySelector(`label[for="${CSS.escape(html.id)}"]`)?.textContent : "") || html.closest("label")?.textContent;
          const value = html instanceof HTMLInputElement && ["submit", "button", "reset"].includes(html.type) ? html.value : "";
          const name = [html.getAttribute("aria-label"), labelledBy, label, value, html.textContent, html.getAttribute("title")]
            .some((part) => part?.trim());
          return !name;
        }).map((element) => {
          const html = element as HTMLElement;
          return `${html.tagName.toLowerCase()}${html.id ? `#${html.id}` : ""}${html.getAttribute("name") ? `[name=${html.getAttribute("name")}]` : ""}`;
        }),
      );
      expect(unnamedControls, `${route} visible controls must have accessible names`).toEqual([]);

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

    await page.goto(`${baseURL}/admin/security`);
    const testBanAddress = "198.51.100.222";
    const banForm = page.locator('form[action="/admin/security/bans"]');
    await banForm.getByLabel("IP adresi veya CIDR").fill(testBanAddress);
    await banForm.getByLabel(/Süre/).fill("1h");
    await banForm.getByRole("button", { name: "IP engelle" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/security`);
    await expect(page.getByText(`${testBanAddress}/32`, { exact: true })).toBeVisible();
    const unbanForm = page.locator('form[action="/admin/security/bans/remove"]');
    await unbanForm.getByLabel("Engeli kaldırılacak adres veya CIDR").fill(testBanAddress);
    await unbanForm.getByRole("button", { name: "Engeli kaldır" }).click();
    await expect(page).toHaveURL(`${baseURL}/admin/security`);
    await expect(page.getByText(testBanAddress, { exact: true })).toHaveCount(0);

    const usersResponse = await page.goto(`${baseURL}/admin/users`);
    expect(usersResponse?.status()).toBe(200);
    const editorRow = page.locator(".studio-table tbody tr").filter({ hasText: "editor@example.test" });
    const permissionForm = editorRow.locator('form[action$="/permissions"]');
    const permissionToken = await permissionForm.locator('[name="_csrf"]').inputValue();
    const permissionAction = await permissionForm.getAttribute("action");
    expect(permissionAction).toBeTruthy();
    const permissionUpdate = await page.request.post(new URL(permissionAction!, baseURL).href, {
      form: { _csrf: permissionToken, "perm_admin.content.write": "allow" },
    });
    expect(await permissionUpdate.text()).toBe("");
    expect(permissionUpdate.status()).toBe(204);
    await page.reload();
    await expect(editorRow.locator('[name="perm_admin.content.write"]')).toHaveValue("allow");

    const roleForm = editorRow.locator('form[action$="/role"]');
    const roleToken = await roleForm.locator('[name="_csrf"]').inputValue();
    const roleAction = await roleForm.getAttribute("action");
    expect(roleAction).toBeTruthy();
    const roleUpdate = await page.request.post(new URL(roleAction!, baseURL).href, {
      form: { _csrf: roleToken, role: "viewer" },
    });
    expect(roleUpdate.status()).toBe(204);
    await page.reload();
    await expect(editorRow.locator('[name="role"]')).toHaveValue("viewer");

    const restoreRoleForm = editorRow.locator('form[action$="/role"]');
    const restoreRoleToken = await restoreRoleForm.locator('[name="_csrf"]').inputValue();
    const restoreRoleAction = await restoreRoleForm.getAttribute("action");
    const restoreRole = await page.request.post(new URL(restoreRoleAction!, baseURL).href, {
      form: { _csrf: restoreRoleToken, role: "editor" },
    });
    expect(restoreRole.status()).toBe(204);
    await page.reload();
    await expect(editorRow.locator('[name="role"]')).toHaveValue("editor");

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
    const faqBlock = editPage.locator("[data-block-row]").first();
    await faqBlock.locator('[name="block_type_0"]').selectOption("faq");
    await expect(faqBlock.locator(".faq-visual-builder")).toBeVisible();
    await faqBlock.locator(".faq-q-input").first().fill("Teslimat ne kadar sürer?");
    await faqBlock.locator(".faq-a-input").first().fill("Siparişiniz iki iş gününde ulaşır.");
    await faqBlock.locator(".faq-add-card-btn").click();
    await faqBlock.locator(".faq-q-input").nth(1).fill("İade yapabilir miyim?");
    await faqBlock.locator(".faq-a-input").nth(1).fill("Evet, 14 gün içinde iade edebilirsiniz.");
    await editPage.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+\?saved=1$/);
    const savedFAQBlock = page.locator('form[data-autosave-resource^="page:"] [data-block-row]').first();
    await expect(savedFAQBlock.locator('[name="block_type_0"]')).toHaveValue("faq");
    await expect(savedFAQBlock.locator(".faq-q-input").nth(1)).toHaveValue("İade yapabilir miyim?");

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
    const restoredFAQBlock = restoredEdit.locator("[data-block-row]").first();
    await restoredFAQBlock.locator('[name="block_type_0"]').selectOption("faq");
    await expect(restoredFAQBlock.locator(".faq-visual-builder")).toBeVisible();
    await restoredFAQBlock.locator(".faq-q-input").first().fill("Teslimat ne kadar sürer?");
    await restoredFAQBlock.locator(".faq-a-input").first().fill("Siparişiniz iki iş gününde ulaşır.");
    await restoredFAQBlock.locator(".faq-add-card-btn").click();
    await restoredFAQBlock.locator(".faq-q-input").nth(1).fill("İade yapabilir miyim?");
    await restoredFAQBlock.locator(".faq-a-input").nth(1).fill("Evet, 14 gün içinde iade edebilirsiniz.");
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
    expect(await publishedPage.text()).toContain("Teslimat ne kadar sürer?");
    expect(await publishedPage.text()).toContain("Evet, 14 gün içinde iade edebilirsiniz.");

    const mediaName = "hydro-picker-e2e.png";
    await page.goto(`${baseURL}/admin/media`);
    await page.locator("#media-file-input").setInputFiles({
      name: mediaName,
      mimeType: "image/png",
      buffer: Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg==", "base64"),
    });
    const uploadResponsePromise = page.waitForResponse((response) => response.url() === `${baseURL}/admin/media` && response.request().method() === "POST");
    await page.locator('#media-upload-form button[type="submit"]').click();
    const uploadResponse = await uploadResponsePromise;
    expect(uploadResponse.status()).toBe(303);
    await expect(page.locator(`.media-card[data-filename="${mediaName}"]`)).toBeVisible();

    await page.goto(`${baseURL}/admin/news/new`);
    const coverField = page.locator('#news-form .media-select-field');
    await coverField.getByRole("button", { name: "Kütüphaneden seç" }).click();
    const mediaModal = page.locator("#admin-media-picker-modal");
    await expect(mediaModal).toBeVisible();
    const mediaChoice = mediaModal.getByRole("button", { name: mediaName });
    await expect(mediaChoice).toBeVisible();
    await mediaChoice.click();
    const coverID = page.locator('#news-form [name="cover_media_id"]');
    await expect(coverID).toHaveValue(/.+/);
    const selectedMediaID = await coverID.inputValue();
    await expect(coverField.locator(".media-select-preview img")).toHaveAttribute("src", `/media/${selectedMediaID}`);
    await coverField.getByRole("button", { name: "Kaldır" }).click();
    await expect(coverID).toHaveValue("");
    await coverField.getByRole("button", { name: "Kütüphaneden seç" }).click();
    await mediaModal.getByRole("button", { name: mediaName }).click();
    await expect(coverID).toHaveValue(selectedMediaID);

    await page.locator('#news-form [name="title"]').fill("E2E media picker story");
    await page.locator('#news-form [name="slug"]').fill("e2e-media-picker-story");
    await page.locator('#news-form .richtext-editor [contenteditable="true"]').fill("Created from the media picker browser flow.");
    await page.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/news\/[^/?]+$/);
    await expect(page.locator('#news-edit-form [name="cover_media_id"]')).toHaveValue(selectedMediaID);

    await page.goto(`${baseURL}/admin/pages/new`);
    const conflictCreate = page.locator('form[action="/admin/pages"]');
    await conflictCreate.locator('[name="slug"]').fill("e2e-conflict-page");
    await conflictCreate.locator('[name="title"]').fill("Conflict draft");
    await conflictCreate.locator(".richtext-editor [contenteditable=true]").fill("Initial conflict content.");
    await conflictCreate.getByRole("button", { name: "Taslak oluştur" }).click();
    await expect(page).toHaveURL(/\/admin\/pages\/[^/?]+$/);

    const staleEditor = await page.context().newPage();
    await staleEditor.goto(page.url());
    const currentEditor = page.locator('form[data-autosave-resource^="page:"]');
    await currentEditor.locator('[name="title"]').fill("Current page version");
    await currentEditor.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\?saved=1$/);

    const staleForm = staleEditor.locator('form[data-autosave-resource^="page:"]');
    await staleForm.locator('[name="title"]').fill("Stale page version");
    const staleSave = staleEditor.waitForResponse((response) => response.request().method() === "POST" && response.status() === 409);
    await staleForm.getByRole("button", { name: "Taslağı kaydet" }).click();
    const conflictResponse = await staleSave;
    expect(await conflictResponse.text()).toContain("Bu içerik başka biri tarafından güncellendi. Sayfayı yenileyip tekrar deneyin.");
    await staleEditor.goto(`${baseURL}${new URL(page.url()).pathname}`);
    await expect(staleEditor.locator('form[data-autosave-resource^="page:"] [name="title"]')).toHaveValue("Current page version");
    await staleEditor.close();

    const viewerContext = await browser.newContext();
    const viewerPage = await viewerContext.newPage();
    await viewerPage.goto(`${baseURL}/login`);
    await viewerPage.getByLabel("Email address").fill("viewer@example.test");
    await viewerPage.getByLabel("Password").fill(adminPassword);
    await viewerPage.getByRole("button", { name: "Sign in" }).click();
    await expect(viewerPage).toHaveURL(/\/admin(?:\?.*)?$/);
    expect((await viewerPage.request.get(`${baseURL}/auth/me`)).status()).toBe(200);

    await page.goto(`${baseURL}/admin/users`);
    const viewerRow = page.locator(".studio-table tbody tr").filter({ hasText: "viewer@example.test" });
    const disableForm = viewerRow.locator('form[action$="/disable"]');
    const disableToken = await disableForm.locator('[name="_csrf"]').inputValue();
    const disableAction = await disableForm.getAttribute("action");
    expect(disableAction).toBeTruthy();
    const disabled = await page.request.post(new URL(disableAction!, baseURL).href, { form: { _csrf: disableToken } });
    expect(disabled.status()).toBe(204);
    expect((await viewerPage.request.get(`${baseURL}/auth/me`)).status()).toBe(401);
    await page.reload();
    await expect(viewerRow).toContainText("Pasif");
    await viewerContext.close();
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
