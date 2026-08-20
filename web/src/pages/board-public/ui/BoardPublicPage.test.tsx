import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { ApiClientError } from "@/shared/api/client";
import BoardPublicPage from "./BoardPublicPage";

// This task (T19) owns only web/src/pages/board-public/ and web/src/routes.ts.
// Per sad.md §5, `features/board/api` owns "typed client calls + SSE
// subscription" for BOTH the team-editor and public-viewer views, so this
// page is expected to fetch via a token-scoped `getPublicBoard` and
// subscribe via a token-scoped `usePublicBoardEvents` rather than doing its
// own fetching (mirrors T18's BoardPage / boardApi.getBoard convention).
const mockGetPublicBoard = vi.fn();

vi.mock("@/features/board/api/boardApi", () => ({
  boardApi: {
    getPublicBoard: (...args: unknown[]) => mockGetPublicBoard(...args),
  },
}));

// Live-update SSE subscription (T9/T13, AC-09) — irrelevant to what this test
// asserts (read-only rendering + the unavailable state), stub it out so the
// page doesn't need a live EventSource in jsdom.
vi.mock("@/features/board/api/useBoardEvents", () => ({
  usePublicBoardEvents: vi.fn(),
}));

const board = {
  columns: [
    {
      id: "col-1",
      name: "To do",
      position: 0,
      tasks: [
        {
          id: "task-1",
          column_id: "col-1",
          title: "Write the report",
          assignee: "Alex",
          created_at: "2026-08-20T00:00:00Z",
          updated_at: "2026-08-20T00:00:00Z",
        },
      ],
    },
    {
      id: "col-2",
      name: "Done",
      position: 1,
      tasks: [],
    },
  ],
};

beforeEach(() => {
  vi.clearAllMocks();
});

function renderPage(token = "valid-token-123") {
  return render(
    <MemoryRouter initialEntries={[`/b/${token}`]}>
      <Routes>
        <Route path="/b/:token" element={<BoardPublicPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("BoardPublicPage — public read-only board view (SCR-05)", () => {
  // AC-09: a valid token shows the board's current state, all columns and
  // tasks, view-only. AC-10: no drag/edit/delete affordances are rendered —
  // the spec's read-only guarantee is satisfied by simply not rendering
  // those controls (matching the backend's structural omission, per T10).
  it("renders the board read-only for a valid token — columns/tasks visible, no drag/edit/delete controls", async () => {
    mockGetPublicBoard.mockResolvedValue(board);

    renderPage();

    await waitFor(() => {
      expect(mockGetPublicBoard).toHaveBeenCalledWith("valid-token-123");
    });

    await waitFor(() => {
      expect(screen.getByText("To do")).toBeInTheDocument();
    });
    expect(screen.getByText("Done")).toBeInTheDocument();
    expect(screen.getByText("Write the report")).toBeInTheDocument();

    // AC-10: no quick-add affordance (create) anywhere on the read-only page.
    expect(screen.queryByRole("button", { name: /додати|add/i })).not.toBeInTheDocument();

    // AC-10: no card is rendered draggable (move affordance removed).
    expect(document.querySelectorAll('[draggable="true"]')).toHaveLength(0);

    // AC-10: clicking a task card must not open an edit/delete affordance.
    const user = userEvent.setup();
    await user.click(screen.getByText("Write the report"));
    expect(screen.queryByRole("heading", { name: /редагувати|edit/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /видалити|delete/i })).not.toBeInTheDocument();
  });

  // AC-11: an invalid/revoked token renders the SCR-06 "link unavailable"
  // state instead of the board — the backend maps this to a 404
  // `board.link_invalid` per contracts/openapi.yaml `PublicLinkInvalid`.
  it("renders the link-unavailable state for an invalid/revoked token, not the board", async () => {
    mockGetPublicBoard.mockRejectedValue(
      new ApiClientError("board.link_invalid", "this link is no longer available", 404),
    );

    renderPage("revoked-token");

    await waitFor(() => {
      expect(mockGetPublicBoard).toHaveBeenCalledWith("revoked-token");
    });

    expect(
      await screen.findByText(/недоступ|unavailable|more available|no longer/i),
    ).toBeInTheDocument();

    // The board itself must never appear alongside/instead of the unavailable state.
    expect(screen.queryByText("To do")).not.toBeInTheDocument();
    expect(screen.queryByText("Write the report")).not.toBeInTheDocument();
  });
});
