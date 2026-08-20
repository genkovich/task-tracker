import { test, expect } from "@playwright/test";

test.describe("Navigation", () => {
  test("home shows the bare Google login button", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("button", { name: /sign in with google/i })).toBeVisible();
  });

  test("404 page for unknown routes", async ({ page }) => {
    await page.goto("/nonexistent-route-xyz");
    await expect(page.getByRole("heading", { name: /page not found/i })).toBeVisible();
    await expect(page.getByRole("link", { name: /dashboard/i })).toBeVisible();
  });

  test("home page has a title", async ({ page }) => {
    await page.goto("/");
    const title = await page.title();
    expect(title).toContain("Task Tracker");
  });

  test("login page has no console errors (regression: React #418)", async ({ page }) => {
    const errors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") errors.push(msg.text());
    });

    await page.goto("/login");
    await page.waitForLoadState("networkidle");

    expect(errors, errors.join("\n")).toEqual([]);
  });

  test("404 page has no console errors (regression: catch-all route)", async ({ page }) => {
    const errors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") errors.push(msg.text());
    });

    await page.goto("/nonexistent-route-xyz");
    await page.waitForLoadState("networkidle");

    expect(errors, errors.join("\n")).toEqual([]);
  });

  test("theme-init script ships in head and tags <html> before hydration", async ({ page }) => {
    await page.goto("/");
    const themeInit = page.locator('script[src="/theme-init.js"]');
    await expect(themeInit).toHaveCount(1);
    const colorScheme = await page.evaluate(() => document.documentElement.style.colorScheme);
    expect(["light", "dark"]).toContain(colorScheme);
  });
});
