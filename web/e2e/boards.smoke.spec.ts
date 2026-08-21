import { test, expect } from "@playwright/test";
import { cardByTitle, SEEDED_ADMIN } from "./helpers/board";
import { loginAsAdmin } from "./helpers/auth";

// Boards smoke (BRD-01/BRD-02/BRD-08): the dashboard lists real boards,
// creating one lands on the fresh board, and a task created there shows up
// in its leftmost column. Needs the live stack + JWT_SECRET, same as the
// other smoke specs.

test("dashboard → create board → the new board opens → a task lands in To Do", async ({
  page,
}) => {
  await loginAsAdmin(page, SEEDED_ADMIN);
  await page.goto("/dashboard");

  // BRD-01: the seeded "first board" is on the dashboard.
  await expect(page.getByRole("heading", { name: "Your boards" })).toBeVisible();
  await expect(page.getByText("Дошка команди")).toBeVisible();

  // BRD-02: create a board with a unique name.
  const boardName = `Дошка e2e ${Date.now()}`;
  await page.getByRole("button", { name: /new board/i }).click();
  await page.getByLabel("Назва дошки").fill(boardName);
  await page.getByRole("button", { name: "Створити", exact: true }).click();

  // The new board opens at its own address with the three fixed columns.
  await page.waitForURL(/\/board\/[0-9a-f-]{36}$/);
  await expect(page.getByRole("heading", { name: "To Do", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "In Progress", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Done", exact: true })).toBeVisible();

  // BRD-08: a task created here lands in this board's leftmost column.
  // (No closing dashboard round-trip: the suite runs against the API's
  // shared 60 req/min budget — see playwright.config.ts workers note.)
  const taskTitle = `Задача на новій дошці ${Date.now()}`;
  await page.getByRole("button", { name: "Додати задачу" }).click();
  await page.getByLabel("Назва задачі").fill(taskTitle);
  await page.getByRole("button", { name: "Додати", exact: true }).click();
  await expect(cardByTitle(page, taskTitle)).toBeVisible();
});
