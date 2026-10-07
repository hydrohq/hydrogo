import { chromium } from "@playwright/test";

const baseURL = process.env.HYDRO_E2E_BASE_URL ?? "http://127.0.0.1:8090";
const email = process.env.HYDRO_E2E_ADMIN_EMAIL;
const password = process.env.HYDRO_E2E_ADMIN_PASSWORD;
const output = process.env.HYDRO_E2E_STORAGE_STATE ?? "auth-state.json";
if (!email || !password) throw new Error("HYDRO_E2E_ADMIN_EMAIL and HYDRO_E2E_ADMIN_PASSWORD are required");

const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({ baseURL });
const page = await context.newPage();
await page.goto("/login?return_to=%2Fadmin");
await page.locator('input[name="email"]').fill(email);
await page.locator('input[name="password"]').fill(password);
await page.locator('button[type="submit"]').click();
await page.waitForURL(/\/admin(?:$|\?)/);
// Persist the onboarding-tour dismissal so admin journeys interact with the
// editor instead of the first-run dialog — same as a returning admin who
// clicked "Bu rehberi bir daha gösterme" (plain key + role-scoped key).
await page.evaluate(() => {
  localStorage.setItem("hydro-onboarding-dismissed", "1");
  const role = document.body?.dataset?.studioRole || "admin";
  localStorage.setItem(`hydro-onboarding-dismissed:${role}`, "1");
});
await context.storageState({ path: output });
await browser.close();
console.log(`wrote ${output}`);
