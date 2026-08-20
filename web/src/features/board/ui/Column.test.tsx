import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Column } from "./Column";
import type { Column as ColumnState } from "@/features/board/api/types";

// api/model already exist (T13) — mocked at the module boundary QuickAddTask
// (owned by this task, per docs/features/board/tasks/T15-ui-card-column-quickadd.md
// "What") is expected to depend on: boardApi.createTask.
const mockCreateTask = vi.fn();

vi.mock("@/features/board/api/boardApi", () => ({
  boardApi: {
    createTask: (...args: unknown[]) => mockCreateTask(...args),
  },
}));

const leftmostColumn: ColumnState = {
  id: "col-1",
  name: "To do",
  position: 0,
  tasks: [],
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("Column — quick-add in the leftmost column", () => {
  // AC-01 (US-01) happy path — spec.md §5: "team member додає нову task із
  // непорожньою назвою" -> "система створює task у найлівішій column і
  // одразу показує її там". DoD (T15): "submitting quick-add with a
  // non-empty title shows the new task in the leftmost column without a
  // page reload".
  it("shows the new task in the leftmost column immediately after submitting a non-empty title", async () => {
    const user = userEvent.setup();
    const created = {
      id: "task-1",
      column_id: "col-1",
      title: "Write the report",
      assignee: null,
      created_at: "2026-08-20T00:00:00Z",
      updated_at: "2026-08-20T00:00:00Z",
    };
    mockCreateTask.mockResolvedValue(created);

    render(<Column column={leftmostColumn} isLeftmost />);

    await user.type(screen.getByLabelText(/назва|title/i), "Write the report");
    await user.click(screen.getByRole("button", { name: /додати|add/i }));

    await waitFor(() => {
      expect(mockCreateTask).toHaveBeenCalledWith({ title: "Write the report" });
    });

    // Observable outcome the AC names: the task shows up in the column right
    // away, no page reload / manual refetch required to see it.
    await waitFor(() => {
      expect(screen.getByText("Write the report")).toBeInTheDocument();
    });
  });

  // AC-02 (US-01) error — spec.md §5: "team member намагається зберегти
  // task із порожньою назвою" -> "система блокує створення й повідомляє
  // team member, що назва task обов'язкова". DoD (T15): "submitting with an
  // empty title shows an inline 'назва обов'язкова' error and makes no API
  // call".
  it("shows an inline required-title error and makes no API call when submitting an empty title", async () => {
    const user = userEvent.setup();

    render(<Column column={leftmostColumn} isLeftmost />);

    await user.click(screen.getByRole("button", { name: /додати|add/i }));

    await waitFor(() => {
      expect(screen.getByText(/назва обов'?язкова|title is required/i)).toBeInTheDocument();
    });
    expect(mockCreateTask).not.toHaveBeenCalled();
  });
});
