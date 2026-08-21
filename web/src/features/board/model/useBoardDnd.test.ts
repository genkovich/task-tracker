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

// Re-review 2026-08-21 re2 #3: the board must adopt fresh server columns on a
// plain rerender — the previous design remounted the whole subtree via
// key={version}, killing typed quick-add text and any active drag.
describe("useBoardDnd — syncs to fresh initialColumns without a remount (re2 #3)", () => {
  it("adopts new initialColumns when the hook rerenders with a fresh reference", () => {
    const moveTask = vi.fn();
    const { result, rerender } = renderHook(
      ({ columns }) => useBoardDnd(columns, { moveTask }),
      { initialProps: { columns: makeColumns() } },
    );

    const fresh = makeColumns();
    fresh[0].tasks.push({
      id: "task-2",
      column_id: "col-todo",
      title: "New from server",
      assignee: null,
      created_at: "2026-08-20T00:05:00Z",
      updated_at: "2026-08-20T00:05:00Z",
    });

    rerender({ columns: fresh });

    const todo = result.current.columns.find((c) => c.id === "col-todo")!;
    expect(todo.tasks.map((t) => t.id)).toEqual(["task-1", "task-2"]);
  });

  // Without the remount, a rollback from a failed move can land AFTER a
  // server sync already put the task back in its source column — re-adding
  // it must not duplicate the card.
  it("does not duplicate the task when a rollback lands after a server sync", async () => {
    let rejectMove!: (err: Error) => void;
    const moveTask = vi.fn().mockReturnValue(
      new Promise((_resolve, reject) => {
        rejectMove = reject;
      }),
    );

    const { result, rerender } = renderHook(
      ({ columns }) => useBoardDnd(columns, { moveTask }),
      { initialProps: { columns: makeColumns() } },
    );

    act(() => {
      result.current.handleDrop("task-1", "col-doing");
    });

    // A refetch snapshot lands mid-flight; the server never saw the failed
    // move, so its columns have task-1 back in col-todo.
    rerender({ columns: makeColumns() });

    act(() => {
      rejectMove(new Error("network error"));
    });

    // Wait for the rollback itself (it reports the error) before asserting,
    // otherwise the snapshot BEFORE the rejection handler satisfies waitFor.
    await waitFor(() => expect(mockShowApiError).toHaveBeenCalled());

    const todo = result.current.columns.find((c) => c.id === "col-todo")!;
    const doing = result.current.columns.find((c) => c.id === "col-doing")!;
    expect(todo.tasks.map((t) => t.id)).toEqual(["task-1"]);
    expect(doing.tasks.map((t) => t.id)).toEqual([]);
  });
});

// The pointer-drag orchestration: TaskCard reports start/move/end, the hook
// resolves the hovered column via document.elementFromPoint against
// [data-column-id] markers and hands the drop to the same optimistic
// handleDrop path.
describe("useBoardDnd — pointer drag orchestration", () => {
  function columnElement(id: string) {
    const el = document.createElement("section");
    el.setAttribute("data-column-id", id);
    return el;
  }

  it("tracks the hovered column for the drop-target highlight and drops on it", async () => {
    const moveTask = vi.fn().mockResolvedValue(undefined);
    const columns = makeColumns();
    const { result } = renderHook(() => useBoardDnd(columns, { moveTask }));

    act(() => {
      result.current.startDrag("task-1");
    });
    expect(result.current.drag).toEqual({ taskId: "task-1", overColumnId: null });

    const elementFromPoint = vi
      .spyOn(document, "elementFromPoint")
      .mockReturnValue(columnElement("col-doing"));
    act(() => {
      result.current.moveDrag(100, 200);
    });
    elementFromPoint.mockRestore();
    expect(result.current.drag).toEqual({ taskId: "task-1", overColumnId: "col-doing" });

    act(() => {
      result.current.endDrag();
    });

    // The drop went through the same optimistic path (AC-04).
    expect(result.current.drag).toBeNull();
    const doing = result.current.columns.find((c) => c.id === "col-doing")!;
    expect(doing.tasks.map((t) => t.id)).toEqual(["task-1"]);
    expect(moveTask).toHaveBeenCalledTimes(1);
    expect(moveTask).toHaveBeenCalledWith("task-1", "col-doing");
    await waitFor(() => expect(moveTask).toHaveResolved());
  });

  // AC-05: releasing outside any column is a no-op — no move, no API call.
  it("does nothing when released outside any column", () => {
    const moveTask = vi.fn();
    const columns = makeColumns();
    const { result } = renderHook(() => useBoardDnd(columns, { moveTask }));

    act(() => {
      result.current.startDrag("task-1");
    });
    const elementFromPoint = vi.spyOn(document, "elementFromPoint").mockReturnValue(null);
    act(() => {
      result.current.moveDrag(5, 5);
    });
    elementFromPoint.mockRestore();
    act(() => {
      result.current.endDrag();
    });

    expect(result.current.drag).toBeNull();
    const todo = result.current.columns.find((c) => c.id === "col-todo")!;
    expect(todo.tasks.map((t) => t.id)).toEqual(["task-1"]);
    expect(moveTask).not.toHaveBeenCalled();
  });

  it("cancels the drag on Escape — releasing afterwards moves nothing", () => {
    const moveTask = vi.fn();
    const columns = makeColumns();
    const { result } = renderHook(() => useBoardDnd(columns, { moveTask }));

    act(() => {
      result.current.startDrag("task-1");
    });
    const elementFromPoint = vi
      .spyOn(document, "elementFromPoint")
      .mockReturnValue(columnElement("col-doing"));
    act(() => {
      result.current.moveDrag(100, 200);
    });
    elementFromPoint.mockRestore();

    act(() => {
      window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    });
    expect(result.current.drag).toBeNull();

    // The pointer is released after the abort — endDrag must be a no-op.
    act(() => {
      result.current.endDrag();
    });
    const todo = result.current.columns.find((c) => c.id === "col-todo")!;
    expect(todo.tasks.map((t) => t.id)).toEqual(["task-1"]);
    expect(moveTask).not.toHaveBeenCalled();
  });

  it("cancels the drag on pointercancel via cancelDrag", () => {
    const moveTask = vi.fn();
    const columns = makeColumns();
    const { result } = renderHook(() => useBoardDnd(columns, { moveTask }));

    act(() => {
      result.current.startDrag("task-1");
    });
    act(() => {
      result.current.cancelDrag();
    });

    expect(result.current.drag).toBeNull();
    expect(moveTask).not.toHaveBeenCalled();
  });
});

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

    // The hook's contract: `initialColumns` is referentially stable between
    // renders (a new reference means fresh server state to adopt).
    const columns = makeColumns();
    const { result } = renderHook(() => useBoardDnd(columns, { moveTask }));

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

    const columns = makeColumns();
    const { result } = renderHook(() => useBoardDnd(columns, { moveTask }));

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

    const columns = makeColumns();
    const { result } = renderHook(() => useBoardDnd(columns, { moveTask }));

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
