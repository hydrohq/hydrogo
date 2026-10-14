// Takes desktop and mobile screenshots of HydroGo pages so a change can be
// looked at, not just tested. Logs in first when admin credentials are set.
//
//   HYDRO_ADMIN_EMAIL=... HYDRO_ADMIN_PASSWORD=... \
//     node e2e/screenshot.mjs --base http://127.0.0.1:8090 /admin /admin/resources/faq
//
// Screenshots go to .tmp/screenshots/<path>-<viewport>.png (override with --out).
// In Git Bash on Windows, prefix with MSYS_NO_PATHCONV=1 so /paths stay URLs.
import { chromium } from "@playwright/test";
import { mkdirSync } from "node:fs";
import { join } from "node:path";

const args = process.argv.slice(2);
let base = "http://127.0.0.1:8090";
let out = ".tmp/screenshots";
const paths = [];
for (let i = 0; i < args.length; i++) {
  if (args[i] === "--base") base = args[++i];
  else if (args[i] === "--out") out = args[++i];
  else paths.push(args[i]);
}
if (paths.length === 0) {
  console.error("usage: node e2e/screenshot.mjs [--base URL] [--out DIR] /path [/path ...]");
  process.exit(2);
}
mkdirSync(out, { recursive: true });

const viewports = { desktop: { width: 1440, height: 900 }, mobile: { width: 390, height: 844 } };
const browser = await chromium.launch();
const context = await browser.newContext({ baseURL: base });

const email = process.env.HYDRO_ADMIN_EMAIL;
const password = process.env.HYDRO_ADMIN_PASSWORD;
if (email && password) {
  const page = await context.newPage();
  await page.goto("/login?return_to=%2Fadmin");
  await page.locator('input[name="email"]').fill(email);
  await page.locator('input[name="password"]').fill(password);
  await Promise.all([page.waitForURL(/\/admin/), page.locator('button[type="submit"]').first().click()]);
  // Skip the first-run onboarding tour, like a returning admin.
  await page.evaluate(() => {
    localStorage.setItem("hydro-onboarding-dismissed", "1");
    localStorage.setItem(`hydro-onboarding-dismissed:${document.body?.dataset?.studioRole || "admin"}`, "1");
  });
  await page.close();
}

const failures = [];
for (const path of paths) {
  for (const [name, viewport] of Object.entries(viewports)) {
    const page = await context.newPage();
    await page.setViewportSize(viewport);
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    const response = await page.goto(path, { waitUntil: "networkidle" });
    const file = join(out, `${path.replace(/^\/+/, "").replace(/[^a-zA-Z0-9]+/g, "-") || "home"}-${name}.png`);
    await page.screenshot({ path: file, fullPage: true });
    const status = response ? response.status() : 0;
    console.log(`${status} ${path} (${name}) -> ${file}${errors.length ? `  JS errors: ${errors.join(" | ")}` : ""}`);
    if (status >= 400 || errors.length) failures.push(path);
    await page.close();
  }
}
await browser.close();
process.exit(failures.length ? 1 : 0);
