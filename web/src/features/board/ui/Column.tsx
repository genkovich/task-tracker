import { useState } from "react";
import { Plus } from "lucide-react";

import type { Column as ColumnState, Task } from "@/features/board/api/types";
import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/lib/utils";
import { QuickAddTask } from "@/features/board/ui/QuickAddTask";
import { TaskCard } from "@/features/board/ui/TaskCard";

export interface ColumnProps {
  column: ColumnState;
  /** Дошка колонки — quick-add створює задачу саме на ній (boards BRD-08).
   * Не потрібен у read-only рендері (public viewer), де quick-add не існує. */
  boardId?: string;
  /** AC-01: quick-add only renders in the leftmost column (`position === 0`). */
  isLeftmost?: boolean;
  onTaskClick?: (task: Task) => void;
  /** Id of the task currently being pointer-dragged (useBoardDnd state) —
   * lets the dragged card react to an external cancel (Escape). */
  dragTaskId?: string | null;
  /** This column is under the pointer during a drag — highlight it as the
   * drop target. */
  isDropTarget?: boolean;
  onTaskDragStart?: (task: Task) => void;
  onTaskDragMove?: (x: number, y: number) => void;
  onTaskDragEnd?: () => void;
  onTaskDragCancel?: () => void;
  /** Quick-add created a task — the caller owns the column data and decides
   * how to show it (append locally or refetch the board). */
  onTaskCreated?: (task: Task) => void;
  /** SCR-05 public viewer (AC-10): renders view-only — no quick-add and no
   * draggable cards, regardless of `isLeftmost`/handlers passed in. */
  readOnly?: boolean;
}

// Статусні кольори колонок за позицією (scr01): нейтральна → синя → зелена.
const COLUMN_ACCENTS = ["bg-status-todo", "bg-status-in-progress", "bg-status-done"];

/** One board column: dot + name + count header (SCR-01), the ordered task
 * list, and — in the leftmost column — the «+» that opens the inline
 * quick-add form (SCR-02). The `data-column-id` marker is the drop target:
 * useBoardDnd resolves it via `document.elementFromPoint(...)` during a
 * pointer drag — Pointer Events, not HTML5 dragover/drop, so touch works. */
export function Column({
  column,
  boardId,
  isLeftmost,
  onTaskClick,
  dragTaskId,
  isDropTarget,
  onTaskDragStart,
  onTaskDragMove,
  onTaskDragEnd,
  onTaskDragCancel,
  onTaskCreated,
  readOnly,
}: ColumnProps) {
  const [adding, setAdding] = useState(false);
  const accent = COLUMN_ACCENTS[column.position % COLUMN_ACCENTS.length];
  const canAdd = isLeftmost && !readOnly && boardId != null;

  return (
    <section
      data-column-id={column.id}
      className={cn(
        "flex w-full flex-col gap-3 rounded-2xl transition-colors sm:w-[340px] sm:shrink-0",
        // ring is a box-shadow — the drop highlight causes no layout shift.
        isDropTarget && "bg-white/[0.06] ring-1 ring-white/20",
      )}
    >
      <header className="flex h-8 items-center gap-2">
        <span aria-hidden className={cn("size-2 rounded-full", accent)} />
        <h2 className="text-[15px] font-semibold">{column.name}</h2>
        <span className="text-sm text-muted-foreground">{column.tasks.length}</span>
        {canAdd && (
          <Button
            variant="ghost"
            size="icon-sm"
            className="ml-auto rounded-full text-muted-foreground hover:text-foreground"
            aria-label="Додати задачу"
            onClick={() => setAdding((open) => !open)}
          >
            <Plus />
          </Button>
        )}
      </header>
      <div className="flex flex-col gap-3">
        {canAdd && boardId && adding && (
          <QuickAddTask
            boardId={boardId}
            onCreated={(task) => onTaskCreated?.(task)}
            onCancel={() => setAdding(false)}
          />
        )}
        {column.tasks.map((task) => (
          <TaskCard
            key={task.id}
            task={task}
            accentClass={accent}
            onClick={readOnly ? undefined : onTaskClick}
            draggable={!readOnly}
            dragging={dragTaskId != null && dragTaskId === task.id}
            onDragStart={readOnly ? undefined : onTaskDragStart}
            onDragMove={readOnly ? undefined : onTaskDragMove}
            onDragEnd={readOnly ? undefined : onTaskDragEnd}
            onDragCancel={readOnly ? undefined : onTaskDragCancel}
          />
        ))}
      </div>
    </section>
  );
}
