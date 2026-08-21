import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { BoardView } from "./BoardView";
import { ApiClientError } from "@/shared/api/client";
import type { Card } from "@/entities/card/model/types";

const mockListCards = vi.fn();
const mockMoveCard = vi.fn();
const mockDeleteCard = vi.fn();
const mockSubscribe = vi.fn();

vi.mock("../api/boardApi", () => ({
  boardApi: {
    listCards: (...args: unknown[]) => mockListCards(...args),
    moveCard: (...args: unknown[]) => mockMoveCard(...args),
    deleteCard: (...args: unknown[]) => mockDeleteCard(...args),
  },
  subscribeToBoardEvents: (...args: unknown[]) => mockSubscribe(...args),
}));

const cardA: Card = {
  id: "a1",
  name: "Write the deck",
  assignee: "Test User",
  column_status: "todo",
  created_at: "2026-08-21T10:00:00Z",
  updated_at: "2026-08-21T10:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  mockSubscribe.mockReturnValue(() => {});
});

/**
 * Simulates a Pointer Events drag (BoardCard/BoardColumn/BoardView use
 * pointerdown/pointermove/pointerup, not HTML5 dragstart/drop, so the
 * gesture works by touch as well as by mouse — spec §6 NFR). jsdom doesn't
 * implement elementFromPoint, so it's stubbed for the duration of the drag
 * to resolve to the given target column.
 */
function dragCardTo(card: HTMLElement, targetColumn: HTMLElement) {
  const elementFromPoint = vi.spyOn(document, "elementFromPoint").mockReturnValue(targetColumn);
  fireEvent.pointerDown(card, { clientX: 0, clientY: 0, pointerId: 1 });
  fireEvent.pointerMove(card, { clientX: 0, clientY: 20, pointerId: 1 }); // past DRAG_THRESHOLD_PX
  fireEvent.pointerUp(window, { clientX: 0, clientY: 20, pointerId: 1 });
  elementFromPoint.mockRestore();
}

describe("BoardView", () => {
  it("shows a loading skeleton while the initial fetch is in flight", () => {
    mockListCards.mockReturnValue(new Promise(() => {})); // never resolves
    render(<BoardView />);
    expect(screen.getByTestId("board-loading")).toBeInTheDocument();
  });

  it("shows the empty state when the board has no cards", async () => {
    mockListCards.mockResolvedValue([]);
    render(<BoardView onAddCard={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText("No cards yet")).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Add card" })).toBeInTheDocument();
  });

  it("shows the error state when the initial fetch fails", async () => {
    mockListCards.mockRejectedValue(new Error("network error"));
    render(<BoardView />);

    await waitFor(() => {
      expect(screen.getByText("Couldn't load the board")).toBeInTheDocument();
    });
  });

  it("renders cards in their column once loaded", async () => {
    mockListCards.mockResolvedValue([cardA]);
    render(<BoardView />);

    await waitFor(() => {
      expect(screen.getByTestId("card-a1")).toBeInTheDocument();
    });
    expect(screen.getByTestId("column-todo")).toContainElement(screen.getByTestId("card-a1"));
  });

  it("reverts the move and shows an error toast when the move request fails (AC-11)", async () => {
    mockListCards.mockResolvedValue([cardA]);
    mockMoveCard.mockRejectedValue(new Error("save failed"));
    render(<BoardView />);

    await waitFor(() => screen.getByTestId("card-a1"));

    const card = screen.getByTestId("card-a1");
    const targetColumn = screen.getByTestId("column-done");

    dragCardTo(card, targetColumn);

    await waitFor(() => {
      expect(mockMoveCard).toHaveBeenCalledWith("a1", "done");
    });

    // After the failed move, the card must still show in its original column.
    await waitFor(() => {
      expect(screen.getByTestId("column-todo")).toContainElement(screen.getByTestId("card-a1"));
    });
  });

  it("silently drops the card on a 404 move — the delete-wins race, no error toast (AC-15)", async () => {
    mockListCards.mockResolvedValue([cardA]);
    mockMoveCard.mockRejectedValue(new ApiClientError("tasks.card_not_found", "card not found", 404));
    render(<BoardView />);

    await waitFor(() => screen.getByTestId("card-a1"));

    dragCardTo(screen.getByTestId("card-a1"), screen.getByTestId("column-done"));

    await waitFor(() => {
      expect(mockMoveCard).toHaveBeenCalledWith("a1", "done");
    });

    // The card is gone — not reverted to its old column, no error toast.
    await waitFor(() => {
      expect(screen.queryByTestId("card-a1")).not.toBeInTheDocument();
    });
  });

  it("keeps showing the board and toasts (not blanks) when a background refetch fails", async () => {
    mockListCards.mockResolvedValueOnce([cardA]).mockRejectedValueOnce(new Error("network error"));
    let triggerRefetch: () => void = () => {};
    mockSubscribe.mockImplementation((onEvent: () => void) => {
      triggerRefetch = onEvent;
      return () => {};
    });

    render(<BoardView />);
    await waitFor(() => screen.getByTestId("card-a1"));

    triggerRefetch();

    // The card stays visible — the board is NOT replaced by the error state.
    await waitFor(() => {
      expect(screen.getByTestId("card-a1")).toBeInTheDocument();
    });
    expect(screen.queryByText("Couldn't load the board")).not.toBeInTheDocument();
  });

  it("opens the edit dialog on a tap (no movement), not on a drag", async () => {
    mockListCards.mockResolvedValue([cardA]);
    const onEditCard = vi.fn();
    render(<BoardView onEditCard={onEditCard} />);

    const card = await waitFor(() => screen.getByTestId("card-a1"));

    // A plain tap: pointerdown + pointerup with no meaningful movement.
    fireEvent.pointerDown(card, { clientX: 0, clientY: 0, pointerId: 1 });
    fireEvent.pointerUp(card, { clientX: 1, clientY: 1, pointerId: 1 });
    fireEvent.click(card);

    expect(onEditCard).toHaveBeenCalledWith(cardA);
  });

  it("tapping the delete button deletes the card, without opening the edit dialog", async () => {
    mockListCards.mockResolvedValue([cardA]);
    mockDeleteCard.mockResolvedValue(undefined);
    const onEditCard = vi.fn();
    render(<BoardView onEditCard={onEditCard} />);

    await waitFor(() => screen.getByTestId("card-a1"));
    const deleteButton = screen.getByRole("button", { name: /delete write the deck/i });

    fireEvent.pointerDown(deleteButton, { clientX: 0, clientY: 0, pointerId: 1 });
    fireEvent.pointerUp(deleteButton, { clientX: 0, clientY: 0, pointerId: 1 });
    fireEvent.click(deleteButton);

    await waitFor(() => {
      expect(mockDeleteCard).toHaveBeenCalledWith("a1");
    });
    expect(onEditCard).not.toHaveBeenCalled();
  });

  it("does not open the edit dialog when the pointer gesture was a drag", async () => {
    mockListCards.mockResolvedValue([cardA]);
    mockMoveCard.mockResolvedValue(cardA);
    const onEditCard = vi.fn();
    render(<BoardView onEditCard={onEditCard} />);

    const card = await waitFor(() => screen.getByTestId("card-a1"));
    const targetColumn = screen.getByTestId("column-done");

    dragCardTo(card, targetColumn);
    fireEvent.click(card);

    expect(onEditCard).not.toHaveBeenCalled();
  });

  it("subscribes to board events and unsubscribes on unmount", async () => {
    mockListCards.mockResolvedValue([]);
    const unsubscribe = vi.fn();
    mockSubscribe.mockReturnValue(unsubscribe);

    const { unmount } = render(<BoardView />);
    await waitFor(() => expect(mockSubscribe).toHaveBeenCalled());

    unmount();
    expect(unsubscribe).toHaveBeenCalled();
  });

  it("hides add/edit/delete affordances in read-only mode (AC-06)", async () => {
    mockListCards.mockResolvedValue([cardA]);
    render(<BoardView readOnly />);

    await waitFor(() => screen.getByTestId("card-a1"));

    expect(screen.queryByRole("button", { name: /add card/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /delete/i })).not.toBeInTheDocument();
  });
});
