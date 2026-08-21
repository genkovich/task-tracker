import { useCallback, useEffect, useRef, useState } from "react";
import { useParams } from "react-router";

import type { Route } from "./+types/BoardPage";
import { boardApi } from "@/features/board/api/boardApi";
import { useBoardEvents } from "@/features/board/api/useBoardEvents";
import { useBoardDnd } from "@/features/board/model/useBoardDnd";
import { showApiError } from "@/shared/lib/showApiError";
import type { BoardState, Task, TaskRecord } from "@/features/board/api/types";
import { BoardLoadError, BoardLoading } from "@/features/board/ui/BoardLoadState";
import { Column } from "@/features/board/ui/Column";
import { TaskDetailsModal } from "@/features/board/ui/TaskDetailsModal";
import { PublicLinkPanel } from "@/features/public-link/ui/PublicLinkPanel";
import { BoardShell } from "@/widgets/board-shell/ui/BoardShell";

export const meta: Route.MetaFunction = () => [{ title: "Дошка — Task Tracker" }];

// ProtectedLayout reads this via useMatches() to drop its default centered
// max-w-5xl column — the three-column board needs the full frame width.
export const handle = { fullWidth: true };

/** SCR-01 team-editor board, параметризований дошкою (boards BRD-04): роут
 * /board/:boardId. Компонує колонки/картки/quick-add, модалку редагування і
 * public-link панель цієї дошки. No fetch/mutation logic of its own — all
 * data access goes through boardApi / useBoardEvents / useBoardDnd. */
export default function BoardPage() {
  const { boardId } = useParams<{ boardId: string }>();
  const [board, setBoard] = useState<BoardState | null>(null);
  const [failed, setFailed] = useState(false);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const boardRef = useRef<BoardState | null>(null);

  const refetch = useCallback(() => {
    if (!boardId) return;
    boardApi
      .getBoard(boardId)
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
  }, [boardId]);

  useEffect(() => {
    // Зміна :boardId — це інша дошка: скинути стан перед першим фетчем, щоб
    // глядач не бачив колонок попередньої дошки.
    boardRef.current = null;
    setBoard(null);
    setFailed(false);
    refetch();
  }, [refetch]);

  useBoardEvents(boardId ?? "", refetch);

  return (
    <BoardShell
      actions={
        boardId && <PublicLinkPanel boardId={boardId} publicLink={board?.public_link ?? null} />
      }
    >
      {failed ? (
        <BoardLoadError onRetry={refetch} />
      ) : !board || !boardId ? (
        <BoardLoading />
      ) : (
        <BoardColumns
          boardId={boardId}
          columns={board.columns}
          onTaskClick={(task) => setSelectedTask(task)}
          onTaskCreated={refetch}
        />
      )}

      {selectedTask && (
        <TaskDetailsModal
          task={selectedTask}
          open
          onOpenChange={(open) => {
            if (!open) setSelectedTask(null);
          }}
          // A comment lands while the dialog stays open, so onSaved only
          // refreshes the board behind it — closing is the caller's job in
          // the save/delete paths, which do it themselves.
          onSaved={refetch}
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
  boardId: string;
  columns: BoardState["columns"];
  onTaskClick: (task: Task) => void;
  onTaskCreated: (task: TaskRecord) => void;
}

/** Owns the drag-and-drop wiring (T14's `useBoardDnd`) for the currently
 * loaded columns; `useBoardDnd` adopts each fresh `initialColumns`
 * reference in place, so a refetch is a plain rerender — never a remount
 * that would wipe typed quick-add text or an active drag. */
function BoardColumns({
  boardId,
  columns: initialColumns,
  onTaskClick,
  onTaskCreated,
}: BoardColumnsProps) {
  const { columns, drag, startDrag, moveDrag, endDrag, cancelDrag } = useBoardDnd(initialColumns, {
    moveTask: boardApi.moveTask,
  });

  return (
    <div className="flex-1 rounded-3xl sm:bg-muted/50 sm:p-5">
      <div className="flex flex-col gap-8 sm:flex-row sm:gap-5 sm:overflow-x-auto sm:pb-1">
        {columns.map((column) => (
          <Column
            key={column.id}
            column={column}
            boardId={boardId}
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
