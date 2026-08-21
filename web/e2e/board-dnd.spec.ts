import { test, expect, type Locator } from "@playwright/test";
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

// A2: previously the drag aimed at the column heading specifically — "the
// rest of the column [was] unreliable" — because an empty/short column had
// no real body to drop into (Column now stretches h-full/flex-1/min-h-24)
// and the dragged card itself got clipped by the columns row's
// overflow-x-auto once it crossed the row's edge (now a portalled ghost).
// One test, two drops into "Done": first while it's still genuinely empty,
// then into the empty space the first card leaves below it — proving both
// fixes hold, not just the heading shortcut. The second drop reuses the
// "DnD escape ..." card the test above left behind in "To Do" instead of
// creating another task, and neither drop reloads to re-check persistence
// (already pinned by the tests above) — this suite runs against the API's
// shared 60 req/min budget (see playwright.config.ts).
test("dropping away from the heading — into an empty column, then below the card it leaves — both move a card", async ({
  page,
}) => {
  await openBoard(page);
  const target = columnByName(page, "Done");

  async function dropCardIntoDoneBody(card: Locator, expectTitle: string) {
    const cardBox = await card.boundingBox();
    const targetBox = await target.boundingBox();
    if (!cardBox || !targetBox) throw new Error("could not measure drag source/target");

    const dropX = targetBox.x + targetBox.width / 2;
    const dropY = targetBox.y + targetBox.height - 16;

    await page.mouse.move(cardBox.x + cardBox.width / 2, cardBox.y + cardBox.height / 2);
    await page.mouse.down();
    await page.mouse.move(dropX, dropY, { steps: 12 });
    await expect(target).toHaveClass(/ring-1/);
    await page.mouse.up();
    await expect(target).toContainText(expectTitle, { timeout: 10_000 });
  }

  // 1) "Done" starts empty — the drop lands in its body, not on the heading.
  const title = `DnD empty column ${Date.now()}`;
  await createTaskViaQuickAdd(page, title);
  await dropCardIntoDoneBody(cardByTitle(page, title), title);

  // 2) "Done" now holds a card — the columns row stretches every column to
  // the tallest one's height (A2: `items-stretch` is the default once
  // `sm:items-start` is gone), so there's still real empty space below it.
  // The only card left in "To Do" at this point is the Escape test's.
  const leftover = columnByName(page, "To Do").locator('[data-slot="card"]').first();
  const leftoverTitle = await readCardTitle(leftover);
  await dropCardIntoDoneBody(leftover, leftoverTitle);
});

async function readCardTitle(card: Locator): Promise<string> {
  const text = await card.innerText();
  return text.trim();
}
