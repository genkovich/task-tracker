import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { useBoardEvents } from "@/features/board/api/useBoardEvents";
import BoardPage from "./BoardPage";

// This task (T18) owns only web/src/pages/board/ and web/src/routes.ts, and per
// its DoD must compose T15/T16/T17 "with no fetch/mutation logic of its own —
// all data access goes through those features' api/model layers". So the page
// is expected to fetch via boardApi.getBoard (T13) and subscribe via
// useBoardEvents (T13/T9) rather than doing its own fetching.
const mockGetBoard = vi.fn();
const mockEditTask = vi.fn();
const mockDeleteTask = vi.fn();
const mockCreateTask = vi.fn();

vi.mock("@/features/board/api/boardApi", () => ({
  boardApi: {
    getBoard: (...args: unknown[]) => mockGetBoard(...args),
    editTask: (...args: unknown[]) => mockEditTask(...args),
    deleteTask: (...args: unknown[]) => mockDeleteTask(...args),
    createTask: (...args: unknown[]) => mockCreateTask(...args),
    moveTask: vi.fn(),
  },
}));

// SSE subscription (T9/T13) — irrelevant to this page-composition test; stub
// it out so BoardPage doesn't need a live EventSource in jsdom.
vi.mock("@/features/board/api/useBoardEvents", () => ({
  useBoardEvents: vi.fn(),
}));

const mockShowApiError = vi.fn();
vi.mock("@/shared/lib/showApiError", () => ({
  showApiError: (...args: unknown[]) => mockShowApiError(...args),
}));

vi.mock("@/features/public-link/api/publicLinkApi", () => ({
  publicLinkApi: {
    issue: vi.fn(),
    revoke: vi.fn(),
  },
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
  public_link: null,
};

beforeEach(() => {
  vi.clearAllMocks();
});

function renderPage() {
  return render(
    <MemoryRouter>
      <BoardPage />
    </MemoryRouter>,
  );
}

// Review 2026-08-21 root I: a failed board fetch left the page as a silent
// white screen (`return null`, no .catch). SCR-01 error/loading states
// (Design/scr01-board-error-*.png, scr01-board-loading-*.png): a visible
// loading indicator while fetching, and on failure a centered message with a
// retry action that actually refetches.
describe("BoardPage — loading and error states (SCR-01)", () => {
  it("shows a loading indicator while the board fetch is pending", async () => {
    let resolveBoard!: (value: unknown) => void;
    mockGetBoard.mockReturnValue(
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

  it("shows an error message with a retry action instead of a white screen when the fetch fails", async () => {
    mockGetBoard.mockRejectedValueOnce(new Error("network down"));
    mockGetBoard.mockResolvedValueOnce(board);
    const user = userEvent.setup();

    renderPage();

    expect(
      await screen.findByText(/не вдалося завантажити дошку|couldn't load board/i),
    ).toBeInTheDocument();

    // Retry refetches and renders the board once the API recovers.
    await user.click(screen.getByRole("button", { name: /спробувати ще|retry/i }));
    await waitFor(() => {
      expect(screen.getByText("To do")).toBeInTheDocument();
    });
    expect(
      screen.queryByText(/не вдалося завантажити дошку|couldn't load board/i),
    ).not.toBeInTheDocument();
  });
});

// Re-review 2026-08-21 re2 #2: a failed refetch (SSE-triggered or after an
// edit) must NOT tear the already-rendered board down to the error screen —
// the stale board stays visible and the failure surfaces as a toast. The
// full error screen is reserved for the initial load only.
describe("BoardPage — refetch failure keeps the stale board (re2 #2)", () => {
  it("keeps the rendered board and surfaces a toast when a background refetch fails", async () => {
    mockGetBoard.mockResolvedValueOnce(board);
    mockGetBoard.mockRejectedValueOnce(new Error("network down"));

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("To do")).toBeInTheDocument();
    });

    // Simulate an SSE board event triggering the page's refetch.
    const onBoardEvent = vi.mocked(useBoardEvents).mock.calls[0][0] as () => void;
    await act(async () => {
      onBoardEvent();
    });

    // The board is still there — no error screen, no white screen.
    expect(screen.getByText("To do")).toBeInTheDocument();
    expect(screen.getByText("Write the report")).toBeInTheDocument();
    expect(
      screen.queryByText(/не вдалося завантажити дошку|couldn't load board/i),
    ).not.toBeInTheDocument();
    // ...and the failure is surfaced, not swallowed.
    expect(mockShowApiError).toHaveBeenCalledTimes(1);
  });
});

describe("BoardPage — composes the team-editor board (SCR-01)", () => {
  // DoD: page composes T15's columns/cards/quick-add — sad.md §5 pages/board
  // "composes features/board". Fetching goes through boardApi.getBoard, and
  // the loaded columns/tasks/quick-add are rendered without page-level
  // business logic.
  it("loads the board via boardApi.getBoard and renders each column with its tasks", async () => {
    mockGetBoard.mockResolvedValue(board);

    renderPage();

    await waitFor(() => {
      expect(mockGetBoard).toHaveBeenCalledTimes(1);
    });

    await waitFor(() => {
      expect(screen.getByText("To do")).toBeInTheDocument();
    });
    expect(screen.getByText("Done")).toBeInTheDocument();
    expect(screen.getByText("Write the report")).toBeInTheDocument();

    // AC-01: quick-add only renders in the leftmost column (position 0).
    expect(screen.getAllByRole("button", { name: /додати|add/i })).toHaveLength(1);
  });

  // DoD: "wires T16's edit modal to a card click" — clicking a task card
  // opens EditTaskModal (SCR-03), an observable outcome (modal contents
  // appear), not an implementation detail.
  it("opens the edit modal with the task's data when a task card is clicked", async () => {
    mockGetBoard.mockResolvedValue(board);
    const user = userEvent.setup();

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Write the report")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Write the report"));

    expect(await screen.findByRole("heading", { name: /редагувати|edit/i })).toBeInTheDocument();
    expect(screen.getByDisplayValue("Write the report")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Alex")).toBeInTheDocument();
  });

  // DoD: "mounts T17's public-link panel behind a 'поділитись' action" —
  // SCR-01 -> SCR-04 entry point per ux-flows.md.
  it("mounts the public-link panel behind a share action", async () => {
    mockGetBoard.mockResolvedValue(board);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("To do")).toBeInTheDocument();
    });

    expect(screen.getByRole("button", { name: /поділитись|share/i })).toBeInTheDocument();
  });
});
