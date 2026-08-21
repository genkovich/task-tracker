/**
 * Board helpers for the drag-and-drop E2E specs.
 *
 * Needs the live stack (API on :8080 + seeded columns) and JWT_SECRET in the
 * environment — the specs authenticate via helpers/auth.ts with the seeded
 * admin user (api/migrations/000003_seed_admin.up.sql).
 */
import { expect, type Locator, type Page } from "@playwright/test";
import { loginAsAdmin } from "./auth";

export const SEEDED_ADMIN = {
  userId: "019a0000-0000-7000-8000-000000000001",
  email: "admin@example.com",
  role: "admin",
};

// The first board's id, resolved once per worker via the /board redirect
// (BRD-07). Cached so every later openBoard skips the extra listBoards
// request — the API's shared 60 req/min limit is the suite's scarcest
// resource (see playwright.config.ts workers note).
let firstBoardPath: string | null = null;

/** Signs in as the seeded admin and opens the first board: /board redirects
 * to /board/{boardId} of the first (seeded) board — BRD-07. */
export async function openBoard(page: Page): Promise<void> {
  await loginAsAdmin(page, SEEDED_ADMIN);
  await page.goto(firstBoardPath ?? "/board");
  await page.waitForURL(/\/board\/[0-9a-f-]{36}$/);
  firstBoardPath = new URL(page.url()).pathname;
  await expect(page.getByRole("heading", { name: "To Do", exact: true })).toBeVisible();
}

/** The column `<section>` (the `data-column-id` drop target) by its name. */
export function columnByName(page: Page, name: string): Locator {
  return page.locator("section[data-column-id]", {
    has: page.getByRole("heading", { name, exact: true }),
  });
}

/** A task card by its (unique per test run) title. */
export function cardByTitle(page: Page, title: string): Locator {
  return page.locator('[data-slot="card"]', { hasText: title });
}

/** Creates a task through the leftmost column's quick-add and waits for it
 * to show up on the board. */
export async function createTaskViaQuickAdd(page: Page, title: string): Promise<void> {
  await page.getByRole("button", { name: "Додати задачу" }).click();
  await page.getByLabel("Назва задачі").fill(title);
  await page.getByRole("button", { name: "Додати", exact: true }).click();
  await expect(cardByTitle(page, title)).toBeVisible();
}
