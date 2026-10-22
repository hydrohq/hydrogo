import { chromium } from "@playwright/test";

const baseURL = process.env.HYDRO_E2E_BASE_URL ?? "http://127.0.0.1:8090";
const email = process.env.HYDRO_E2E_ADMIN_EMAIL;
const password = process.env.HYDRO_E2E_ADMIN_PASSWORD;
const output = process.env.HYDRO_E2E_STORAGE_STATE ?? "auth-state.json";
if (!email || !password) throw new Error("HYDRO_E2E_ADMIN_EMAIL and HYDRO_E2E_ADMIN_PASSWORD are required");

const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({ baseURL });
const page = await context.newPage();
await page.goto("/admin/#/login");
await page.locator('input[type="email"], input[name="email"]').fill(email);
await page.locator('input[type="password"], input[name="password"]').fill(password);
// The sign-in page itself lives under /admin, so waiting for an /admin URL
// returns at once and closing the browser cancels the request. Wait for the
// login response and for the SPA to leave the sign-in page.
const [login] = await Promise.all([
  page.waitForResponse((response) => response.url().includes("/auth/login") && response.request().method() === "POST"),
  page.locator('button[type="submit"]').click(),
]);
if (!login.ok()) throw new Error(`sign-in failed with HTTP ${login.status()}`);
await page.waitForURL((url) => !url.href.includes("login"));
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
