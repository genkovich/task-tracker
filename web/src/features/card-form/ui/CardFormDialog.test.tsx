import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CardFormDialog } from "./CardFormDialog";
import { ApiClientError } from "@/shared/api/client";
import type { Card } from "@/entities/card/model/types";

const mockCreateCard = vi.fn();
const mockUpdateCard = vi.fn();

vi.mock("../api/cardFormApi", () => ({
  cardFormApi: {
    createCard: (...args: unknown[]) => mockCreateCard(...args),
    updateCard: (...args: unknown[]) => mockUpdateCard(...args),
  },
}));

const savedCard: Card = {
  id: "c1",
  name: "Write the deck",
  assignee: "Test User",
  column_status: "todo",
  created_at: "2026-08-21T10:00:00Z",
  updated_at: "2026-08-21T10:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  mockCreateCard.mockResolvedValue(savedCard);
  mockUpdateCard.mockResolvedValue(savedCard);
});

describe("CardFormDialog — add (AC-02, AC-03)", () => {
  it("submits a new card with name and assignee", async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();
    const onClose = vi.fn();

    render(<CardFormDialog open onClose={onClose} onSaved={onSaved} />);

    await user.type(screen.getByLabelText("Name"), "Write the deck");
    await user.type(screen.getByLabelText("Assignee (optional)"), "Test User");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(mockCreateCard).toHaveBeenCalledWith("Write the deck", "Test User");
    });
    expect(onSaved).toHaveBeenCalledWith(savedCard);
    expect(onClose).toHaveBeenCalled();
  });

  it("blocks save and shows an error when the server rejects an empty name (AC-03)", async () => {
    const user = userEvent.setup();
    mockCreateCard.mockRejectedValue(
      new ApiClientError("tasks.name_required", "Card name is required.", 400),
    );

    render(<CardFormDialog open onClose={vi.fn()} onSaved={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(screen.getByText("Card name is required.")).toBeInTheDocument();
    });
    // The dialog stays open so the user can fix the name.
    expect(screen.getByLabelText("Name")).toBeInTheDocument();
  });
});

describe("CardFormDialog — edit (AC-13, AC-14)", () => {
  it("pre-fills the form from the existing card and submits an update", async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();

    render(<CardFormDialog open card={savedCard} onClose={vi.fn()} onSaved={onSaved} />);

    expect(screen.getByLabelText("Name")).toHaveValue("Write the deck");
    expect(screen.getByLabelText("Assignee (optional)")).toHaveValue("Test User");

    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Write the deck (v2)");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(mockUpdateCard).toHaveBeenCalledWith("c1", "Write the deck (v2)", "Test User");
    });
    expect(onSaved).toHaveBeenCalledWith(savedCard);
  });

  it("blocks save and shows an error when the server rejects an empty name", async () => {
    const user = userEvent.setup();
    mockUpdateCard.mockRejectedValue(
      new ApiClientError("tasks.name_required", "Card name is required.", 400),
    );

    render(<CardFormDialog open card={savedCard} onClose={vi.fn()} onSaved={vi.fn()} />);

    await user.clear(screen.getByLabelText("Name"));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(screen.getByText("Card name is required.")).toBeInTheDocument();
    });
  });
});
