import { useCallback, useState } from "react";
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
 */
export function useBoardDnd(initialColumns: Column[], { moveTask }: UseBoardDndDeps) {
  const [columns, setColumns] = useState<Column[]>(initialColumns);

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
        setColumns((prev) =>
          prev.map((column) => {
            if (column.id === targetColumnId) {
              return { ...column, tasks: column.tasks.filter((t) => t.id !== taskId) };
            }
            if (column.id === fromColumnId) {
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
