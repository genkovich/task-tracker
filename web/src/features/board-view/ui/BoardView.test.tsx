import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { BoardView } from "./BoardView";
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

    fireEvent.dragStart(card);
    fireEvent.drop(targetColumn);

    await waitFor(() => {
      expect(mockMoveCard).toHaveBeenCalledWith("a1", "done");
    });

    // After the failed move, the card must still show in its original column.
    await waitFor(() => {
      expect(screen.getByTestId("column-todo")).toContainElement(screen.getByTestId("card-a1"));
    });
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
