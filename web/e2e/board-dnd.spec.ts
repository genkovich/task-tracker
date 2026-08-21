import { test, expect } from "@playwright/test";
import { cardByTitle, columnByName, createTaskViaQuickAdd, openBoard } from "./helpers/board";

// Desktop mouse drag over the Pointer Events implementation (TaskCard
// captures the pointer, useBoardDnd resolves the hovered column via
// elementFromPoint). Real mouse input — not dispatchEvent — so pointer
// capture and the pointer-events:none hit-testing trick are exercised for
// real. Needs the live stack, same as the smoke specs.

test("dragging a card with the mouse moves it to another column and persists", async ({ page }) => {
  const title = `DnD mouse ${Date.now()}`;
  await openBoard(page);
  await createTaskViaQuickAdd(page, title);

  const card = cardByTitle(page, title);
  const target = columnByName(page, "In Progress");
  const cardBox = await card.boundingBox();
  // Aim at the column heading — the top of the section, always on screen.
  const targetBox = await target.getByRole("heading").boundingBox();
  if (!cardBox || !targetBox) throw new Error("could not measure drag source/target");

  await page.mouse.move(cardBox.x + cardBox.width / 2, cardBox.y + cardBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(targetBox.x + targetBox.width / 2, targetBox.y + targetBox.height / 2, {
    steps: 12,
  });
  // The hovered column is highlighted as the drop target while dragging.
  await expect(target).toHaveClass(/ring-1/);
  await page.mouse.up();

  await expect(target).toContainText(title, { timeout: 10_000 });

  // The optimistic move must have persisted through the API, not just
  // locally — a reload shows the card in its new column.
  await page.reload();
  await expect(columnByName(page, "In Progress")).toContainText(title, { timeout: 10_000 });
});

test("Escape aborts the drag — the card stays in its column", async ({ page }) => {
  const title = `DnD escape ${Date.now()}`;
  await openBoard(page);
  await createTaskViaQuickAdd(page, title);

  const card = cardByTitle(page, title);
  const target = columnByName(page, "In Progress");
  const cardBox = await card.boundingBox();
  const targetBox = await target.getByRole("heading").boundingBox();
  if (!cardBox || !targetBox) throw new Error("could not measure drag source/target");

  await page.mouse.move(cardBox.x + cardBox.width / 2, cardBox.y + cardBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(targetBox.x + targetBox.width / 2, targetBox.y + targetBox.height / 2, {
    steps: 12,
  });
  await expect(target).toHaveClass(/ring-1/);
  await page.keyboard.press("Escape");
  await page.mouse.up();

  // No move happened — not even optimistically — and none is persisted.
  await expect(columnByName(page, "To Do")).toContainText(title);
  await expect(target).not.toContainText(title);
  await page.reload();
  await expect(columnByName(page, "To Do")).toContainText(title, { timeout: 10_000 });
});
