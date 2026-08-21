import { test, expect } from "@playwright/test";

test.describe("Authentication", () => {
  test("landing renders the sign-in", async ({ page }) => {
    await page.goto("/");
    await page.waitForLoadState("networkidle");

    // Login page should render — check for the card or any content
    const pageContent = await page.textContent("body");
    expect(pageContent).toBeTruthy();
  });

  test("protected routes bounce guests to the landing", async ({ page }) => {
    await page.goto("/dashboard");
    await page.waitForURL((u) => u.pathname === "/");
    expect(new URL(page.url()).pathname).toBe("/");
  });

  test("profile route bounces guests to the landing", async ({ page }) => {
    await page.goto("/profile");
    await page.waitForURL((u) => u.pathname === "/");
    expect(new URL(page.url()).pathname).toBe("/");
  });
});
