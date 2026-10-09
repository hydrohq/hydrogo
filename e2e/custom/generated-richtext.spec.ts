import { test, expect } from "@playwright/test";
import { type ChildProcess } from "node:child_process";
import net from "node:net";
import path from "node:path";
import { spawnServer, stopServer } from "./server-process";

const repoRoot = path.resolve(process.cwd(), "..");
let server: ChildProcess | undefined;
let baseURL = "";

test.describe("generated rich-text Edition Admin", () => {
  test.beforeAll(async () => {
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawnServer("go", ["run", "./tests/e2e/support/generatedrichtext"], {
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
    await stopServer(server);
  });

  test("Quill formats generated content and preserves it through API create/edit", async ({ page }) => {
    await page.goto(`${baseURL}/admin/resources/article/new`);
    await dismissOnboarding(page);
    await expect.poll(() => page.evaluate(() => typeof (window as any).HydroInitQuill)).toBe("function");
    const form = page.locator('form[action="/admin/resources/article/new"]');
    const editor = page.locator(".richtext-editor .ql-editor");
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

    const editContent = restoredEditor;
    await editContent.fill("An updated formatted article.");
    await editContent.press("Control+A");
    const boldButton = page.locator(".richtext-toolbar .ql-bold");
    if (!(await boldButton.evaluate((button) => button.classList.contains("ql-active")))) await boldButton.click();
    await editForm.getByRole("button", { name: "Kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    await page.getByRole("link", { name: "Generated rich text article" }).click();
    await expect(page.getByRole("textbox", { name: "Article body" }).locator("strong")).toHaveText("An updated formatted article.");
  });

  test("generated blocks lazy-load the Edition editor asset and round-trip JSON", async ({ browser, page, request }) => {
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    const mobileContext = await browser.newContext({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });
    const mobilePage = await mobileContext.newPage();
    await mobilePage.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await mobilePage.goto(`${baseURL}/admin/resources/article/new`);
    await dismissOnboarding(mobilePage);
    const mobileMount = mobilePage.locator('[data-editorjs-mount="blocks"]');
    const mobileEditor = mobileMount.locator('.ce-paragraph[contenteditable="true"]').first();
    await expect(mobileEditor).toBeVisible();
    await expect.poll(() => mobilePage.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await expectNoA11yViolations(mobilePage);
    await mobileEditor.fill("Touch generated paragraph.");
    await mobileEditor.tap();
    const mobileAddBlock = mobileMount.locator(".ce-toolbar__plus");
    await expect(mobileAddBlock).toBeVisible();
    await mobileAddBlock.tap();
    const mobileTextTool = mobilePage.locator(".ce-popover-item").filter({ hasText: "Text" });
    await expect(mobileTextTool).toBeVisible();
    await mobileTextTool.click();
    await expect(mobileMount.locator('.ce-paragraph[contenteditable="true"]')).toHaveCount(2);
    await mobilePage.screenshot({ path: test.info().outputPath("generated-block-editor-mobile.png"), fullPage: true });
    await mobileContext.close();

    await page.goto(`${baseURL}/admin/resources/article/new`);
    await dismissOnboarding(page);
    for (const family of ["hydrogo", "shadcn", "nova", "aurora"]) {
      for (const mode of ["light", "dark"]) {
        await page.evaluate(([theme, colorMode]) => localStorage.setItem("hydro-ui-theme-v2", `${theme}-${colorMode}`), [family, mode]);
        const response = await page.goto(`${baseURL}/admin/resources/article/new`);
        expect(response?.status(), `generated block editor should render in ${family}-${mode}`).toBe(200);
        await dismissOnboarding(page);
        await expect(page.locator("html")).toHaveAttribute("data-ui-theme", family);
        await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
        await expect(page.locator('[data-editorjs-mount="blocks"] .ce-paragraph[contenteditable="true"]').first()).toBeVisible();
        await expectNoA11yViolations(page);
      }
    }
    await page.evaluate(() => localStorage.setItem("hydro-ui-theme-v2", "hydrogo-light"));

    await page.goto(`${baseURL}/admin/resources/article`);
    await expect(page.locator('script[src="/test/editorjs.js"]')).toHaveCount(0);
    await page.goto(`${baseURL}/admin/resources/article/new`);
    const mount = page.locator('[data-editorjs-mount="blocks"]');
    const editor = mount.locator('.ce-paragraph[contenteditable="true"]').first();
    await expect(editor).toBeVisible();
    await expectNoA11yViolations(page);
    await editor.fill("A generated block paragraph.");
    await editor.click();
    const addBlock = mount.locator(".ce-toolbar__plus");
    await expect(addBlock).toBeVisible();
    await addBlock.click();
    const paragraphTool = page.locator(".ce-popover-item").filter({ hasText: "Text" });
    await expect(paragraphTool).toBeVisible();
    await paragraphTool.click();
    const secondEditor = mount.locator('.ce-paragraph[contenteditable="true"]').nth(1);
    await expect(secondEditor).toBeVisible();
    await secondEditor.fill("A second generated paragraph.");
    const form = page.locator('form[action="/admin/resources/article/new"]');
    await form.getByLabel("Title").fill("Generated blocks article");
    const value = form.locator('textarea[name="blocks"]');
    await expect.poll(async () => {
      try {
        return JSON.parse(await value.inputValue()).blocks?.map((block: { data: { text: string } }) => block.data.text) ?? [];
      } catch {
        return [];
      }
    }).toEqual(["A generated block paragraph.", "A second generated paragraph."]);
    await form.getByRole("button", { name: "Kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);

    const records = await (await request.get(`${baseURL}/api/v1/article`)).json();
    const record = records.find((candidate: { id: string; title: string }) => candidate.title === "Generated blocks article");
    const expected = JSON.parse(record?.blocks ?? "null");
    expect(expected.blocks?.map((block: { type: string; data: { text: string } }) => ({ type: block.type, text: block.data.text }))).toEqual([
      { type: "paragraph", text: "A generated block paragraph." },
      { type: "paragraph", text: "A second generated paragraph." },
    ]);
    const href = page.getByRole("link", { name: "Generated blocks article" });
    await href.click();
    const editValue = page.locator('textarea[name="blocks"]');
    await expect.poll(async () => JSON.parse(await editValue.inputValue())).toEqual(expected);
    const restored = page.locator('[data-editorjs-mount="blocks"] .ce-paragraph[contenteditable="true"]');
    await expect(restored.nth(0)).toHaveText("A generated block paragraph.");
    await expect(restored.nth(1)).toHaveText("A second generated paragraph.");

    await page.context().setExtraHTTPHeaders({ "X-E2E-Resource-Access": "none" });
    await page.goto(`${baseURL}/admin/resources/article`);
    await expect(page.getByRole("link", { name: "Yeni kayıt" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Generated blocks article" })).toHaveCount(0);
    await expect(page.getByText("Generated blocks article", { exact: true })).toBeVisible();
    expect((await page.goto(`${baseURL}/admin/resources/article/new`))?.status()).toBe(403);
    expect((await page.goto(`${baseURL}/admin/resources/article/${record.id}`))?.status()).toBe(200);
    await expect(page.getByRole("heading", { name: /Articles · Generated blocks article/ })).toBeVisible();
    await expect(page.locator("form")).toHaveCount(0);
    for (const path of [
      "/admin/resources/article/new",
      `/admin/resources/article/${record.id}`,
      `/admin/resources/article/${record.id}/delete`,
    ]) {
      const status = await page.evaluate(async (target) => {
        const response = await fetch(target, {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body: "_csrf=generated-richtext-e2e-csrf&title=Denied+change",
        });
        return response.status;
      }, path);
      expect(status, `${path} must be denied before API mutation`).toBe(403);
    }
    const unchanged = await (await request.get(`${baseURL}/api/v1/article/${record.id}`)).json();
    expect(unchanged.title).toBe("Generated blocks article");
    expect(JSON.parse(unchanged.blocks).blocks).toEqual(expected.blocks);
  });

  test("a stale generated edit gets a conflict and keeps the latest saved value", async ({ browser }) => {
    const context = await browser.newContext();
    const first = await context.newPage();
    await first.goto(`${baseURL}/admin/resources/article/new`);
    await dismissOnboarding(first);
    const createForm = first.locator('form[action="/admin/resources/article/new"]');
    await createForm.getByLabel("Title").fill("Concurrent generated article");
    await createForm.getByRole("button", { name: "Kaydet" }).click();
    await expect(first).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    const recordURL = await first.getByRole("link", { name: "Concurrent generated article" }).getAttribute("href");
    expect(recordURL).toBeTruthy();

    const second = await context.newPage();
    await Promise.all([first.goto(new URL(recordURL!, baseURL).toString()), second.goto(new URL(recordURL!, baseURL).toString())]);
    await dismissOnboarding(second);
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

  test("generated relation and media fields round-trip through the shared Admin picker", async ({ page, request }) => {
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await page.goto(`${baseURL}/admin/resources/article/new`);
    await dismissOnboarding(page);
    const form = page.locator('form[action="/admin/resources/article/new"]');
    await form.getByLabel("Title").fill("Generated relation and media");
    const tags = form.locator('[data-field-name="tags"]');
    await tags.getByRole("button", { name: "Tags: madde ekle" }).click();
    const firstTag = tags.locator(".item-list-input").nth(0);
    await firstTag.fill("featured");
    await firstTag.press("Enter");
    await tags.locator(".item-list-input").nth(1).fill("editorial");
    await form.getByLabel("Author").selectOption("author-1");
    await form.getByRole("button", { name: "Kütüphaneden seç" }).click();
    const modal = page.getByRole("dialog", { name: "Medya kütüphanesi" });
    await expect(modal).toBeVisible();
    await expectNoA11yViolations(page);
    const choice = modal.locator(".admin-media-picker-item");
    await expect(choice).toHaveAttribute("data-asset-id", "fixture-image");
    await choice.focus();
    await page.keyboard.press("Enter");
    await expect(modal).toBeHidden();
    await expect(form.locator('input[name="cover_id"]')).toHaveValue("fixture-image");
    await expect(form.locator(".media-select-field")).toHaveClass(/has-selected/);

    await form.getByRole("button", { name: "Kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    const recordLink = page.getByRole("link", { name: "Generated relation and media" });
    await expect(recordLink).toBeVisible();
    const recordPath = await recordLink.getAttribute("href");
    expect(recordPath).toBeTruthy();
    const recordAPI = recordPath!.replace("/admin/resources", "/api/v1");
    await expect((await request.get(`${baseURL}${recordAPI}`)).json()).resolves.toMatchObject({
      author_id: "author-1",
      cover_id: "fixture-image",
      tags: ["featured", "editorial"],
    });
    await page.goto(new URL(recordPath!, baseURL).toString());
    const editForm = page.locator(`form[action="${recordPath}"]`);
    await expect(editForm.locator('[data-field-name="tags"] .item-list-input')).toHaveCount(2);
    await expect(editForm.locator('[data-field-name="tags"] .item-list-input').nth(0)).toHaveValue("featured");
    await expect(editForm.locator('[data-field-name="tags"] .item-list-input').nth(1)).toHaveValue("editorial");
    await expect(editForm.getByLabel("Author")).toHaveValue("author-1");
    await expect(editForm.locator('input[name="cover_id"]')).toHaveValue("fixture-image");
    await expect(editForm.locator(".media-select-field")).toHaveClass(/has-selected/);
  });

  test("generated localized draft publishes without changing the default locale", async ({ page, request }) => {
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await page.goto(`${baseURL}/admin/resources/story/story-1?locale=tr`);
    await dismissOnboarding(page);
    await expect(page.getByRole("navigation", { name: "İçerik dili" })).toBeVisible();
    const title = page.getByLabel("Title · tr");
    await expect(title).toHaveValue("Türkçe taslak");
    await expectNoA11yViolations(page);

    await title.fill("Onaylanan Türkçe başlık");
    await page.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/locale=tr/);
    await expect(page.getByLabel("Title · tr")).toHaveValue("Onaylanan Türkçe başlık");
    await page.getByRole("button", { name: "Yayımla" }).click();
    await expect(page).toHaveURL(/locale=tr/);
    await expect(page.getByLabel("Title · tr")).toHaveValue("Onaylanan Türkçe başlık");

    await page.goto(`${baseURL}/admin/resources/story?locale=tr`);
    await expect(page.getByRole("link", { name: "Onaylanan Türkçe başlık" })).toBeVisible();
    const defaultRecord = await request.get(`${baseURL}/api/v1/story/story-1`);
    await expect(defaultRecord.json()).resolves.toMatchObject({ title: "Published story", version: 1 });
  });

  test("generated draft review and publish flow keeps working copy separate", async ({ page, request }) => {
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await page.goto(`${baseURL}/admin/resources/story/drafts`);
    await dismissOnboarding(page);
    await expect(page.locator(".studio-sidebar")).toBeVisible();
    await expect(page.locator(".studio-sidebar")).toContainText("İÇERİK");
    await expect(page.getByRole("heading", { name: "Taslaklar · Stories" }).first()).toBeVisible();
    await expect(page.getByRole("button", { name: "Review copy" })).toBeVisible();
    await page.screenshot({ path: test.info().outputPath("generated-draft-inbox.png") });
    await expectNoA11yViolations(page);

    await page.getByRole("button", { name: "Review copy" }).click();
    const editor = page.locator('form[action^="/admin/resources/story/story-1/draft"]');
    await expect(editor.locator('input[name="title"]')).toHaveValue("Review copy");
    await page.screenshot({ path: test.info().outputPath("generated-draft-editor.png") });
    await editor.locator('input[name="title"]').fill("Approved story copy");
    await editor.getByRole("button", { name: "Taslağı kaydet" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/story\/story-1(?:\?locale=en)?$/);
    await expect(page.locator('form[action^="/admin/resources/story/story-1/draft"] input[name="title"]')).toHaveValue("Approved story copy");
    await expectNoA11yViolations(page);

    await page.getByRole("button", { name: "Yayımla" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/story\/story-1(?:\?locale=en)?$/);
    const published = await request.get(`${baseURL}/api/v1/story/story-1`);
    expect(published.status()).toBe(200);
    await expect(published.json()).resolves.toMatchObject({ title: "Approved story copy", status: "published", version: 2 });
    await page.goto(`${baseURL}/admin/resources/story`);
    await expect(page.getByRole("link", { name: "Approved story copy" })).toBeVisible();
  });

  test("generated draft discard removes only the unpublished working copy", async ({ page, request }) => {
    await page.addInitScript({ path: path.join(repoRoot, "e2e/node_modules/axe-core/axe.min.js") });
    await page.goto(`${baseURL}/admin/resources/story/drafts`);
    await dismissOnboarding(page);
    await expect(page.getByRole("button", { name: "Discard-only draft" })).toBeVisible();
    await expectNoA11yViolations(page);

    await page.getByRole("button", { name: "Discard-only draft" }).click();
    let discardConfirmation = "";
    page.once("dialog", async (dialog) => {
      discardConfirmation = dialog.message();
      await dialog.dismiss();
    });
    await page.getByRole("button", { name: "Taslağı sil" }).click();
    expect(discardConfirmation).toBe("Yalnızca yayımlanmamış taslak silinecek. Devam edilsin mi?");
    const draftsAfterCancel = await request.get(`${baseURL}/admin/resources/story/drafts`);
    expect(draftsAfterCancel.status()).toBe(200);
    expect(await draftsAfterCancel.text()).toContain("Discard-only draft");
    await expect(page.getByRole("button", { name: "Taslağı sil" })).toBeVisible();
    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "Taslağı sil" }).click();
    await expect(page).toHaveURL(/\/admin\/resources\/story\/drafts(?:\?locale=en)?$/);
    await expect(page.getByRole("button", { name: "Discard-only draft" })).toHaveCount(0);
    await expectNoA11yViolations(page);
    const unpublished = await request.get(`${baseURL}/api/v1/story/story-discard`);
    expect(unpublished.status()).toBe(404);
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

async function dismissOnboarding(page: import("@playwright/test").Page): Promise<void> {
  const onboarding = page.locator("#admin-onboarding-dialog");
  await page.waitForTimeout(500);
  if (await onboarding.isVisible()) {
    await onboarding.getByRole("button", { name: "Bir daha gösterme" }).click();
    await expect(onboarding).toBeHidden();
  }
}
