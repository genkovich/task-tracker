import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";
import { PublicBoardView } from "./PublicBoardView";
import { ApiClientError } from "@/shared/api/client";
import type { Card } from "@/entities/card/model/types";

const mockGetBoard = vi.fn();
const mockSubscribe = vi.fn();

vi.mock("../api/publicBoardApi", () => ({
  publicBoardApi: {
    getBoard: (...args: unknown[]) => mockGetBoard(...args),
  },
  subscribeToPublicBoardEvents: (...args: unknown[]) => mockSubscribe(...args),
}));

const cardA: Card = {
  id: "a1",
  name: "On the board",
  assignee: null,
  column_status: "todo",
  created_at: "2026-08-21T10:00:00Z",
  updated_at: "2026-08-21T10:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  mockSubscribe.mockReturnValue(() => {});
});

describe("PublicBoardView", () => {
  it("shows a loading skeleton while the initial fetch is in flight", () => {
    mockGetBoard.mockReturnValue(new Promise(() => {}));
    render(<PublicBoardView token="tok" onNotFound={vi.fn()} />);
    expect(screen.getByTestId("public-board-loading")).toBeInTheDocument();
  });

  it("shows the empty state when the board has no cards", async () => {
    mockGetBoard.mockResolvedValue([]);
    render(<PublicBoardView token="tok" onNotFound={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText("No cards yet")).toBeInTheDocument();
    });
  });

  it("renders cards read-only, with no edit affordances (AC-06, AC-08)", async () => {
    mockGetBoard.mockResolvedValue([cardA]);
    render(<PublicBoardView token="tok" onNotFound={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByTestId("card-a1")).toBeInTheDocument();
    });
    expect(screen.queryByRole("button", { name: /add card/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /delete/i })).not.toBeInTheDocument();
  });

  it("calls onNotFound when the token resolves 404 (AC-05)", async () => {
    mockGetBoard.mockRejectedValue(new ApiClientError("common.not_found", "Not found.", 404));
    const onNotFound = vi.fn();
    render(<PublicBoardView token="bad-token" onNotFound={onNotFound} />);

    await waitFor(() => {
      expect(onNotFound).toHaveBeenCalled();
    });
  });

  it("shows a retry-able error state for a non-404 failure", async () => {
    mockGetBoard.mockRejectedValue(new Error("network error"));
    render(<PublicBoardView token="tok" onNotFound={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText("Couldn't load the board")).toBeInTheDocument();
    });
  });

  it("keeps showing the board (not the error state) when a background refetch fails", async () => {
    mockGetBoard.mockResolvedValueOnce([cardA]).mockRejectedValueOnce(new Error("network error"));
    let triggerRefetch: () => void = () => {};
    mockSubscribe.mockImplementation((_token, onEvent) => {
      triggerRefetch = onEvent;
      return () => {};
    });

    render(<PublicBoardView token="tok" onNotFound={vi.fn()} />);
    await waitFor(() => screen.getByTestId("card-a1"));

    act(() => {
      triggerRefetch();
    });

    await waitFor(() => {
      expect(screen.getByTestId("card-a1")).toBeInTheDocument();
    });
    expect(screen.queryByText("Couldn't load the board")).not.toBeInTheDocument();
  });

  it("calls onNotFound when the SSE stream reports the link was disabled (AC-12)", async () => {
    mockGetBoard.mockResolvedValue([cardA]);
    let disabledCallback: (() => void) | undefined;
    mockSubscribe.mockImplementation((_token, _onEvent, onLinkDisabled) => {
      disabledCallback = onLinkDisabled;
      return () => {};
    });

    const onNotFound = vi.fn();
    render(<PublicBoardView token="tok" onNotFound={onNotFound} />);

    await waitFor(() => screen.getByTestId("card-a1"));
    await waitFor(() => expect(mockSubscribe).toHaveBeenCalled());

    act(() => {
      disabledCallback?.();
    });

    await waitFor(() => {
      expect(onNotFound).toHaveBeenCalled();
    });
  });
});
