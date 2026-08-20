import type { Task } from "@/features/board/api/types";
import { Card, CardContent } from "@/shared/ui/card";
import { cn } from "@/shared/lib/utils";

export interface TaskCardProps {
  task: Task;
  onClick?: (task: Task) => void;
  draggable?: boolean;
  onDragStart?: (task: Task) => void;
  className?: string;
}

/** A single task, shown inside a `Column`. Draggable so `Column` (via T14's
 * `useBoardDnd`, wired at the page level in T18) can move it between columns. */
export function TaskCard({
  task,
  onClick,
  draggable = true,
  onDragStart,
  className,
}: TaskCardProps) {
  return (
    <Card
      className={cn("cursor-pointer gap-2 py-3", className)}
      role="button"
      tabIndex={0}
      draggable={draggable}
      onClick={() => onClick?.(task)}
      onDragStart={(e) => {
        e.dataTransfer.setData("text/plain", task.id);
        onDragStart?.(task);
      }}
    >
      <CardContent className="flex flex-col gap-1 px-3">
        <p className="text-sm font-medium">{task.title}</p>
        {task.assignee && <p className="text-muted-foreground text-xs">{task.assignee}</p>}
      </CardContent>
    </Card>
  );
}
