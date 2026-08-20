import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EditTaskModal } from "./EditTaskModal";
import type { Task } from "@/features/board/api/types";

// api/model already exist (T13) — mocked at the module boundary the component
// is expected to depend on (boardApi.editTask / boardApi.deleteTask, per
// docs/features/board/tasks/T16-ui-edit-modal.md "What").
const mockEditTask = vi.fn();
const mockDeleteTask = vi.fn();

vi.mock("@/features/board/api/boardApi", () => ({
  boardApi: {
    editTask: (...args: unknown[]) => mockEditTask(...args),
    deleteTask: (...args: unknown[]) => mockDeleteTask(...args),
  },
}));

const existingTask: Task = {
  id: "task-1",
  column_id: "col-1",
  title: "Original title",
  assignee: "Alice",
  created_at: "2026-08-20T00:00:00Z",
  updated_at: "2026-08-20T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("EditTaskModal", () => {
  // AC-03 (US-02) happy path — spec.md §5: "team member змінює її назву або
  // виконавця і зберігає" -> "система записує нові значення й одразу показує
  // їх на board". DoD: "changing title/assignee and saving updates the card's
  // displayed values without a full reload".
  it("editing title and assignee and saving updates the card immediately", async () => {
    const user = userEvent.setup();
    const updated: Task = { ...existingTask, title: "New title", assignee: "Bob" };
    mockEditTask.mockResolvedValue(updated);
    const onSaved = vi.fn();

    render(
      <EditTaskModal task={existingTask} open onOpenChange={() => {}} onSaved={onSaved} onDeleted={() => {}} />,
    );

    const titleInput = screen.getByLabelText(/назва|title/i);
    const assigneeInput = screen.getByLabelText(/виконавець|assignee/i);

    await user.clear(titleInput);
    await user.type(titleInput, "New title");
    await user.clear(assigneeInput);
    await user.type(assigneeInput, "Bob");

    await user.click(screen.getByRole("button", { name: /зберегти|save/i }));

    await waitFor(() => {
      expect(mockEditTask).toHaveBeenCalledWith("task-1", {
        title: "New title",
        assignee: "Bob",
      });
    });

    // Observable outcome the AC names: the board/card reflects the new
    // values immediately, without a full reload — the modal reports the
    // saved task back to its caller so the card can update in place.
    await waitFor(() => {
      expect(onSaved).toHaveBeenCalledWith(updated);
    });
  });

  it("saving with an empty title shows the inline error and does not call the API", async () => {
    const user = userEvent.setup();

    render(
      <EditTaskModal task={existingTask} open onOpenChange={() => {}} onSaved={() => {}} onDeleted={() => {}} />,
    );

    await user.clear(screen.getByLabelText(/назва|title/i));
    await user.click(screen.getByRole("button", { name: /зберегти|save/i }));

    await waitFor(() => {
      expect(screen.getByText(/назва обов'?язкова|title is required/i)).toBeInTheDocument();
    });
    expect(mockEditTask).not.toHaveBeenCalled();
  });

  // AC-06 (US-04) happy path — spec.md §5: "team member видаляє цю task" ->
  // "система прибирає task з board, і вона більше не показується нікому".
  // DoD: "clicking delete removes the task from the board".
  it("clicking delete removes the task from the board", async () => {
    const user = userEvent.setup();
    mockDeleteTask.mockResolvedValue(undefined);
    const onDeleted = vi.fn();

    render(
      <EditTaskModal task={existingTask} open onOpenChange={() => {}} onSaved={() => {}} onDeleted={onDeleted} />,
    );

    await user.click(screen.getByRole("button", { name: /видалити|delete/i }));

    await waitFor(() => {
      expect(mockDeleteTask).toHaveBeenCalledWith("task-1");
    });

    // Observable outcome the AC names: the task is gone from the board —
    // the modal reports the deletion so the caller can remove the card.
    await waitFor(() => {
      expect(onDeleted).toHaveBeenCalledWith("task-1");
    });
  });
});
