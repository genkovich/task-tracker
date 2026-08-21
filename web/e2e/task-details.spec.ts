import { test, expect } from "@playwright/test";
import { cardByTitle, createTaskViaQuickAdd, openBoard } from "./helpers/board";

// Rich tasks end to end (docs/features/tasks/tasks/T12): the one path no
// single layer can prove on its own — an editor fills in the details, the
// card starts showing markers, and a viewer on the public link reads exactly
// the same thing without a single field to change it.
//
// Deliberately one scenario, not five: the suite shares the API's 60 req/min
// per-IP budget, and every mutation broadcasts an SSE event that refetches
// every open board.
//
// KNOWN CONSTRAINT: the whole `default` project already spends ~55 of the
// API's 60 requests per minute before this file starts, so running it last in
// one process leaves nothing for it — `/auth/me` answers 429, the client
// reads that as a lost session and bounces to the landing page. Run this spec
// in its own invocation until the limit is configurable for the local stack:
//   npx playwright test --config e2e/playwright.config.ts \
//     --project=default e2e/task-details.spec.ts
// (Raising it per-spec via X-Forwarded-For does not work: the limiter would
// honour the header, but the API's CORS allow-list does not include it, so
// the browser never sends it.)

// A deadline far enough ahead that the run never drifts past it.
function futureDueDate(): { value: string; label: string } {
  const due = new Date();
  due.setDate(due.getDate() + 30);
  const value = `${due.getFullYear()}-${String(due.getMonth() + 1).padStart(2, "0")}-${String(due.getDate()).padStart(2, "0")}`;
  const monthsShort = [
    "січ",
    "лют",
    "бер",
    "квіт",
    "трав",
    "черв",
    "лип",
    "серп",
    "вер",
    "жовт",
    "лист",
    "груд",
  ];
  return { value, label: `${due.getDate()} ${monthsShort[due.getMonth()]}` };
}

test("task details: description + priority + deadline + comment → card markers → public read-only view", async ({
  page,
  context,
}) => {
  await openBoard(page);

  const title = `Деталі e2e ${Date.now()}`;
  await createTaskViaQuickAdd(page, title);

  // One modal session, not two: every save and every comment broadcasts a
  // board.state_changed that refetches the open board, and the suite shares
  // one 60 req/min budget across every spec.
  await cardByTitle(page, title).click();
  await expect(page.getByRole("heading", { name: "Деталі задачі" })).toBeVisible();

  // TSK-08: a comment lands in the thread and bumps the card's counter.
  await page.getByLabel("Новий коментар").fill("Стенд піднято, лишився TLS");
  await page.getByRole("button", { name: "Додати коментар" }).click();
  await expect(page.getByText("Стенд піднято, лишився TLS")).toBeVisible();

  // TSK-01/TSK-03/TSK-05: fill in the details and save.
  const due = futureDueDate();
  await page.getByLabel("Опис").fill("Зібрати цифри за тиждень");
  await page.getByLabel("Пріоритет").selectOption("high");
  await page.getByLabel("Дедлайн").fill(due.value);
  await page.getByRole("button", { name: "Зберегти" }).click();

  // On the card: markers, never content.
  const card = cardByTitle(page, title);
  await expect(card.locator('[data-priority="high"]')).toBeVisible();
  await expect(card.locator('[data-slot="due-badge"]')).toHaveText(due.label);
  await expect(card.locator('[data-slot="has-description"]')).toBeVisible();
  await expect(card.locator('[data-slot="comment-count"]')).toHaveText("1");
  await expect(card).not.toContainText("Зібрати цифри за тиждень");

  // TSK-12: the viewer opens the same card on the public link.
  await page.getByRole("button", { name: "Поділитись" }).click();
  const issueButton = page.getByRole("button", { name: "Отримати лінк" });
  if (await issueButton.isVisible().catch(() => false)) {
    await issueButton.click();
  }
  const publicUrl = await page.getByText(/\/b\/[A-Za-z0-9_-]+/).innerText();
  expect(publicUrl).toContain("/b/");

  const viewer = await context.newPage();
  await viewer.goto(new URL(publicUrl).pathname);
  await expect(viewer.getByText("Лише перегляд")).toBeVisible();

  const viewerCard = cardByTitle(viewer, title);
  await expect(viewerCard.locator('[data-priority="high"]')).toBeVisible();
  await expect(viewerCard.locator('[data-slot="comment-count"]')).toHaveText("1");

  await viewerCard.click();
  await expect(viewer.getByText("Зібрати цифри за тиждень")).toBeVisible();
  await expect(viewer.getByText("Стенд піднято, лишився TLS")).toBeVisible();

  // Read-only is a fact about the markup: there is nothing here to type into
  // and nothing to press that would change the board.
  const dialog = viewer.getByRole("dialog");
  await expect(dialog.locator("input")).toHaveCount(0);
  await expect(dialog.locator("textarea")).toHaveCount(0);
  await expect(dialog.locator("select")).toHaveCount(0);
  await expect(dialog.getByRole("button", { name: "Додати коментар" })).toHaveCount(0);
  await expect(dialog.getByRole("button", { name: "Видалити" })).toHaveCount(0);

  await viewer.close();
});
