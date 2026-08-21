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
  // Read-only rendering (SCR-05, AC-10): without a click handler the card is
  // plain content — no button semantics, focusability or pointer cursor; and
  // when not draggable, no drag wiring either.
  const clickable = onClick !== undefined;

  return (
    <Card
      className={cn(clickable && "cursor-pointer", "gap-2 py-3", className)}
      role={clickable ? "button" : undefined}
      tabIndex={clickable ? 0 : undefined}
      draggable={draggable}
      onClick={clickable ? () => onClick(task) : undefined}
      onDragStart={
        draggable
          ? (e) => {
              e.dataTransfer.setData("text/plain", task.id);
              onDragStart?.(task);
            }
          : undefined
      }
    >
      <CardContent className="flex flex-col gap-1 px-3">
        <p className="text-sm font-medium">{task.title}</p>
        {task.assignee && <p className="text-muted-foreground text-xs">{task.assignee}</p>}
      </CardContent>
    </Card>
  );
}
