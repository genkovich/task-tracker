import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import DashboardPage from "./DashboardPage";

const mockListBoards = vi.fn();
const mockCreateBoard = vi.fn();

vi.mock("@/features/board/api/boardApi", () => ({
  boardApi: {
    listBoards: (...args: unknown[]) => mockListBoards(...args),
    createBoard: (...args: unknown[]) => mockCreateBoard(...args),
  },
}));

const mockNavigate = vi.fn();
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return { ...actual, useNavigate: () => mockNavigate };
});

const boards = [
  { id: "board-1", name: "Дошка команди", created_at: "2026-08-20T00:00:00Z", task_count: 3 },
  { id: "board-2", name: "Воркшоп", created_at: "2026-08-21T00:00:00Z", task_count: 0 },
];

beforeEach(() => {
  vi.clearAllMocks();
});

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/dashboard"]}>
      <DashboardPage />
    </MemoryRouter>,
  );
}

describe("DashboardPage — live board list (BRD-01)", () => {
  it("lists every board with its name and task count", async () => {
    mockListBoards.mockResolvedValue(boards);

    renderPage();

    expect(await screen.findByText("Дошка команди")).toBeInTheDocument();
    expect(screen.getByText("Воркшоп")).toBeInTheDocument();
    expect(screen.getByText("3 задач")).toBeInTheDocument();
    expect(screen.getByText("0 задач")).toBeInTheDocument();
  });

  it("navigates to the board on click (BRD-04 entry)", async () => {
    mockListBoards.mockResolvedValue(boards);
    const user = userEvent.setup();

    renderPage();
    await user.click(await screen.findByText("Воркшоп"));

    expect(mockNavigate).toHaveBeenCalledWith("/board/board-2");
  });

  it("shows the empty state when there are no boards", async () => {
    mockListBoards.mockResolvedValue([]);

    renderPage();

    expect(await screen.findByText("No boards yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create board/i })).toBeInTheDocument();
  });
});

describe("DashboardPage — create board (BRD-02/BRD-03)", () => {
  it("creates a board with the typed name and opens it", async () => {
    mockListBoards.mockResolvedValue(boards);
    mockCreateBoard.mockResolvedValue({ id: "board-3", name: "Нова", columns: [] });
    const user = userEvent.setup();

    renderPage();
    await screen.findByText("Дошка команди");

    await user.click(screen.getByRole("button", { name: /new board/i }));
    await user.type(screen.getByLabelText("Назва дошки"), "Нова");
    await user.click(screen.getByRole("button", { name: "Створити" }));

    await waitFor(() => {
      expect(mockCreateBoard).toHaveBeenCalledWith("Нова");
    });
    expect(mockNavigate).toHaveBeenCalledWith("/board/board-3");
  });

  it("blocks an empty name without calling the API (BRD-03)", async () => {
    mockListBoards.mockResolvedValue(boards);
    const user = userEvent.setup();

    renderPage();
    await screen.findByText("Дошка команди");

    await user.click(screen.getByRole("button", { name: /new board/i }));
    await user.click(screen.getByRole("button", { name: "Створити" }));

    expect(await screen.findByText("Назва обов'язкова")).toBeInTheDocument();
    expect(mockCreateBoard).not.toHaveBeenCalled();
  });
});
