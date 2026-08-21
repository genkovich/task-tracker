import { test, expect } from "@playwright/test";

// Full board lifecycle across two browser contexts — the team-member editor
// and a public viewer — per ux-flows.md's flows. Needs the live stack
// (`make up`, or `go run ./cmd/api` + `npm run dev` against a real
// Postgres) — the board endpoints are real HTTP + SSE, not mockable here.
test.describe("tasks board — full lifecycle", () => {
  // All tests share one live board (no per-test backend reset) — run them
  // serially so they don't race each other's mutations.
  test.describe.configure({ mode: "serial" });

  test("add, drag, edit, share, view, disable, delete", async ({ browser }) => {
    const cardName = `E2E card ${Date.now()}`;
    const renamedCardName = `${cardName} (renamed)`;

    const editorContext = await browser.newContext();
    const editor = await editorContext.newPage();
    await editor.goto("/board");

    // --- US-02: add a new card (AC-02) ---
    // "Add card" is the EmptyState's CTA on an empty board; "Add card to
    // To Do" is the per-column "+" once ≥1 card exists. Tests run in
    // parallel against the same live board, so match either.
    await editor
      .getByRole("button", { name: /^add card/i })
      .first()
      .click();
    await editor.getByLabel("Name").fill(cardName);
    await editor.getByLabel("Assignee (optional)").fill("E2E Bot");
    await editor.getByRole("button", { name: "Save" }).click();

    const editorCard = editor.getByTestId(/^card-/).filter({ hasText: cardName });
    await expect(editorCard).toBeVisible();
    await expect(editor.getByTestId("column-todo")).toContainText(cardName);

    // --- US-01: drag the card to In Progress (AC-01) ---
    await editorCard.dragTo(editor.getByTestId("column-in_progress"));
    await expect(editor.getByTestId("column-in_progress")).toContainText(cardName, {
      timeout: 10_000,
    });

    // --- US-07: edit the card's name (AC-13) ---
    await editorCard.click();
    const nameInput = editor.getByLabel("Name");
    await nameInput.fill(renamedCardName);
    await editor.getByRole("button", { name: "Save" }).click();
    await expect(editor.getByTestId("column-in_progress")).toContainText(renamedCardName);

    // --- US-03: get a public link (AC-09) ---
    await editor.getByRole("button", { name: "Get link" }).click();
    const linkInput = editor.getByTestId("public-link-input");
    await expect(linkInput).toBeVisible({ timeout: 10_000 });
    const shareUrl = await linkInput.inputValue();
    expect(shareUrl).toContain("/b/");

    // --- US-04: a viewer opens the link and sees the live board (AC-05, AC-06, AC-08) ---
    const viewerContext = await browser.newContext();
    const viewer = await viewerContext.newPage();
    const path = new URL(shareUrl).pathname;
    await viewer.goto(path);

    await expect(viewer.getByText("View only")).toBeVisible();
    await expect(viewer.getByTestId("column-in_progress")).toContainText(renamedCardName, {
      timeout: 10_000,
    });
    // Read-only: no edit/delete affordances anywhere on the viewer's page.
    await expect(viewer.getByRole("button", { name: /add card/i })).toHaveCount(0);
    await expect(viewer.getByRole("button", { name: /delete/i })).toHaveCount(0);

    // --- US-05: disable the link (AC-04), viewer auto-transitions (AC-12) ---
    await editor.getByRole("button", { name: "Disable link" }).click();
    await expect(editor.getByRole("button", { name: "Get link" })).toBeVisible({
      timeout: 10_000,
    });
    await expect(viewer.getByRole("heading", { name: /page not found/i })).toBeVisible({
      timeout: 10_000,
    });

    // --- US-06: delete the card (AC-10) ---
    await editorCard.locator('button[aria-label^="Delete"]').click();
    await expect(editor.getByTestId(/^card-/).filter({ hasText: renamedCardName })).toHaveCount(0);

    await editorContext.close();
    await viewerContext.close();
  });

  test("empty name is rejected on add (AC-03)", async ({ page }) => {
    await page.goto("/board");
    await page
      .getByRole("button", { name: /^add card/i })
      .first()
      .click();
    await page.getByRole("button", { name: "Save" }).click();
    // Lowercase, no trailing punctuation — the repo's error-string
    // convention (.claude/rules/go-naming.md), returned as-is by the API.
    await expect(page.getByText("card name is required")).toBeVisible();
  });

  test("a never-valid public link renders the generic not-found page (AC-05)", async ({ page }) => {
    await page.goto("/b/this-token-never-existed");
    await expect(page.getByRole("heading", { name: /page not found/i })).toBeVisible();
  });
});
