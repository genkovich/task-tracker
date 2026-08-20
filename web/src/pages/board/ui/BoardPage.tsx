import { useCallback, useEffect, useState } from "react";

import type { Route } from "./+types/BoardPage";
import { boardApi } from "@/features/board/api/boardApi";
import { useBoardEvents } from "@/features/board/api/useBoardEvents";
import { useBoardDnd } from "@/features/board/model/useBoardDnd";
import type { BoardState, Task } from "@/features/board/api/types";
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
  const [version, setVersion] = useState(0);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);

  const refetch = useCallback(() => {
    boardApi.getBoard().then((state) => {
      setBoard(state);
      setVersion((v) => v + 1);
    });
  }, []);

  useEffect(() => {
    refetch();
  }, [refetch]);

  useBoardEvents(refetch);

  if (!board) return null;

  return (
    <main className="flex h-screen flex-col gap-4 p-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Дошка команди</h1>
        <PublicLinkPanel />
      </div>

      <BoardColumns
        key={version}
        columns={board.columns}
        onTaskClick={(task) => setSelectedTask(task)}
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
}

/** Owns the drag-and-drop wiring (T14's `useBoardDnd`) for the currently
 * loaded columns; remounted (via the parent's `key`) whenever the board is
 * refetched so it always starts from the freshest server state. */
function BoardColumns({ columns: initialColumns, onTaskClick }: BoardColumnsProps) {
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
        />
      ))}
    </div>
  );
}
