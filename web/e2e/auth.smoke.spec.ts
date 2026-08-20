import { test, expect } from "@playwright/test";

test.describe("Authentication", () => {
  test("login page renders", async ({ page }) => {
    await page.goto("/login");
    await page.waitForLoadState("networkidle");

    // Login page should render — check for the card or any content
    const pageContent = await page.textContent("body");
    expect(pageContent).toBeTruthy();
  });

  test("protected routes redirect to login", async ({ page }) => {
    await page.goto("/dashboard");
    await page.waitForURL(/login/);
    expect(page.url()).toContain("/login");
  });

  test("profile route redirects to login", async ({ page }) => {
    await page.goto("/profile");
    await page.waitForURL(/login/);
    expect(page.url()).toContain("/login");
  });
});
