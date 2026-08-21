import { useCallback, useEffect, useRef, useState } from "react";
import { showApiError } from "@/shared/lib/showApiError";
import type { Column, Task } from "../api/types";

interface UseBoardDndDeps {
  moveTask: (taskId: string, columnId: string) => Promise<Task>;
}

/** An in-flight pointer drag: which task is being dragged and which column
 * is currently under the pointer (the drop-target highlight). */
export interface BoardDrag {
  taskId: string;
  overColumnId: string | null;
}

/**
 * Drag-and-drop board state with optimistic task moves.
 *
 * The drag gesture itself runs on Pointer Events (TaskCard captures the
 * pointer and reports start/move/end here) — HTML5 drag events never fire
 * for touch gestures, so the board was mouse-only. The hover target is
 * resolved via `document.elementFromPoint` against `[data-column-id]`
 * markers, since a captured pointer keeps delivering events to the card
 * element even once the finger/cursor has moved off it.
 *
 * - Dropping on a valid column (AC-04): moves the task locally immediately,
 *   then calls `moveTask`; rolls back the local move if the call fails.
 * - Dropping outside any known column (AC-05): no-op, no API call.
 * - Escape (or a pointercancel) aborts the drag without a move.
 * - A new `initialColumns` reference (fresh server state after a refetch)
 *   replaces the local columns in place — no remount required, so typed
 *   quick-add text and an active drag survive board broadcasts.
 *
 * `initialColumns` must be referentially stable between renders (e.g. held
 * in the caller's state); a new reference is read as fresh server state.
 */
export function useBoardDnd(initialColumns: Column[], { moveTask }: UseBoardDndDeps) {
  const [columns, setColumns] = useState<Column[]>(initialColumns);
  const [drag, setDrag] = useState<BoardDrag | null>(null);

  // Render-phase adoption of fresh server state (the React "derive state
  // from props" pattern) — server truth wins over any local optimistic move.
  const prevInitialRef = useRef(initialColumns);
  if (prevInitialRef.current !== initialColumns) {
    prevInitialRef.current = initialColumns;
    setColumns(initialColumns);
  }

  const handleDrop = useCallback(
    (taskId: string, targetColumnId: string | null) => {
      if (targetColumnId === null) {
        return;
      }

      let sourceColumnId: string | null = null;
      let movedTask: Task | null = null;

      for (const column of columns) {
        const found = column.tasks.find((t) => t.id === taskId);
        if (found) {
          sourceColumnId = column.id;
          movedTask = found;
          break;
        }
      }

      if (!movedTask || sourceColumnId === null) {
        return;
      }
      if (sourceColumnId === targetColumnId) {
        return;
      }
      if (!columns.some((c) => c.id === targetColumnId)) {
        return;
      }

      const taskToMove = movedTask;
      const fromColumnId = sourceColumnId;

      setColumns((prev) =>
        prev.map((column) => {
          if (column.id === fromColumnId) {
            return { ...column, tasks: column.tasks.filter((t) => t.id !== taskId) };
          }
          if (column.id === targetColumnId) {
            return {
              ...column,
              tasks: [...column.tasks, { ...taskToMove, column_id: targetColumnId }],
            };
          }
          return column;
        }),
      );

      moveTask(taskId, targetColumnId).catch((err: unknown) => {
        showApiError(err);
        // The rollback may land after a server sync already restored the
        // task to its source column — re-adding must stay idempotent.
        setColumns((prev) =>
          prev.map((column) => {
            if (column.id === targetColumnId) {
              return { ...column, tasks: column.tasks.filter((t) => t.id !== taskId) };
            }
            if (column.id === fromColumnId && !column.tasks.some((t) => t.id === taskId)) {
              return { ...column, tasks: [...column.tasks, taskToMove] };
            }
            return column;
          }),
        );
      });
    },
    [columns, moveTask],
  );

  const startDrag = useCallback((taskId: string) => {
    setDrag({ taskId, overColumnId: null });
  }, []);

  const moveDrag = useCallback((x: number, y: number) => {
    const columnEl = document.elementFromPoint(x, y)?.closest("[data-column-id]");
    const overColumnId = columnEl?.getAttribute("data-column-id") ?? null;
    setDrag((prev) =>
      prev === null || prev.overColumnId === overColumnId ? prev : { ...prev, overColumnId },
    );
  }, []);

  const endDrag = useCallback(() => {
    if (drag === null) return;
    setDrag(null);
    handleDrop(drag.taskId, drag.overColumnId);
  }, [drag, handleDrop]);

  const cancelDrag = useCallback(() => {
    setDrag(null);
  }, []);

  // Escape aborts the drag; TaskCard sees its `dragging` prop flip false and
  // resets the card visuals, ignoring pointer moves until release.
  const dragActive = drag !== null;
  useEffect(() => {
    if (!dragActive) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") cancelDrag();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [dragActive, cancelDrag]);

  return { columns, handleDrop, drag, startDrag, moveDrag, endDrag, cancelDrag };
}
