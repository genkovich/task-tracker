import { useState } from "react";

import type { Column as ColumnState, Task } from "@/features/board/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { QuickAddTask } from "@/features/board/ui/QuickAddTask";
import { TaskCard } from "@/features/board/ui/TaskCard";

export interface ColumnProps {
  column: ColumnState;
  /** AC-01: quick-add only renders in the leftmost column (`position === 0`). */
  isLeftmost?: boolean;
  onTaskClick?: (task: Task) => void;
  onTaskDragStart?: (task: Task) => void;
  onDropTask?: (taskId: string) => void;
  /** SCR-05 public viewer (AC-10): renders view-only — no quick-add and no
   * draggable cards, regardless of `isLeftmost`/handlers passed in. */
  readOnly?: boolean;
}

/** One board column: name + its ordered task list (SCR-01), plus the
 * leftmost column's inline quick-add form (SCR-02). */
export function Column({
  column,
  isLeftmost,
  onTaskClick,
  onTaskDragStart,
  onDropTask,
  readOnly,
}: ColumnProps) {
  const [tasks, setTasks] = useState<Task[]>(column.tasks);

  return (
    <Card
      className="w-72 shrink-0 gap-3 py-4"
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
      <CardHeader className="px-4">
        <CardTitle>{column.name}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 px-4">
        {tasks.map((task) => (
          <TaskCard
            key={task.id}
            task={task}
            onClick={readOnly ? undefined : onTaskClick}
            onDragStart={readOnly ? undefined : onTaskDragStart}
            draggable={!readOnly}
          />
        ))}
        {isLeftmost && !readOnly && (
          <QuickAddTask onCreated={(task) => setTasks((prev) => [...prev, task])} />
        )}
      </CardContent>
    </Card>
  );
}
