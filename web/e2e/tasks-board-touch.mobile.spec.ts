import { test, expect } from "@playwright/test";

// Runs under the "mobile" Playwright project (Pixel 7, hasTouch: true) —
// proves the drag genuinely works by touch, not just by mouse (spec §6 NFR
// "Touch-переміщення карток"). Needs the live stack, same as
// tasks-board.smoke.spec.ts.
test("dragging a card by touch moves it to another column", async ({ page }) => {
  const cardName = `Touch card ${Date.now()}`;

  await page.goto("/board");
  await page.getByRole("button", { name: /^add card/i }).first().click();
  await page.getByLabel("Name").fill(cardName);
  await page.getByRole("button", { name: "Save" }).click();

  const card = page.getByTestId(/^card-/).filter({ hasText: cardName });
  await expect(card).toBeVisible();
  await expect(page.getByTestId("column-todo")).toContainText(cardName);
  // The add-card dialog's close animation keeps its fixed, viewport-covering
  // overlay mounted for a moment after Save — wait it out, or the touch
  // gesture below hits the fading overlay instead of the column beneath it.
  await expect(page.getByRole("dialog")).toHaveCount(0);

  // Manual touch sequence (not Playwright's higher-level dragTo, which
  // targets HTML5 drag internals and is flaky against a custom Pointer
  // Events implementation) — a real pointerdown/move/up gesture with
  // pointerType "touch", exactly what BoardCard/BoardColumn listen for.
  const cardBox = await card.boundingBox();
  const targetBox = await page.getByTestId("column-in_progress").boundingBox();
  if (!cardBox || !targetBox) throw new Error("could not measure drag source/target");

  const startX = cardBox.x + cardBox.width / 2;
  const startY = cardBox.y + cardBox.height / 2;
  const endX = targetBox.x + targetBox.width / 2;
  const endY = targetBox.y + targetBox.height / 2;

  await card.dispatchEvent("pointerdown", {
    pointerId: 1,
    pointerType: "touch",
    clientX: startX,
    clientY: startY,
    bubbles: true,
  });
  for (let i = 1; i <= 5; i++) {
    const x = startX + ((endX - startX) * i) / 5;
    const y = startY + ((endY - startY) * i) / 5;
    await card.dispatchEvent("pointermove", {
      pointerId: 1,
      pointerType: "touch",
      clientX: x,
      clientY: y,
      bubbles: true,
    });
  }
  await page.dispatchEvent("body", "pointerup", {
    pointerId: 1,
    pointerType: "touch",
    clientX: endX,
    clientY: endY,
    bubbles: true,
  });

  await expect(page.getByTestId("column-in_progress")).toContainText(cardName, {
    timeout: 10_000,
  });
});
