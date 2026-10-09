import { test, expect } from "@playwright/test";
import type { ChildProcess } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { runCommand, spawnServer, stopServer } from "./server-process";

const repoRoot = path.resolve(process.cwd(), "..");
let server: ChildProcess | undefined;
let tempDir = "";
let baseURL = "";

test.describe("generated Edition Admin CRUD", () => {
  test.beforeAll(async () => {
    tempDir = await mkdtemp(path.join(os.tmpdir(), "hydro-admin-crud-"));
    const dialect = process.env.HYDRO_E2E_DATABASE_DIALECT ?? "sqlite";
    const database = process.env.HYDRO_E2E_DATABASE_DSN ?? path.join(tempDir, "edition.sqlite");
    runCommand("go", ["run", "./examples/project-edition/cmd/server", "migrate", "--dialect", dialect, "--dsn", database], { cwd: repoRoot, stdio: "pipe" });
    const port = await unusedPort();
    baseURL = `http://127.0.0.1:${port}`;
    server = spawnServer("go", ["run", "./examples/project-edition/cmd/server", "serve", "--dialect", dialect, "--dsn", database, "--address", `127.0.0.1:${port}`], { cwd: repoRoot, stdio: "ignore" });
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
    await stopServer(server);
    if (tempDir) await rm(tempDir, { recursive: true, force: true });
  });

  test("finds a generated collection and creates, edits, and deletes a record", async ({ page, request }) => {
    test.setTimeout(240_000);
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
    for (const family of ["hydrogo", "shadcn", "nova", "aurora"]) {
      for (const mode of ["light", "dark"]) {
        await page.evaluate(([theme, colorMode]) => localStorage.setItem("hydro-ui-theme-v2", `${theme}-${colorMode}`), [family, mode]);
        const formResponse = await page.goto(`${baseURL}/admin/resources/project/new`);
        expect(formResponse?.status(), `create form should render in ${family}-${mode}`).toBe(200);
        await expect(page.locator("html")).toHaveAttribute("data-ui-theme", family);
        await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
        const invalidCreate = page.locator('form[action="/admin/resources/project/new"]');
        await expect(invalidCreate.getByLabel("Title")).toBeVisible();
        await expect(invalidCreate.locator('[name="id"]')).toHaveAttribute("type", "hidden");
        await expect(invalidCreate.locator('label[for="resource-id"]')).toHaveCount(0);
        await expect(invalidCreate.getByLabel("Project code")).toBeVisible();
        await invalidCreate.getByLabel("Title").fill("HydroGo E2E project");
        const validationResponse = page.waitForResponse((response) => response.request().method() === "POST" && response.url().endsWith("/admin/resources/project/new"));
        await invalidCreate.evaluate((form: HTMLFormElement) => HTMLFormElement.prototype.submit.call(form));
        expect((await validationResponse).status(), `validation should render in ${family}-${mode}`).toBe(422);
        await expect(invalidCreate.getByRole("alert")).toContainText("Bu alanları kontrol edin");
        await expect(invalidCreate.getByRole("link", { name: /Project code/ })).toHaveAttribute("href", "#project_code");
        await expect(invalidCreate.getByLabel("Title")).toHaveValue("HydroGo E2E project");
        await expect(invalidCreate.getByLabel("Project code")).toHaveValue("");
        await expectNoA11yViolations(page);
      }
    }
    await page.evaluate(() => localStorage.setItem("hydro-ui-theme-v2", "hydrogo-light"));
    await page.goto(`${baseURL}/admin/resources/project/new`);
    const create = page.locator('form[action="/admin/resources/project/new"]');
    await create.locator('button[type="submit"]').click();
    await expect(create.getByLabel("Project code")).toBeFocused();
    expect(await create.getByLabel("Project code").evaluate((input: HTMLInputElement) => input.validity.valueMissing)).toBe(true);
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

    for (const family of ["hydrogo", "shadcn", "nova", "aurora"]) {
      for (const mode of ["light", "dark"]) {
        await page.evaluate(([theme, colorMode]) => localStorage.setItem("hydro-ui-theme-v2", `${theme}-${colorMode}`), [family, mode]);
        const response = await page.goto(`${baseURL}${recordPath}`);
        expect(response?.status(), `${recordPath} should render in ${family}-${mode}`).toBe(200);
        await expect(page.locator("html")).toHaveAttribute("data-ui-theme", family);
        await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
        await expectNoA11yViolations(page);
      }
    }
    await page.goto(`${baseURL}/admin/resources/project`);

    await recordLink.click();
    await expect(page.locator('[name="title"]')).toHaveValue("HydroGo E2E project");
    await page.locator('[name="title"]').fill("HydroGo E2E updated");
    await page.locator(`form[action="${recordPath}"] button[type="submit"]`).click();
    await expect(page).toHaveURL(/\/admin\/resources\/project(?:\?.*)?$/);
    await expect(page.getByRole("link", { name: "HydroGo E2E updated" })).toBeVisible();

    await page.getByRole("link", { name: "HydroGo E2E updated" }).click();
    let deleteConfirmation = "";
    page.once("dialog", async (dialog) => {
      deleteConfirmation = dialog.message();
      await dialog.dismiss();
    });
    await page.locator(`form[action="${recordPath}/delete"] button[type="submit"]`).click();
    expect(deleteConfirmation).toBe("Bu kayıt kalıcı olarak silinecek. Devam edilsin mi?");
    expect((await request.get(`${baseURL}${recordPath!.replace("/admin/resources", "/api/v1")}`)).status()).toBe(200);
    page.once("dialog", (dialog) => dialog.accept());
    await page.locator(`form[action="${recordPath}/delete"] button[type="submit"]`).click();
    await expect(page).toHaveURL(/\/admin\/resources\/project(?:\?.*)?$/);
    await expect(page.getByText("HydroGo E2E updated")).toHaveCount(0);
    const deleted = await request.get(`${baseURL}${recordPath!.replace("/admin/resources", "/api/v1")}`);
    expect(deleted.status()).toBe(404);

    await page.goto(`${baseURL}/admin/resources/milestone`);
    await expect(page.getByRole("heading", { name: "Milestone" })).toBeVisible();
    await expectNoA11yViolations(page);
    await page.getByRole("link", { name: "Yeni kayıt" }).click();
    const milestoneForm = page.locator('form[action="/admin/resources/milestone/new"]');
    await expectNoA11yViolations(page);
    await milestoneForm.getByLabel("Title").fill("Blueprint generated milestone");
    await milestoneForm.getByLabel("Status").fill("planned");
    await milestoneForm.locator('button[type="submit"]').click();
    await expect(page).toHaveURL(/\/admin\/resources\/milestone(?:\?.*)?$/);
    const milestoneLink = page.getByRole("link", { name: "Blueprint generated milestone" });
    await expect(milestoneLink).toBeVisible();
    const milestonePath = await milestoneLink.getAttribute("href");
    expect(milestonePath).toMatch(/^\/admin\/resources\/milestone\//);
    const milestoneAPI = milestonePath!.replace("/admin/resources", "/api/v1");
    const milestoneRecord = await request.get(`${baseURL}${milestoneAPI}`);
    expect(milestoneRecord.status()).toBe(200);
    await milestoneLink.click();
    await page.locator('[name="title"]').fill("Blueprint generated milestone updated");
    await page.locator(`form[action="${milestonePath}"] button[type="submit"]`).click();
    await expect(page.getByRole("link", { name: "Blueprint generated milestone updated" })).toBeVisible();
    await page.getByRole("link", { name: "Blueprint generated milestone updated" }).click();
    page.once("dialog", (dialog) => dialog.accept());
    await page.locator(`form[action="${milestonePath}/delete"] button[type="submit"]`).click();
    await expect(page).toHaveURL(/\/admin\/resources\/milestone(?:\?.*)?$/);
    await expect(page.getByText("Blueprint generated milestone updated")).toHaveCount(0);
    const deletedMilestone = await request.get(`${baseURL}${milestoneAPI}`);
    expect(deletedMilestone.status()).toBe(404);

    const quillAsset = await request.get(`${baseURL}/vendor/quill/quill.js`);
    expect(quillAsset.status()).toBe(200);
    expect(quillAsset.headers()["content-type"]).toContain("javascript");
    const rejectedArticle = await request.post(`${baseURL}/api/v1/article`, {
      data: { title: "Invalid article", body: "" },
    });
    expect(rejectedArticle.status()).toBe(422);
    expect(await rejectedArticle.json()).toMatchObject({
      code: "validation_failed",
      fields: { body: "required" },
    });
    await page.goto(`${baseURL}/admin/resources/article`);
    await page.getByRole("link", { name: "Yeni kayıt" }).click();
    const articleForm = page.locator('form[action="/admin/resources/article/new"]');
    const articleEditor = articleForm.getByRole("textbox", { name: "Body" });
    await expect(articleEditor).toBeVisible();
    for (const family of ["hydrogo", "shadcn", "nova", "aurora"]) {
      for (const mode of ["light", "dark"]) {
        await page.evaluate(([theme, colorMode]) => localStorage.setItem("hydro-ui-theme-v2", `${theme}-${colorMode}`), [family, mode]);
        await page.reload();
        await expect(page.locator("html")).toHaveAttribute("data-ui-theme", family);
        await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
        await expect(articleEditor).toBeVisible();
        await expectNoA11yViolations(page);
      }
    }
    await page.evaluate(() => localStorage.setItem("hydro-ui-theme-v2", "hydrogo-light"));
    await page.reload();
    await expectNoA11yViolations(page);
    await page.setViewportSize({ width: 390, height: 844 });
    const richTextMobileOverflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
    expect(richTextMobileOverflow).toBeLessThanOrEqual(1);
    await expect(articleEditor).toBeVisible();
    await page.setViewportSize({ width: 1280, height: 720 });
    await articleForm.getByLabel("Title").fill("Generated Edition rich text article");
    await articleEditor.fill("Stored through the real Edition migration.");
    await articleEditor.press("Control+A");
    await articleEditor.press("Control+B");
    await expect(articleEditor.locator("strong")).toHaveText("Stored through the real Edition migration.");
    const createArticleButton = articleForm.locator('button[type="submit"]');
    await createArticleButton.focus();
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    const articleLink = page.getByRole("link", { name: "Generated Edition rich text article" });
    await expect(articleLink).toBeVisible();
    const articlePath = await articleLink.getAttribute("href");
    expect(articlePath).toMatch(/^\/admin\/resources\/article\//);
    const articleAPI = articlePath!.replace("/admin/resources", "/api/v1");
    const persistedArticle = await request.get(`${baseURL}${articleAPI}`);
    expect(persistedArticle.status()).toBe(200);
    const persistedBody = JSON.parse((await persistedArticle.json()).body as string);
    expect(persistedBody.ops).toContainEqual({ insert: "Stored through the real Edition migration.", attributes: { bold: true } });

    await articleLink.click();
    const articleEditForm = page.locator(`form[action="${articlePath}"]`);
    const restoredArticleEditor = articleEditForm.getByRole("textbox", { name: "Body" });
    await expect(restoredArticleEditor.locator("strong")).toHaveText("Stored through the real Edition migration.");
    await expect(page.locator("html")).toHaveAttribute("data-ui-theme", "hydrogo");
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
    const formColors = await page.evaluate(() => ({
      bodyLabel: getComputedStyle(document.querySelector("#body-label")!).color,
      title: getComputedStyle(document.querySelector<HTMLInputElement>("#title")!).color,
    }));
    expect(formColors).toEqual({ bodyLabel: "rgb(60, 66, 85)", title: "rgb(16, 20, 37)" });
    await expectNoA11yViolations(page);
    await restoredArticleEditor.fill("Updated in the generated Edition Admin.");
    await restoredArticleEditor.press("Control+A");
    await restoredArticleEditor.press("Control+I");
    await expect(restoredArticleEditor.locator("em")).toHaveText("Updated in the generated Edition Admin.");
    await articleEditForm.locator('button[type="submit"]').focus();
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    await page.getByRole("link", { name: "Generated Edition rich text article" }).click();
    await expect(page.locator(".richtext-editor .ql-editor em")).toHaveText("Updated in the generated Edition Admin.");
    const updatedArticle = await request.get(`${baseURL}${articleAPI}`);
    const updatedBody = JSON.parse((await updatedArticle.json()).body as string);
    const formattedUpdate = updatedBody.ops.find((operation: { insert?: string }) => operation.insert === "Updated in the generated Edition Admin.");
    expect(formattedUpdate.attributes).toMatchObject({ bold: true, italic: true });
    page.once("dialog", (dialog) => dialog.accept());
    await page.locator(`form[action="${articlePath}/delete"] button[type="submit"]`).click();
    await expect(page).toHaveURL(/\/admin\/resources\/article(?:\?.*)?$/);
    expect((await request.get(`${baseURL}${articleAPI}`)).status()).toBe(404);

    await page.goto(`${baseURL}/admin/resources/content_block`);
    await expect(page.getByRole("heading", { name: "Content blocks" })).toBeVisible();
    expect(await page.evaluate(() => typeof (window as any).htmx)).toBe("undefined");
    const blockFormLink = page.getByRole("link", { name: "Yeni kayıt" });
    await blockFormLink.click();
    const blockForm = page.locator('form[action="/admin/resources/content_block/new"]');
    await page.setViewportSize({ width: 390, height: 844 });
    await expectNoA11yViolations(page);
    expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(1);
    for (const family of ["hydrogo", "shadcn", "nova", "aurora"]) {
      for (const mode of ["light", "dark"]) {
        await page.evaluate(([theme, colorMode]) => localStorage.setItem("hydro-ui-theme-v2", `${theme}-${colorMode}`), [family, mode]);
        await page.reload();
        await expect(page.locator("html")).toHaveAttribute("data-ui-theme", family);
        await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
        await expect(page.getByLabel("Title")).toBeVisible();
        await expect(page.locator('[data-editorjs-mount="body"] .ce-paragraph[contenteditable="true"]')).toBeVisible();
        await expectNoA11yViolations(page);
      }
    }
    await page.evaluate(() => localStorage.setItem("hydro-ui-theme-v2", "hydrogo-light"));
    await page.reload();
    await expectNoA11yViolations(page);
    await blockForm.getByLabel("Title").fill("Generated Edition block content");
    const paragraph = page.locator('[data-editorjs-mount="body"] .ce-paragraph[contenteditable="true"]');
    await expect(paragraph).toBeVisible();
    const previewResponse = page.waitForResponse(async (response) => response.url().endsWith("/admin/resources/content_block/preview") && response.status() === 200 && (await response.text()).includes("A real Edition block paragraph."), { timeout: 5000 });
    await paragraph.fill("A real Edition block paragraph.");
    expect((await previewResponse).status()).toBe(200);
    await expect(page.locator("#body-preview")).toContainText("A real Edition block paragraph.");
    await blockForm.locator('button[type="submit"]').click();
    await expect(page).toHaveURL(/\/admin\/resources\/content_block(?:\?.*)?$/);
    const blockLink = page.getByRole("link", { name: "Generated Edition block content" });
    await expect(blockLink).toBeVisible();
    const blockPath = await blockLink.getAttribute("href");
    expect(blockPath).toMatch(/^\/admin\/resources\/content_block\//);
    const blockAPI = blockPath!.replace("/admin/resources", "/api/v1");
    const storedBlocks = JSON.parse((await (await request.get(`${baseURL}${blockAPI}`)).json()).body as string);
    expect(storedBlocks.blocks).toHaveLength(1);
    expect(storedBlocks.blocks[0].data.text).toBe("A real Edition block paragraph.");

    await blockLink.click();
    const editParagraph = page.locator('[data-editorjs-mount="body"] .ce-paragraph[contenteditable="true"]');
    await expect(editParagraph).toHaveText("A real Edition block paragraph.");
    for (const family of ["hydrogo", "shadcn", "nova", "aurora"]) {
      for (const mode of ["light", "dark"]) {
        await page.evaluate(([theme, colorMode]) => localStorage.setItem("hydro-ui-theme-v2", `${theme}-${colorMode}`), [family, mode]);
        await page.reload();
        await expect(page.locator("html")).toHaveAttribute("data-ui-theme", family);
        await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
        await expect(page.locator('[data-editorjs-mount="body"] .ce-paragraph[contenteditable="true"]')).toHaveText("A real Edition block paragraph.");
        await expectBlockEditorThemeInk(page);
        await expectNoA11yViolations(page, `generated block edit (${family}/${mode})`);
      }
    }
    await page.evaluate(() => localStorage.setItem("hydro-ui-theme-v2", "hydrogo-light"));
    await page.reload();
    await expect(page.locator("html")).toHaveAttribute("data-ui-theme", "hydrogo");
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
    await expectBlockEditorThemeInk(page);
    await expectNoA11yViolations(page);
    const editPreview = page.waitForResponse(async (response) => response.url().endsWith("/admin/resources/content_block/preview") && response.status() === 200 && (await response.text()).includes("Updated Edition block paragraph."), { timeout: 5000 });
    await editParagraph.fill("Updated Edition block paragraph.");
    expect((await editPreview).status()).toBe(200);
    await page.locator(`form[action="${blockPath}"] button[type="submit"]`).click();
    await expect(page).toHaveURL(/\/admin\/resources\/content_block(?:\?.*)?$/);
    const updatedBlocks = JSON.parse((await (await request.get(`${baseURL}${blockAPI}`)).json()).body as string);
    expect(updatedBlocks.blocks[0].data.text).toBe("Updated Edition block paragraph.");
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.getByRole("link", { name: "Generated Edition block content" }).click();
    page.once("dialog", (dialog) => dialog.accept());
    await page.locator(`form[action="${blockPath}/delete"] button[type="submit"]`).click();
    await expect(page).toHaveURL(/\/admin\/resources\/content_block(?:\?.*)?$/);
    expect((await request.get(`${baseURL}${blockAPI}`)).status()).toBe(404);

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`${baseURL}/admin/resources/milestone/new`);
    await expect(page.getByLabel("Title")).toBeVisible();
    expect(await page.evaluate(() => typeof (window as any).htmx)).toBe("undefined");
    for (const family of ["hydrogo", "shadcn", "nova", "aurora"]) {
      for (const mode of ["light", "dark"]) {
        await page.evaluate(([theme, colorMode]) => localStorage.setItem("hydro-ui-theme-v2", `${theme}-${colorMode}`), [family, mode]);
        await page.reload();
        await expect(page.locator("html")).toHaveAttribute("data-ui-theme", family);
        await expect(page.locator("html")).toHaveAttribute("data-theme", mode);
        await expectNoA11yViolations(page);
      }
    }
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

async function expectNoA11yViolations(page: import("@playwright/test").Page, context = "Admin page"): Promise<void> {
  const result = await page.evaluate(async () => (window as any).axe.run(document, {
    runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] },
  }));
  const violations = result.violations.map((violation: any) => ({
    id: violation.id,
    nodes: violation.nodes.map((node: any) => ({
      target: node.target,
      failureSummary: node.failureSummary,
      html: node.html,
      checks: node.any?.map((check: any) => check.data),
    })),
  }));
  expect(violations, `${context} accessibility violations`).toEqual([]);
}

async function expectBlockEditorThemeInk(page: import("@playwright/test").Page): Promise<void> {
  await page.waitForFunction(() => {
    const editor = document.querySelector<HTMLElement>('.block-editor .ce-paragraph[contenteditable="true"]');
    if (!editor) return false;
    const probe = document.createElement("span");
    probe.style.color = "var(--ink)";
    document.body.append(probe);
    const tokenColor = getComputedStyle(probe).color;
    probe.remove();
    if (getComputedStyle(editor).color !== tokenColor) return false;
    for (let node: HTMLElement | null = editor; node; node = node.parentElement) {
      if (Number.parseFloat(getComputedStyle(node).opacity) < 0.999) return false;
      if (node.classList.contains("block-editor")) break;
    }
    return true;
  }, undefined, { timeout: 5000 });
}
