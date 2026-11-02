import { expect, test, type Page } from "@playwright/test";
import { ADMIN_PASSWORD } from "./global-setup";

const hydroPort = () => process.env.JOURNEY_HYDRO_PORT!;
const alphaUrl = (path = "") => `http://alpha.example.test:${hydroPort()}${path}`;
const betaUrl = (path = "") => `http://beta.example.test:${hydroPort()}${path}`;
const directUrl = (path = "") => `${process.env.JOURNEY_HYDRO_URL}${path}`;

async function signIn(page: Page, url: string, email: string) {
  await page.goto(`${url}/admin/login`);
  await page.locator('input[type="email"]').fill(email);
  await page.locator('input[type="password"]').fill(ADMIN_PASSWORD);
  await Promise.all([
    page.waitForURL((u) => !u.pathname.includes("/login")),
    page.locator('button[type="submit"]').first().click(),
  ]);
}

test.describe("multi-tenant: data isolation and cross-tenant tampering", () => {
  test("subdomain authentication separation and cross-tenant session rejection", async ({ browser }) => {
    // 1. Log into Tenant Alpha
    const alphaContext = await browser.newContext();
    const alphaPage = await alphaContext.newPage();
    await signIn(alphaPage, alphaUrl(), "admin@alpha.test");
    await expect(alphaPage).toHaveURL(/\/admin\/(dashboard|content)/);

    // 2. Open Tenant Beta in separate context; it must NOT be authenticated
    const betaContext = await browser.newContext();
    const betaPage = await betaContext.newPage();
    await betaPage.goto(betaUrl("/admin/dashboard"));
    await expect(betaPage).toHaveURL(/\/admin\/login/);

    // 3. Log into Tenant Beta
    await signIn(betaPage, betaUrl(), "admin@beta.test");
    await expect(betaPage).toHaveURL(/\/admin\/(dashboard|content)/);

    // 4. Session Tampering: Forge Tenant Alpha session cookie into Tenant Beta request
    const alphaCookies = await alphaContext.cookies(alphaUrl());
    const alphaSession = alphaCookies.find((c) => c.name === "hydro_session")?.value;
    expect(alphaSession).toBeDefined();

    // Query Tenant Beta /api/v1/admin/me using Alpha's session cookie via direct request
    const forgedResp = await betaPage.request.get(directUrl("/api/v1/admin/me"), {
      headers: {
        Host: "beta.example.test",
        Cookie: `hydro_session=${alphaSession}`,
      },
    });
    // Must be rejected as 401 Unauthorized because session is scoped to tenant-alpha
    expect(forgedResp.status()).toBe(401);

    await alphaContext.close();
    await betaContext.close();
  });

  test("collection records and direct ID lookups are strictly isolated across tenants", async ({ browser }) => {
    const alphaContext = await browser.newContext();
    const alphaPage = await alphaContext.newPage();
    await signIn(alphaPage, alphaUrl(), "admin@alpha.test");

    const betaContext = await browser.newContext();
    const betaPage = await betaContext.newPage();
    await signIn(betaPage, betaUrl(), "admin@beta.test");

    // Alpha creates a company via in-browser fetch with session cookie & CSRF header
    const alphaRecord = await alphaPage.evaluate(async () => {
      const csrf = document.cookie.split("; ").find((c) => c.startsWith("hydro_csrf="))?.split("=")[1] || "";
      const res = await fetch("/api/v1/admin/content/companies", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
        body: JSON.stringify({ name: "Alpha Secret Corp" }),
      });
      return { status: res.status, body: await res.json() };
    });
    expect(alphaRecord.status).toBe(201);
    const alphaId = alphaRecord.body.data.id;

    // Beta creates a company via in-browser fetch with session cookie & CSRF header
    const betaRecord = await betaPage.evaluate(async () => {
      const csrf = document.cookie.split("; ").find((c) => c.startsWith("hydro_csrf="))?.split("=")[1] || "";
      const res = await fetch("/api/v1/admin/content/companies", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
        body: JSON.stringify({ name: "Beta Independent LLC" }),
      });
      return { status: res.status, body: await res.json() };
    });
    expect(betaRecord.status).toBe(201);
    const betaId = betaRecord.body.data.id;

    // In Alpha browser: only Alpha's record is listed
    const alphaCompanies = await alphaPage.evaluate(async () => {
      const res = await fetch("/api/v1/admin/content/companies");
      return (await res.json()).data as { id: string; name: string }[];
    });
    expect(alphaCompanies.some((c) => c.name === "Alpha Secret Corp")).toBe(true);
    expect(alphaCompanies.some((c) => c.name === "Beta Independent LLC")).toBe(false);

    // In Beta browser: only Beta's record is listed
    const betaCompanies = await betaPage.evaluate(async () => {
      const res = await fetch("/api/v1/admin/content/companies");
      return (await res.json()).data as { id: string; name: string }[];
    });
    expect(betaCompanies.some((c) => c.name === "Beta Independent LLC")).toBe(true);
    expect(betaCompanies.some((c) => c.name === "Alpha Secret Corp")).toBe(false);

    // Cross-tenant direct record lookup tampering:
    // Tenant Beta tries to read Tenant Alpha's record by ID -> 404 Not Found
    const crossLookupStatus = await betaPage.evaluate(async (id) => {
      const res = await fetch(`/api/v1/admin/content/companies/${id}`);
      return res.status;
    }, alphaId);
    expect(crossLookupStatus).toBe(404);

    // Tenant Alpha tries to read Tenant Beta's record by ID -> 404 Not Found
    const reverseCrossLookupStatus = await alphaPage.evaluate(async (id) => {
      const res = await fetch(`/api/v1/admin/content/companies/${id}`);
      return res.status;
    }, betaId);
    expect(reverseCrossLookupStatus).toBe(404);

    // Public API content isolation:
    const pubAlphaData = await alphaPage.evaluate(async () => {
      const res = await fetch("/api/v1/content/companies");
      return res.ok ? ((await res.json()).data as { id: string }[]) : [];
    });
    expect(pubAlphaData.some((c) => c.id === betaId)).toBe(false);

    const pubBetaData = await betaPage.evaluate(async () => {
      const res = await fetch("/api/v1/content/companies");
      return res.ok ? ((await res.json()).data as { id: string }[]) : [];
    });
    expect(pubBetaData.some((c) => c.id === alphaId)).toBe(false);

    await alphaContext.close();
    await betaContext.close();
  });

  test("invitation links created on Tenant Alpha cannot be redeemed on Tenant Beta", async ({ browser }) => {
    const alphaContext = await browser.newContext();
    const alphaPage = await alphaContext.newPage();
    await signIn(alphaPage, alphaUrl(), "admin@alpha.test");

    // Send invitation on Tenant Alpha via browser
    const inviteResult = await alphaPage.evaluate(async () => {
      const csrf = document.cookie.split("; ").find((c) => c.startsWith("hydro_csrf="))?.split("=")[1] || "";
      const res = await fetch("/api/v1/admin/users/invites", {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded", "X-CSRF-Token": csrf },
        body: new URLSearchParams({ email: "isolated_staff@journeys.test", role: "viewer" }).toString(),
      });
      return { status: res.status, body: await res.json() };
    });
    expect(inviteResult.status).toBe(202);
    const originalUrl = inviteResult.body.url as string;
    expect(originalUrl).toContain("/admin/accept-invite?token=");

    // Extract token
    const token = new URL(originalUrl).searchParams.get("token")!;
    expect(token).toBeTruthy();

    // Tampering attempt: Navigate to Tenant Beta with the Alpha invitation token
    const betaContext = await browser.newContext();
    const betaPage = await betaContext.newPage();
    await betaPage.goto(betaUrl(`/admin/accept-invite?token=${token}`));

    // Fill in password on Tenant Beta
    await betaPage.locator('input[type="password"]').nth(0).fill("Tampered-Cross-Tenant-Pass-99!");
    await betaPage.locator('input[type="password"]').nth(1).fill("Tampered-Cross-Tenant-Pass-99!");
    await betaPage.locator('button[type="submit"]').click();

    // Token must be rejected because it belongs to tenant-alpha, not tenant-beta
    await expect(betaPage.getByRole("alert")).toContainText(/invalid or has expired/i);

    await alphaContext.close();
    await betaContext.close();
  });

  test("ai site generation generates layout and products, rejects unauthorized access", async ({ browser }) => {
    const alphaContext = await browser.newContext();
    const alphaPage = await alphaContext.newPage();
    await signIn(alphaPage, alphaUrl(), "admin@alpha.test");

    // Call /api/v1/admin/ai/generate with a boutique prompt
    const genResult = await alphaPage.evaluate(async () => {
      const csrf = document.cookie.split("; ").find((c) => c.startsWith("hydro_csrf="))?.split("=")[1] || "";
      const res = await fetch("/api/v1/admin/ai/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
        body: JSON.stringify({
          prompt: "Koyu altın ve siyah el yapımı takılar",
          name: "Noir Takı Atölyesi",
          phone: "+905551234567",
        }),
      });
      return { status: res.status, body: await res.json() };
    });

    expect(genResult.status).toBe(200);
    expect(genResult.body.data.products_generated).toBeGreaterThan(0);
    expect(genResult.body.data.sections_generated).toBeGreaterThan(0);

    // Verify unauthenticated user cannot invoke AI generate
    const anonContext = await browser.newContext();
    const anonPage = await anonContext.newPage();
    const anonResp = await anonPage.request.post(alphaUrl("/api/v1/admin/ai/generate"), {
      data: { prompt: "Attack without auth" },
    });
    expect(anonResp.status()).toBe(401);

    await alphaContext.close();
    await anonContext.close();
  });
});
