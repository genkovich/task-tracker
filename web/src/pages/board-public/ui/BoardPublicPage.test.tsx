import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { ApiClientError } from "@/shared/api/client";
import { usePublicBoardEvents } from "@/features/board/api/useBoardEvents";
import BoardPublicPage from "./BoardPublicPage";

// This task (T19) owns only web/src/pages/board-public/ and web/src/routes.ts.
// Per sad.md §5, `features/board/api` owns "typed client calls + SSE
// subscription" for BOTH the team-editor and public-viewer views, so this
// page is expected to fetch via a token-scoped `getPublicBoard` and
// subscribe via a token-scoped `usePublicBoardEvents` rather than doing its
// own fetching (mirrors T18's BoardPage / boardApi.getBoard convention).
const mockGetPublicBoard = vi.fn();
const mockGetPublicTask = vi.fn();

vi.mock("@/features/board/api/boardApi", () => ({
  boardApi: {
    getPublicBoard: (...args: unknown[]) => mockGetPublicBoard(...args),
    getPublicTask: (...args: unknown[]) => mockGetPublicTask(...args),
  },
}));

// Live-update SSE subscription (T9/T13, AC-09) — irrelevant to what this test
// asserts (read-only rendering + the unavailable state), stub it out so the
// page doesn't need a live EventSource in jsdom.
vi.mock("@/features/board/api/useBoardEvents", () => ({
  usePublicBoardEvents: vi.fn(),
}));

const mockShowApiError = vi.fn();
vi.mock("@/shared/lib/showApiError", () => ({
  showApiError: (...args: unknown[]) => mockShowApiError(...args),
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
          priority: "high",
          due_date: "2026-09-01",
          has_description: true,
          comment_count: 1,
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

// Review 2026-08-21 root I: only the 404 branch was handled — any other
// failure (network, 500) rethrew into the void and left a white screen, and
// there was no loading indicator. Mirror BoardPage's SCR-01 states here.
describe("BoardPublicPage — loading and generic error states", () => {
  it("shows a loading indicator while the public board fetch is pending", async () => {
    let resolveBoard!: (value: unknown) => void;
    mockGetPublicBoard.mockReturnValue(
      new Promise((resolve) => {
        resolveBoard = resolve;
      }),
    );

    renderPage();

    expect(screen.getByRole("status")).toBeInTheDocument();

    resolveBoard(board);
    await waitFor(() => {
      expect(screen.getByText("To do")).toBeInTheDocument();
    });
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("shows an error message with a retry action for a non-404 failure, not a white screen", async () => {
    mockGetPublicBoard.mockRejectedValueOnce(
      new ApiClientError("internal", "internal server error", 500),
    );
    mockGetPublicBoard.mockResolvedValueOnce(board);
    const user = userEvent.setup();

    renderPage();

    expect(
      await screen.findByText(/не вдалося завантажити дошку|couldn't load board/i),
    ).toBeInTheDocument();
    // A generic failure is not the SCR-06 unavailable state.
    expect(screen.queryByText(/більше недоступний/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /спробувати ще|retry/i }));
    await waitFor(() => {
      expect(screen.getByText("To do")).toBeInTheDocument();
    });
  });
});

// Re-review 2026-08-21 re2 #2: a failed background refetch must not wipe an
// already-rendered public board. Two distinct failure meanings here:
// non-404 (network/500) = transient → keep the stale board + toast;
// 404 `board.link_invalid` = the link was revoked → SCR-06 always, even
// over a rendered board (that is NOT a stale-state case).
describe("BoardPublicPage — refetch failures after a rendered board (re2 #2)", () => {
  function fireBoardEvent() {
    const onBoardEvent = vi.mocked(usePublicBoardEvents).mock.calls[0][1] as () => void;
    return act(async () => {
      onBoardEvent();
    });
  }

  it("keeps the rendered board and surfaces a toast when a non-404 refetch fails", async () => {
    mockGetPublicBoard.mockResolvedValueOnce(board);
    mockGetPublicBoard.mockRejectedValueOnce(
      new ApiClientError("internal", "internal server error", 500),
    );

    renderPage();
    await waitFor(() => {
      expect(screen.getByText("To do")).toBeInTheDocument();
    });

    await fireBoardEvent();

    expect(screen.getByText("To do")).toBeInTheDocument();
    expect(screen.getByText("Write the report")).toBeInTheDocument();
    expect(
      screen.queryByText(/не вдалося завантажити дошку|couldn't load board/i),
    ).not.toBeInTheDocument();
    expect(mockShowApiError).toHaveBeenCalledTimes(1);
  });

  it("switches to the link-unavailable state when a refetch 404s over a rendered board", async () => {
    mockGetPublicBoard.mockResolvedValueOnce(board);
    mockGetPublicBoard.mockRejectedValueOnce(
      new ApiClientError("board.link_invalid", "this link is no longer available", 404),
    );

    renderPage();
    await waitFor(() => {
      expect(screen.getByText("To do")).toBeInTheDocument();
    });

    await fireBoardEvent();

    expect(await screen.findByText(/більше недоступний/i)).toBeInTheDocument();
    expect(screen.queryByText("To do")).not.toBeInTheDocument();
  });
});

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
    // Since the tasks feature the click does open something — read-only
    // details (TSK-12) — so this pins that it is still not editable.
    const user = userEvent.setup();
    await user.click(screen.getByText("Write the report"));
    expect(screen.queryByRole("heading", { name: /редагувати|edit/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /видалити|delete/i })).not.toBeInTheDocument();
  });

  // TSK-12: a viewer's card click opens the details, fetched through the very
  // token they hold — never through the editor route.
  it("opens read-only task details through the public token when a card is clicked", async () => {
    mockGetPublicBoard.mockResolvedValue(board);
    mockGetPublicTask.mockResolvedValue({
      task: {
        id: "task-1",
        column_id: "col-1",
        title: "Write the report",
        assignee: "Alex",
        description: "Зібрати цифри за тиждень",
        priority: "high",
        due_date: "2026-09-01",
        created_at: "2026-08-20T00:00:00Z",
        updated_at: "2026-08-20T00:00:00Z",
      },
      comments: [
        {
          id: "comment-1",
          task_id: "task-1",
          author: "Grace",
          body: "Цифри вже є в дашборді",
          created_at: "2026-08-21T09:00:00Z",
        },
      ],
    });

    renderPage("valid-token");
    await screen.findByText("Write the report");

    const user = userEvent.setup();
    await user.click(screen.getByText("Write the report"));

    expect(await screen.findByText("Зібрати цифри за тиждень")).toBeInTheDocument();
    expect(mockGetPublicTask).toHaveBeenCalledWith("valid-token", "task-1");
    expect(screen.getByText("Цифри вже є в дашборді")).toBeInTheDocument();

    // Read-only is a fact about the markup: nothing here can be typed into.
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Додати коментар" })).not.toBeInTheDocument();
  });

  // Review I2 / TSK-13: the link died (or the task was never on this board)
  // while the viewer was reading. That belongs on the same honest SCR-06
  // screen a dead link gets — not on a generic "could not load the details"
  // inside a dialog the board still shows behind it.
  it("sends the viewer to the link-unavailable screen when the details come back link_invalid", async () => {
    mockGetPublicBoard.mockResolvedValue(board);
    mockGetPublicTask.mockRejectedValue(
      new ApiClientError("board.link_invalid", "this link is no longer available", 404),
    );

    renderPage("valid-token");
    await screen.findByText("Write the report");

    const user = userEvent.setup();
    await user.click(screen.getByText("Write the report"));

    expect(await screen.findByText(/недоступ|no longer/i)).toBeInTheDocument();
    expect(screen.queryByText(/не вдалося завантажити/i)).not.toBeInTheDocument();
    // SCR-06 replaces the screen — the board must not stay behind it.
    expect(screen.queryByText("To do")).not.toBeInTheDocument();
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
