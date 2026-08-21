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
import { BoardShell } from "@/widgets/board-shell/ui/BoardShell";
import { BoardUserBadge } from "@/widgets/board-shell/ui/BoardUserBadge";

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

  return (
    <BoardShell
      actions={
        <>
          <PublicLinkPanel publicLink={board?.public_link ?? null} />
          <BoardUserBadge />
        </>
      }
    >
      {failed ? (
        <BoardLoadError onRetry={refetch} />
      ) : !board ? (
        <BoardLoading />
      ) : (
        <BoardColumns
          columns={board.columns}
          onTaskClick={(task) => setSelectedTask(task)}
          onTaskCreated={refetch}
        />
      )}

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
    </BoardShell>
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
  const { columns, drag, startDrag, moveDrag, endDrag, cancelDrag } = useBoardDnd(initialColumns, {
    moveTask: boardApi.moveTask,
  });

  return (
    <div className="flex-1 rounded-3xl sm:bg-white/[0.04] sm:p-5">
      <div className="flex flex-col gap-8 sm:flex-row sm:items-start sm:gap-5 sm:overflow-x-auto sm:pb-1">
        {columns.map((column) => (
          <Column
            key={column.id}
            column={column}
            isLeftmost={column.position === 0}
            onTaskClick={onTaskClick}
            dragTaskId={drag?.taskId ?? null}
            isDropTarget={drag !== null && drag.overColumnId === column.id}
            onTaskDragStart={(task) => startDrag(task.id)}
            onTaskDragMove={moveDrag}
            onTaskDragEnd={endDrag}
            onTaskDragCancel={cancelDrag}
            onTaskCreated={onTaskCreated}
          />
        ))}
      </div>
    </div>
  );
}
