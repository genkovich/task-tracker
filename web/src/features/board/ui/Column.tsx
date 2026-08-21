import { useState } from "react";
import { Plus } from "lucide-react";

import type { Column as ColumnState, Task } from "@/features/board/api/types";
import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/lib/utils";
import { QuickAddTask } from "@/features/board/ui/QuickAddTask";
import { TaskCard } from "@/features/board/ui/TaskCard";

export interface ColumnProps {
  column: ColumnState;
  /** AC-01: quick-add only renders in the leftmost column (`position === 0`). */
  isLeftmost?: boolean;
  onTaskClick?: (task: Task) => void;
  onTaskDragStart?: (task: Task) => void;
  onDropTask?: (taskId: string) => void;
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
 * quick-add form (SCR-02). */
export function Column({
  column,
  isLeftmost,
  onTaskClick,
  onTaskDragStart,
  onDropTask,
  onTaskCreated,
  readOnly,
}: ColumnProps) {
  const [adding, setAdding] = useState(false);
  const accent = COLUMN_ACCENTS[column.position % COLUMN_ACCENTS.length];
  const canAdd = isLeftmost && !readOnly;

  return (
    <section
      className="flex w-full flex-col gap-3 sm:w-[340px] sm:shrink-0"
      onDragOver={!readOnly && onDropTask ? (e) => e.preventDefault() : undefined}
      onDrop={
        !readOnly && onDropTask
          ? (e) => {
              e.preventDefault();
              const taskId = e.dataTransfer.getData("text/plain");
              if (taskId) onDropTask(taskId);
            }
          : undefined
      }
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
        {canAdd && adding && (
          <QuickAddTask
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
            onDragStart={readOnly ? undefined : onTaskDragStart}
            draggable={!readOnly}
          />
        ))}
      </div>
    </section>
  );
}
