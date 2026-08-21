import { useCallback, useRef, useState } from "react";
import { showApiError } from "@/shared/lib/showApiError";
import type { Column, Task } from "../api/types";

interface UseBoardDndDeps {
  moveTask: (taskId: string, columnId: string) => Promise<Task>;
}

/**
 * Drag-and-drop board state with optimistic task moves.
 *
 * - Dropping on a valid column (AC-04): moves the task locally immediately,
 *   then calls `moveTask`; rolls back the local move if the call fails.
 * - Dropping outside any known column (AC-05): no-op, no API call.
 * - A new `initialColumns` reference (fresh server state after a refetch)
 *   replaces the local columns in place — no remount required, so typed
 *   quick-add text and an active drag survive board broadcasts.
 *
 * `initialColumns` must be referentially stable between renders (e.g. held
 * in the caller's state); a new reference is read as fresh server state.
 */
export function useBoardDnd(initialColumns: Column[], { moveTask }: UseBoardDndDeps) {
  const [columns, setColumns] = useState<Column[]>(initialColumns);

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

  return { columns, handleDrop };
}
