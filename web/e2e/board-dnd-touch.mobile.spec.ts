import { test, expect } from "@playwright/test";
import { cardByTitle, columnByName, createTaskViaQuickAdd, openBoard } from "./helpers/board";

// Runs under the "mobile" Playwright project (Pixel 7, hasTouch: true) —
// proves the drag genuinely works by touch, which HTML5 drag events never
// did. Needs the live stack, same as the smoke specs.
//
// Manual touch sequence (not Playwright's higher-level dragTo, which targets
// HTML5 drag internals and is flaky against a custom Pointer Events
// implementation) — a real pointerdown/move/up gesture with pointerType
// "touch", exactly what TaskCard listens for. All events are dispatched at
// the card: with pointer capture that is where a real browser delivers them
// for the whole gesture.
test("dragging a card by touch moves it to another column and persists", async ({ page }) => {
  const title = `DnD touch ${Date.now()}`;
  await openBoard(page);
  await createTaskViaQuickAdd(page, title);

  const card = cardByTitle(page, title);
  const target = columnByName(page, "In Progress");
  // On the phone viewport the columns stack vertically — make sure the drop
  // point (the target column's heading) is inside the viewport, since
  // useBoardDnd resolves it via document.elementFromPoint.
  const heading = target.getByRole("heading");
  await heading.scrollIntoViewIfNeeded();

  const cardBox = await card.boundingBox();
  const targetBox = await heading.boundingBox();
  if (!cardBox || !targetBox) throw new Error("could not measure drag source/target");

  const startX = cardBox.x + cardBox.width / 2;
  const startY = cardBox.y + cardBox.height / 2;
  const endX = targetBox.x + targetBox.width / 2;
  const endY = targetBox.y + targetBox.height / 2;

  await card.dispatchEvent("pointerdown", {
    pointerId: 1,
    pointerType: "touch",
    isPrimary: true,
    clientX: startX,
    clientY: startY,
    bubbles: true,
  });
  for (let i = 1; i <= 5; i++) {
    await card.dispatchEvent("pointermove", {
      pointerId: 1,
      pointerType: "touch",
      isPrimary: true,
      clientX: startX + ((endX - startX) * i) / 5,
      clientY: startY + ((endY - startY) * i) / 5,
      bubbles: true,
    });
  }
  await card.dispatchEvent("pointerup", {
    pointerId: 1,
    pointerType: "touch",
    isPrimary: true,
    clientX: endX,
    clientY: endY,
    bubbles: true,
  });

  await expect(target).toContainText(title, { timeout: 10_000 });

  // Persisted through the API, not just optimistically.
  await page.reload();
  await expect(columnByName(page, "In Progress")).toContainText(title, { timeout: 10_000 });
});
