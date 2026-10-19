import { test, expect } from "@playwright/test";
import { createHydroGoClient } from "../../packages/client/dist/hydrogo.js";

test("headless starter reads published content through the versioned API", async ({ request }) => {
  const client = createHydroGoClient({
    origin: process.env.HYDRO_E2E_BASE_URL ?? "http://127.0.0.1:8090",
    fetch: async (input, init) => {
      const response = await request.fetch(String(input), {
        method: init?.method,
        headers: init?.headers as Record<string, string> | undefined,
        data: init?.body,
      });
      return new Response(await response.body(), {
        status: response.status(),
        headers: response.headers(),
      });
    },
  });

  const page = await client.getPage("tr", "demo");
  expect(page.status).toBe("published");
  expect(page.title).toBeTruthy();
  expect(page.locale).toBe("tr");
});
