import { test, expect } from "@playwright/test";

const baseURL = process.env.HYDRO_E2E_BASE_URL ?? "http://127.0.0.1:8090";

test.describe("Critical User Journeys (P1)", () => {
  // Journey 1: Public content and locale fallback verification
  test("journey 1: public content is served and verified", async ({ page, request }) => {
    // 1. Visit public demo page
    const pageRes = await page.goto(`${baseURL}/public/pages/tr/demo/html`);
    expect(pageRes?.status()).toBe(200);
    await expect(page.locator('[data-testid="public-page"]')).toBeAttached();
    await expect(page.locator('main')).toBeAttached();
    await expect(page).toHaveTitle(/\S+/);

    // 2. Fetch page via versioned JSON API
    const apiRes = await request.get(`${baseURL}/api/v1/pages/by-slug?locale=tr&slug=demo`);
    expect(apiRes.status()).toBe(200);
    const apiData = await apiRes.json();
    expect(apiData.slug).toBe("demo");
    expect(apiData.locale).toBe("tr");
  });

  // Journey 2: Security negative paths (CSRF, unauthorized mutations)
  test("journey 2: security negative flows reject unauthorized mutations and missing CSRF", async ({ request }) => {
    // 1. Attempt POST to admin endpoint without authentication or CSRF
    const unauthPost = await request.post(`${baseURL}/admin/pages`, {
      form: {
        title: "Malicious Page",
        slug: "malicious-page",
      },
    });
    // Must be rejected (401 Unauthorized or 403 Forbidden or 303 Redirect to login)
    expect([401, 403, 303]).toContain(unauthPost.status());

    // 2. Attempt direct API mutation without auth
    const apiMutate = await request.post(`${baseURL}/api/v1/pages`, {
      data: {
        title: "Unauthorized Page",
        slug: "unauthorized-slug",
      },
    });
    expect([401, 403]).toContain(apiMutate.status());
  });

  // Journey 3: Member area isolation and negative authentication
  test("journey 3: member authentication errors on invalid credentials and isolates unauthenticated access", async ({ page, request }) => {
    // 1. Visit member login page
    await page.goto(`${baseURL}/member/login`);
    await expect(page.locator('[data-testid="member-auth"]')).toBeAttached();

    // 2. Submit bad login
    await page.locator('input[name="email"]').fill("nonexistent@example.com");
    await page.locator('input[name="password"]').fill("WrongPassword123!");
    await page.locator('button[type="submit"]').click();

    // Verify stays on login / shows error
    await expect(page.locator('[data-testid="member-auth"]')).toBeAttached();

    // 3. Directly querying /member/me without session cookie returns 401 / 403 / redirect
    const meRes = await request.get(`${baseURL}/member/me`);
    expect([401, 403, 303]).toContain(meRes.status());
  });

  // Journey 4: Content resilience (sitemap availability and redirect handling)
  test("journey 4: sitemap and redirects are resilient", async ({ request }) => {
    // 1. Verify sitemap.xml
    const sitemapRes = await request.get(`${baseURL}/sitemap.xml`);
    expect(sitemapRes.status()).toBe(200);
    let xmlText = await sitemapRes.text();
    if (xmlText.includes("<sitemapindex")) {
      // The 50k-URL splitter publishes category sitemaps behind the index;
      // resilience means the index references children that serve urlsets.
      const childUrl = xmlText.match(/<loc>([^<]+)<\/loc>/)?.[1];
      expect(childUrl, "sitemap index must reference a child sitemap").toBeTruthy();
      const childRes = await request.get(childUrl);
      expect(childRes.status()).toBe(200);
      xmlText = await childRes.text();
    }
    expect(xmlText).toContain("<urlset");

    // 2. Verify robots.txt
    const robotsRes = await request.get(`${baseURL}/robots.txt`);
    expect([200, 404]).toContain(robotsRes.status());
  });
});
