import type { Task } from "@/features/board/api/types";
import { Card, CardContent } from "@/shared/ui/card";
import { getNameInitials } from "@/shared/lib/user";
import { cn } from "@/shared/lib/utils";

export interface TaskCardProps {
  task: Task;
  onClick?: (task: Task) => void;
  draggable?: boolean;
  onDragStart?: (task: Task) => void;
  /** Клас кольорової смужки зверху картки — статусний колір колонки (scr01). */
  accentClass?: string;
  className?: string;
}

/** A single task, shown inside a `Column`. Draggable so `Column` (via T14's
 * `useBoardDnd`, wired at the page level in T18) can move it between columns. */
export function TaskCard({
  task,
  onClick,
  draggable = true,
  onDragStart,
  accentClass,
  className,
}: TaskCardProps) {
  // Read-only rendering (SCR-05, AC-10): without a click handler the card is
  // plain content — no button semantics, focusability or pointer cursor; and
  // when not draggable, no drag wiring either.
  const clickable = onClick !== undefined;

  return (
    <Card
      className={cn(
        "gap-0 overflow-hidden rounded-2xl border-white/10 bg-background p-0 shadow-none transition-colors",
        clickable && "cursor-pointer hover:bg-white/[0.04]",
        className,
      )}
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
      <div aria-hidden className={cn("h-1 shrink-0", accentClass ?? "bg-white/15")} />
      <CardContent className="flex min-h-[76px] flex-col justify-between gap-3 p-4 pt-3">
        <p className="text-[15px] leading-snug font-medium">{task.title}</p>
        <span className="flex justify-end">
          {task.assignee && (
            <span
              title={task.assignee}
              className="flex size-7 items-center justify-center rounded-full bg-white/10 text-[11px] font-semibold"
            >
              {getNameInitials(task.assignee)}
            </span>
          )}
        </span>
      </CardContent>
    </Card>
  );
}
