import { defineConfig, devices } from "@playwright/test";

// Browser journeys that start their own server (see journeys/global-setup.ts):
// no running instance, credentials or seeded data are needed.
//
//   npm run journeys
//
// One worker and no retries on purpose: the journeys share one server and one
// sign-in rate limit (5 a minute), and a retry would hide a flaky journey.
export default defineConfig({
  testDir: "./journeys",
  globalSetup: "./journeys/global-setup.ts",
  outputDir: "./test-results-journeys",
  timeout: 60_000,
  workers: 1,
  fullyParallel: false,
  retries: 0,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? "line" : "list",
  use: {
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    ...devices["Desktop Chrome"],
    launchOptions: {
      args: ["--host-resolver-rules=MAP *.example.test 127.0.0.1"],
    },
  },
});
