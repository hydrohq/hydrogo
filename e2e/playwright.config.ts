import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  outputDir: "./artifacts",
  timeout: 30_000,
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? "line" : "list",
  use: {
    baseURL: process.env.HYDRO_E2E_BASE_URL ?? "http://127.0.0.1:8090",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    ...process.env.HYDRO_E2E_STORAGE_STATE ? { storageState: process.env.HYDRO_E2E_STORAGE_STATE } : {},
  },
  projects: [
    { name: "chromium", testIgnore: /visual\.spec\.ts|locale\.spec\.ts/, use: { ...devices["Desktop Chrome"] } },
  ],
});
