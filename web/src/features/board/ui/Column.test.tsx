import { useState } from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Column } from "./Column";
import type { Column as ColumnState, Task } from "@/features/board/api/types";

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

describe("Column — renders the tasks the caller passes in", () => {
  // Review 2026-08-21 root E: Column froze `column.tasks` in useState, so a
  // re-render with fresh tasks (optimistic move, rollback, public refetch)
  // changed nothing on screen. Pin: new `column.tasks` -> new cards visible.
  it("shows the new tasks when re-rendered with an updated column.tasks", () => {
    const task = (id: string, title: string) => ({
      id,
      column_id: "col-1",
      title,
      assignee: null,
      created_at: "2026-08-20T00:00:00Z",
      updated_at: "2026-08-20T00:00:00Z",
    });

    const { rerender } = render(
      <Column column={{ ...leftmostColumn, tasks: [task("task-1", "First task")] }} />,
    );
    expect(screen.getByText("First task")).toBeInTheDocument();

    rerender(
      <Column
        column={{
          ...leftmostColumn,
          tasks: [task("task-1", "First task"), task("task-2", "Second task")],
        }}
      />,
    );

    expect(screen.getByText("Second task")).toBeInTheDocument();

    rerender(<Column column={{ ...leftmostColumn, tasks: [task("task-2", "Second task")] }} />);

    expect(screen.queryByText("First task")).not.toBeInTheDocument();
    expect(screen.getByText("Second task")).toBeInTheDocument();
  });
});

describe("Column — pointer drag wiring", () => {
  const taskInColumn = {
    id: "task-1",
    column_id: "col-1",
    title: "Draggable task",
    assignee: null,
    created_at: "2026-08-20T00:00:00Z",
    updated_at: "2026-08-20T00:00:00Z",
  };

  // useBoardDnd resolves the drop target via
  // document.elementFromPoint(...).closest("[data-column-id]") — the marker
  // is the drop contract between the hook and the column.
  it("marks the section with data-column-id as the drop target", () => {
    const { container } = render(<Column column={leftmostColumn} />);
    expect(container.querySelector('section[data-column-id="col-1"]')).not.toBeNull();
  });

  it("highlights the column while it is the drop target", () => {
    const { container, rerender } = render(<Column column={leftmostColumn} />);
    const section = container.querySelector("section")!;
    expect(section.className).not.toContain("ring-1");

    rerender(<Column column={leftmostColumn} isDropTarget />);
    expect(section.className).toContain("ring-1");
  });

  it("starts a pointer drag on a card and reports it up", () => {
    const onTaskDragStart = vi.fn();
    render(
      <Column
        column={{ ...leftmostColumn, tasks: [taskInColumn] }}
        onTaskDragStart={onTaskDragStart}
      />,
    );

    const card = screen.getByText("Draggable task").closest('[data-slot="card"]')!;
    fireEvent.pointerDown(card, { clientX: 0, clientY: 0, pointerId: 1 });
    fireEvent.pointerMove(card, { clientX: 0, clientY: 40, pointerId: 1 });

    expect(onTaskDragStart).toHaveBeenCalledWith(taskInColumn);
  });

  // AC-10: the public viewer's cards must not be draggable even when drag
  // handlers are (incorrectly) passed in.
  it("does not wire drag on read-only cards", () => {
    const onTaskDragStart = vi.fn();
    render(
      <Column
        column={{ ...leftmostColumn, tasks: [taskInColumn] }}
        onTaskDragStart={onTaskDragStart}
        readOnly
      />,
    );

    const card = screen.getByText("Draggable task").closest('[data-slot="card"]')!;
    fireEvent.pointerDown(card, { clientX: 0, clientY: 0, pointerId: 1 });
    fireEvent.pointerMove(card, { clientX: 0, clientY: 40, pointerId: 1 });

    expect(onTaskDragStart).not.toHaveBeenCalled();
  });
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

    // Column renders whatever the caller passes and hands the created task up
    // via onTaskCreated (review root E) — the harness plays the caller's role
    // exactly as BoardPage does (append/refetch into `column.tasks`).
    function Caller() {
      const [tasks, setTasks] = useState<Task[]>([]);
      return (
        <Column
          column={{ ...leftmostColumn, tasks }}
          isLeftmost
          onTaskCreated={(task) => setTasks((prev) => [...prev, task])}
        />
      );
    }

    render(<Caller />);

    // Quick-add ховається за «+» у хедері колонки (scr02) — відкрити спершу.
    await user.click(screen.getByRole("button", { name: "Додати задачу" }));
    await user.type(screen.getByLabelText(/назва|title/i), "Write the report");
    await user.click(screen.getByRole("button", { name: "Додати" }));

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

    await user.click(screen.getByRole("button", { name: "Додати задачу" }));
    await user.click(screen.getByRole("button", { name: "Додати" }));

    await waitFor(() => {
      expect(screen.getByText(/назва обов'?язкова|title is required/i)).toBeInTheDocument();
    });
    expect(mockCreateTask).not.toHaveBeenCalled();
  });
});
