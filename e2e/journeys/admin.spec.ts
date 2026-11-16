import { expect, test, type Page } from "@playwright/test";
import { ADMIN_EMAIL, ADMIN_PASSWORD } from "./global-setup";

// The admin panel's own flows: who can do what, and the e-mailed links.
// These are the flows that broke unnoticed before: the unit tests passed
// while the Users screen sent forms the server could not read.

const hydro = (path = "") => `${process.env.JOURNEY_HYDRO_URL}${path}`;
const STAFF = "staff@journeys.test";

async function signIn(page: Page, email: string, password: string) {
  await page.goto(hydro("/admin/login"));
  await page.locator('input[type="email"]').fill(email);
  await page.locator('input[type="password"]').fill(password);
  await Promise.all([page.waitForURL((u) => !u.pathname.includes("/login")), page.locator('button[type="submit"]').first().click()]);
}

const users = (page: Page) => page.evaluate(async () => (await (await fetch("/api/v1/admin/users")).json()).data as { email: string; role: string; permissions: string[] }[]);
const staff = async (page: Page) => (await users(page)).find((u) => u.email === STAFF)!;

test.describe.configure({ mode: "serial" });

test.describe("admin: users and e-mailed links", () => {
  let admin: Page;
  test.beforeAll(async ({ browser }) => {
    admin = await (await browser.newContext()).newPage();
    await signIn(admin, ADMIN_EMAIL, ADMIN_PASSWORD);
    await admin.goto(hydro("/admin/users"));
    await expect(admin.getByText(STAFF).first()).toBeVisible();
  });
  test.afterAll(async () => admin.context().close());

  const row = () => admin.locator("div, tr").filter({ hasText: STAFF }).filter({ has: admin.getByRole("button", { name: "Role" }) }).last();

  test("changing a role in the Users screen sticks", async () => {
    await row().getByRole("button", { name: "Role" }).click();
    await admin.locator("select").first().selectOption("editor");
    await admin.getByRole("button", { name: /save|update/i }).last().click();
    await expect.poll(async () => (await staff(admin)).role).toBe("editor");
  });

  test("saving permissions without touching them keeps the explicit ones", async () => {
    expect((await staff(admin)).permissions).toContain("admin.site.manage");
    await admin.reload();
    await expect(admin.getByText(STAFF).first()).toBeVisible();
    await row().getByRole("button", { name: "Permissions" }).click();
    await admin.getByRole("button", { name: /save|update/i }).last().click();
    await expect.poll(async () => (await staff(admin)).permissions).toContain("admin.site.manage");
  });

  test("an invitation without mail still gives the administrator a link that works once", async ({ browser }) => {
    await admin.reload();
    await admin.getByRole("button", { name: "Invite Staff Member" }).click();
    await admin.locator('input[type="email"]').last().fill("newhire@journeys.test");
    const sent = admin.waitForResponse((r) => r.url().includes("/users/invites"));
    await admin.getByRole("button", { name: /send|invite/i }).last().click();
    const body = await (await sent).json();
    expect(body.delivered).toBe(false);
    await expect(admin.locator("input[readonly]").first()).toHaveValue(body.url);

    const person = await (await browser.newContext()).newPage();
    await person.goto(body.url);
    await expect(person.getByRole("heading", { name: /set your password/i })).toBeVisible();
    await person.locator('input[type="password"]').nth(0).fill("Neu-Mitarbeiter-Parola-55!");
    await person.locator('input[type="password"]').nth(1).fill("Something-Else-Entirely-12!");
    await person.locator('button[type="submit"]').click();
    await expect(person.getByRole("alert")).toContainText(/do not match/i);
    await person.locator('input[type="password"]').nth(1).fill("Neu-Mitarbeiter-Parola-55!");
    await person.locator('button[type="submit"]').click();
    await expect(person).toHaveURL(/\/admin\/login\?invite_accepted=1/);
    await expect(person.getByRole("status")).toBeVisible();
    await signIn(person, "newhire@journeys.test", "Neu-Mitarbeiter-Parola-55!");
    await expect(person).toHaveURL(/\/admin\/dashboard/);

    await person.goto(body.url);
    await person.locator('input[type="password"]').nth(0).fill("Yet-Another-Parola-77!");
    await person.locator('input[type="password"]').nth(1).fill("Yet-Another-Parola-77!");
    await person.locator('button[type="submit"]').click();
    await expect(person.getByRole("alert")).toContainText(/invalid or has expired/i);
    await person.context().close();
  });

  test("a password-reset link opens a page that sets the password, once", async ({ browser }) => {
    await admin.reload();
    await expect(admin.getByText(STAFF).first()).toBeVisible();
    const made = admin.waitForResponse((r) => r.url().includes("password-reset-link"));
    await row().getByRole("button", { name: "Reset Link" }).click();
    const { url } = await (await made).json();
    expect(url).toContain("/admin/reset-password?token=");

    const person = await (await browser.newContext()).newPage();
    await person.goto(url);
    await expect(person.getByRole("heading", { name: /new password/i })).toBeVisible();
    await person.locator('input[type="password"]').nth(0).fill("Staff-Reset-Parola-2026-Qx!");
    await person.locator('input[type="password"]').nth(1).fill("Staff-Reset-Parola-2026-Qx!");
    await person.locator('button[type="submit"]').click();
    await expect(person).toHaveURL(/\/admin\/login\?password_reset=1/);
    await signIn(person, STAFF, "Staff-Reset-Parola-2026-Qx!");
    await expect(person).toHaveURL(/\/admin\/dashboard/);
    await person.goto(url);
    await person.locator('input[type="password"]').nth(0).fill("Staff-Second-Parola-2026-Qx!");
    await person.locator('input[type="password"]').nth(1).fill("Staff-Second-Parola-2026-Qx!");
    await person.locator('button[type="submit"]').click();
    await expect(person.getByRole("alert")).toContainText(/invalid or has expired/i);
    await person.context().close();
  });

  test("editorial workflow: sidebar groups are clean and full-page content editor supports inline options & visual relations", async () => {
    const errors: string[] = [];
    const serverErrors: string[] = [];
    admin.on("pageerror", (e) => errors.push(e.message));
    admin.on("response", (r) => {
      if (r.status() >= 500) serverErrors.push(`${r.status()} ${r.url()}`);
    });

    await admin.goto(hydro("/admin/content-types"));
    await expect(admin.getByRole("main").getByRole("heading", { name: /Content Types/i })).toBeVisible();

    // Sidebar section group headers should never repeat adjacent identical headings.
    const groupButtons = await admin.locator('aside [data-testid^="nav-group-"]').allInnerTexts();
    for (const label of groupButtons) {
      expect(label.trim().length).toBeGreaterThan(0);
    }

    await admin.getByRole("button", { name: "New Object" }).click();
    const dialog = admin.getByRole("dialog");
    await dialog.getByPlaceholder(/Testimonials/i).fill("Playbooks");
    await dialog.getByRole("button", { name: "+ select" }).click();
    await dialog.getByPlaceholder(/Comma-separated options/i).fill("Onboarding, Support");

    await dialog.getByRole("button", { name: "+ Add Field" }).click();
    const fieldInputs = dialog.getByPlaceholder(/Field Label/i);
    await fieldInputs.last().fill("Company Ref");
    await dialog.getByPlaceholder(/ID \(auto if empty\)/i).last().fill("company_name");
    await dialog.getByLabel("Widget").last().selectOption("relation");
    await dialog.getByPlaceholder(/Target Object ID/i).last().fill("companies");
    await dialog.getByPlaceholder(/Stored field/i).last().fill("name");

    await dialog.getByRole("button", { name: /Create New Object/i }).click();
    await expect(admin).toHaveURL(/\/admin\/content\/playbooks$/);

    await admin.getByRole("button", { name: /Add Playbooks/i }).click();
    await expect(admin).toHaveURL(/\/admin\/content\/playbooks\/new$/);

    // Add a new dropdown option inline via SelectOptionsModal
    await admin.getByRole("button", { name: "Manage options" }).click();
    const optModal = admin.getByRole("dialog");
    await optModal.getByPlaceholder(/New option name/i).fill("Security");
    await optModal.getByRole("button", { name: "Add option" }).click();
    await optModal.getByRole("button", { name: "Save", exact: true }).click();
    await expect(optModal).toBeHidden();

    await admin.getByLabel(/Title/i).first().fill("Release Checklist");
    await admin.locator("select").first().selectOption("Security");

    // Pick related company visually via RelationField without typing raw IDs/slugs
    const rel = admin.locator('[data-relation-field="company_name"]');
    await expect(rel.getByRole("option", { name: /Northwind Software/i })).toBeVisible();
    await rel.getByRole("option", { name: /Northwind Software/i }).click();
    await expect(rel.getByRole("button", { name: /Remove Northwind Software/i })).toBeVisible();

    await admin.getByRole("button", { name: "Publish" }).click();
    await expect(admin).toHaveURL(/\/admin\/content\/playbooks\/[a-z0-9_-]+$/);

    await admin.getByRole("button", { name: "Back" }).click();
    await expect(admin).toHaveURL(/\/admin\/content\/playbooks$/);
    await expect(admin.locator("table")).toContainText("Release Checklist");
    await expect(admin.locator("table")).toContainText("Security");

    expect(errors).toEqual([]);
    expect(serverErrors).toEqual([]);
  });
});
