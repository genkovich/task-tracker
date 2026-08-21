import { useCallback, useEffect, useRef, useState } from "react";

import type { Route } from "./+types/BoardPage";
import { boardApi } from "@/features/board/api/boardApi";
import { useBoardEvents } from "@/features/board/api/useBoardEvents";
import { useBoardDnd } from "@/features/board/model/useBoardDnd";
import { showApiError } from "@/shared/lib/showApiError";
import type { BoardState, Task } from "@/features/board/api/types";
import { BoardLoadError, BoardLoading } from "@/features/board/ui/BoardLoadState";
import { Column } from "@/features/board/ui/Column";
import { EditTaskModal } from "@/features/board/ui/EditTaskModal";
import { PublicLinkPanel } from "@/features/public-link/ui/PublicLinkPanel";

export const meta: Route.MetaFunction = () => [{ title: "Дошка — Task Tracker" }];

/** SCR-01 team-editor board: composes T15's columns/cards/quick-add, wires
 * T16's edit modal to a card click, and mounts T17's public-link panel
 * (SCR-01 -> SCR-04). No fetch/mutation logic of its own — all data access
 * goes through boardApi / useBoardEvents / useBoardDnd. */
export default function BoardPage() {
  const [board, setBoard] = useState<BoardState | null>(null);
  const [failed, setFailed] = useState(false);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const boardRef = useRef<BoardState | null>(null);

  const refetch = useCallback(() => {
    boardApi
      .getBoard()
      .then((state) => {
        boardRef.current = state;
        setBoard(state);
        setFailed(false);
      })
      .catch((err: unknown) => {
        // The full error screen is for the initial load only; a failed
        // refetch keeps the stale board on screen and surfaces a toast.
        if (boardRef.current === null) {
          setFailed(true);
          return;
        }
        showApiError(err);
      });
  }, []);

  useEffect(() => {
    refetch();
  }, [refetch]);

  useBoardEvents(refetch);

  if (failed) {
    return (
      <main className="flex h-screen flex-col p-4">
        <BoardLoadError onRetry={refetch} />
      </main>
    );
  }

  if (!board) {
    return (
      <main className="flex h-screen flex-col p-4">
        <BoardLoading />
      </main>
    );
  }

  return (
    <main className="flex h-screen flex-col gap-4 p-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Дошка команди</h1>
        <PublicLinkPanel publicLink={board.public_link} />
      </div>

      <BoardColumns
        columns={board.columns}
        onTaskClick={(task) => setSelectedTask(task)}
        onTaskCreated={refetch}
      />

      {selectedTask && (
        <EditTaskModal
          task={selectedTask}
          open
          onOpenChange={(open) => {
            if (!open) setSelectedTask(null);
          }}
          onSaved={() => {
            setSelectedTask(null);
            refetch();
          }}
          onDeleted={() => {
            setSelectedTask(null);
            refetch();
          }}
        />
      )}
    </main>
  );
}

interface BoardColumnsProps {
  columns: BoardState["columns"];
  onTaskClick: (task: Task) => void;
  onTaskCreated: (task: Task) => void;
}

/** Owns the drag-and-drop wiring (T14's `useBoardDnd`) for the currently
 * loaded columns; `useBoardDnd` adopts each fresh `initialColumns`
 * reference in place, so a refetch is a plain rerender — never a remount
 * that would wipe typed quick-add text or an active drag. */
function BoardColumns({ columns: initialColumns, onTaskClick, onTaskCreated }: BoardColumnsProps) {
  const { columns, handleDrop } = useBoardDnd(initialColumns, { moveTask: boardApi.moveTask });

  return (
    <div className="flex flex-1 gap-4 overflow-x-auto">
      {columns.map((column) => (
        <Column
          key={column.id}
          column={column}
          isLeftmost={column.position === 0}
          onTaskClick={onTaskClick}
          onDropTask={(taskId) => handleDrop(taskId, column.id)}
          onTaskCreated={onTaskCreated}
        />
      ))}
    </div>
  );
}
