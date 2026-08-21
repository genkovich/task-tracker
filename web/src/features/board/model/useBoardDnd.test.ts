import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useBoardDnd } from "./useBoardDnd";
import type { Column } from "../api/types";

// Review 2026-08-21 root I: a failed move must not be silent — the user sees
// the card snap back AND gets told why.
const mockShowApiError = vi.fn();
vi.mock("@/shared/lib/showApiError", () => ({
  showApiError: (...args: unknown[]) => mockShowApiError(...args),
}));

beforeEach(() => {
  vi.clearAllMocks();
});

function makeColumns(): Column[] {
  return [
    {
      id: "col-todo",
      name: "To do",
      position: 0,
      tasks: [
        {
          id: "task-1",
          column_id: "col-todo",
          title: "Test task",
          assignee: null,
          created_at: "2026-08-20T00:00:00Z",
          updated_at: "2026-08-20T00:00:00Z",
        },
      ],
    },
    {
      id: "col-doing",
      name: "Doing",
      position: 1,
      tasks: [],
    },
  ];
}

describe("useBoardDnd", () => {
  // AC-04: dropping a task on a valid column optimistically moves it locally
  // and calls the move API exactly once.
  it("moves the task into the target column immediately and calls moveTask once", async () => {
    const moveTask = vi.fn().mockResolvedValue({
      id: "task-1",
      column_id: "col-doing",
      title: "Test task",
      assignee: null,
      created_at: "2026-08-20T00:00:00Z",
      updated_at: "2026-08-20T00:03:00Z",
    });

    const { result } = renderHook(() => useBoardDnd(makeColumns(), { moveTask }));

    act(() => {
      result.current.handleDrop("task-1", "col-doing");
    });

    // Optimistic: local state already reflects the move before the API call resolves.
    const todo = result.current.columns.find((c) => c.id === "col-todo")!;
    const doing = result.current.columns.find((c) => c.id === "col-doing")!;
    expect(todo.tasks.map((t) => t.id)).toEqual([]);
    expect(doing.tasks.map((t) => t.id)).toEqual(["task-1"]);

    expect(moveTask).toHaveBeenCalledTimes(1);
    expect(moveTask).toHaveBeenCalledWith("task-1", "col-doing");

    await waitFor(() => expect(moveTask).toHaveResolved());
  });

  // AC-05: dropping outside any valid column is a no-op: no local state
  // change, no API call.
  it("leaves the task in place and calls no API when dropped outside any column", () => {
    const moveTask = vi.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() => useBoardDnd(makeColumns(), { moveTask }));

    act(() => {
      result.current.handleDrop("task-1", null);
    });

    const todo = result.current.columns.find((c) => c.id === "col-todo")!;
    expect(todo.tasks.map((t) => t.id)).toEqual(["task-1"]);
    expect(moveTask).not.toHaveBeenCalled();
  });

  // A failed moveTask call rolls the optimistic move back to the original column.
  it("rolls back the optimistic move when moveTask fails", async () => {
    const moveTask = vi.fn().mockRejectedValue(new Error("network error"));

    const { result } = renderHook(() => useBoardDnd(makeColumns(), { moveTask }));

    act(() => {
      result.current.handleDrop("task-1", "col-doing");
    });

    // Immediately after the drop, the optimistic move is visible.
    expect(result.current.columns.find((c) => c.id === "col-doing")!.tasks.map((t) => t.id)).toEqual([
      "task-1",
    ]);

    await waitFor(() => {
      const todo = result.current.columns.find((c) => c.id === "col-todo")!;
      const doing = result.current.columns.find((c) => c.id === "col-doing")!;
      expect(todo.tasks.map((t) => t.id)).toEqual(["task-1"]);
      expect(doing.tasks.map((t) => t.id)).toEqual([]);
    });

    // The rollback is visible feedback, but the failure itself must be
    // surfaced too, not swallowed (review root I).
    expect(mockShowApiError).toHaveBeenCalledTimes(1);
  });
});
