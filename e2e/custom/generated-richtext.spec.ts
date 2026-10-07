import { test, expect } from "@playwright/test";
import { spawn, type ChildProcess } from "node:child_process";
import net from "node:net";
import path from "node:path";

const repoRoot = path.resolve(process.cwd(), "..");
let server: ChildProcess | undefined;
let baseURL = "";

test.describe("generated rich-text Edition Admin", () => {
  test.beforeAll(async () => {
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawn("go", ["run", "./tests/e2e/support/generatedrichtext"], {
      cwd: repoRoot,
      stdio: "ignore",
      env: { ...process.env, HYDRO_E2E_ADDRESS: `127.0.0.1:${port}` },
    });
    const deadline = Date.now() + 30_000;
    while (Date.now() < deadline) {
      if (server.exitCode !== null) throw new Error(`generated rich-text fixture exited with ${server.exitCode}`);
      try {
        const response = await fetch(`${baseURL}/healthz`);
        if (response.ok) return;
      } catch {
        // Wait while the Go fixture compiles and starts.
      }
      await new Promise((resolve) => setTimeout(resolve, 200));
    }
    throw new Error("generated rich-text fixture did not become healthy within 30 seconds");
  }, 60_000);

  test.afterAll(async () => {
    if (server && server.exitCode === null) {
      server.kill("SIGTERM");
      await new Promise<void>((resolve) => {
        const timeout = setTimeout(resolve, 5_000);
        server?.once("exit", () => { clearTimeout(timeout); resolve(); });
      });
    }
  });

  test("Quill formats generated content and preserves it through API create/edit", async ({ page }) => {
    await page.goto(`${baseURL}/admin/resources/article/new`);
    await expect.poll(() => page.evaluate(() => typeof (window as any).HydroInitQuill)).toBe("function");
    const form = page.locator('form[action="/admin/resources/article/new"]');
    const editor = page.getByRole("textbox", { name: "Article body" }).locator(".ql-editor");
    await expect(editor).toBeVisible();
    await form.getByLabel("Title").fill("Generated rich text article");
    await editor.fill("A formatted generated article.");
    await editor.press("Control+A");
    await page.locator(".richtext-toolbar .ql-bold").click();

    const value = form.locator('textarea[name="body"]');
    await expect.poll(async () => {
      const parsed = JSON.parse(await value.inputValue());
      return parsed.ops?.some((operation: { attributes?: { bold?: boolean } }) => operation.attributes?.bold === true) ?? false;
    }).toBe(true);

    await form.getByRole("button", { name: "Kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    const recordURL = await page.getByRole("link", { name: "Generated rich text article" }).getAttribute("href");
    expect(recordURL).toBeTruthy();
    await page.goto(new URL(recordURL!, baseURL).toString());
    const editForm = page.locator('form[action^="/admin/resources/article/"]');
    const restoredEditor = page.getByRole("textbox", { name: "Article body" });
    await expect(restoredEditor.locator("strong")).toHaveText("A formatted generated article.");
    await expect.poll(async () => JSON.parse(await editForm.locator('textarea[name="body"]').inputValue()).ops[0].attributes?.bold).toBe(true);

    const editContent = restoredEditor.locator(".ql-editor");
    await editContent.fill("An updated formatted article.");
    await editContent.press("Control+A");
    const boldButton = page.locator(".richtext-toolbar .ql-bold");
    if (!(await boldButton.evaluate((button) => button.classList.contains("ql-active")))) await boldButton.click();
    await editForm.getByRole("button", { name: "Kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    await page.getByRole("link", { name: "Generated rich text article" }).click();
    await expect(page.getByRole("textbox", { name: "Article body" }).locator("strong")).toHaveText("An updated formatted article.");
  });

  test("a stale generated edit gets a conflict and keeps the latest saved value", async ({ browser }) => {
    const context = await browser.newContext();
    const first = await context.newPage();
    await first.goto(`${baseURL}/admin/resources/article/new`);
    const createForm = first.locator('form[action="/admin/resources/article/new"]');
    await createForm.getByLabel("Title").fill("Concurrent generated article");
    await createForm.getByRole("button", { name: "Kaydet" }).click();
    await expect(first).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    const recordURL = await first.getByRole("link", { name: "Concurrent generated article" }).getAttribute("href");
    expect(recordURL).toBeTruthy();

    const second = await context.newPage();
    await Promise.all([first.goto(new URL(recordURL!, baseURL).toString()), second.goto(new URL(recordURL!, baseURL).toString())]);
    const firstForm = first.locator('form[action^="/admin/resources/article/"]');
    const secondForm = second.locator('form[action^="/admin/resources/article/"]');
    await firstForm.getByLabel("Title").fill("Latest generated title");
    await firstForm.getByRole("button", { name: "Kaydet" }).click();
    await expect(first).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);

    await secondForm.getByLabel("Title").fill("Stale generated title");
    await secondForm.getByRole("button", { name: "Kaydet" }).click();
    await expect(second.locator("body")).toContainText("Bu kayıt başka biri tarafından güncellendi. Sayfayı yenileyip tekrar deneyin.");
    await second.goto(new URL(recordURL!, baseURL).toString());
    await expect(secondForm.getByLabel("Title")).toHaveValue("Latest generated title");
    await context.close();
  });

  test("generated relation and media fields round-trip through the shared Admin picker", async ({ page }) => {
    await page.goto(`${baseURL}/admin/resources/article/new`);
    const form = page.locator('form[action="/admin/resources/article/new"]');
    await form.getByLabel("Title").fill("Generated relation and media");
    await form.getByLabel("Author").selectOption("author-1");
    await form.getByRole("button", { name: "Kütüphaneden seç" }).click();
    const modal = page.getByRole("dialog", { name: "Medya kütüphanesi" });
    await expect(modal).toBeVisible();
    const choice = modal.locator(".admin-media-picker-item");
    await expect(choice).toHaveAttribute("data-asset-id", "fixture-image");
    await choice.click();
    await expect(modal).toBeHidden();
    await expect(form.locator('input[name="cover_id"]')).toHaveValue("fixture-image");
    await expect(form.locator(".media-select-field")).toHaveClass(/has-selected/);

    await form.getByRole("button", { name: "Kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    const recordLink = page.getByRole("link", { name: "Generated relation and media" });
    await expect(recordLink).toBeVisible();
    const recordPath = await recordLink.getAttribute("href");
    expect(recordPath).toBeTruthy();
    await page.goto(new URL(recordPath!, baseURL).toString());
    const editForm = page.locator(`form[action="${recordPath}"]`);
    await expect(editForm.getByLabel("Author")).toHaveValue("author-1");
    await expect(editForm.locator('input[name="cover_id"]')).toHaveValue("fixture-image");
    await expect(editForm.locator(".media-select-field")).toHaveClass(/has-selected/);
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
