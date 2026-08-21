import { useEffect, useRef, useState } from "react";
import type { Ref } from "react";
import type { Task } from "@/features/board/api/types";
import { Card, CardContent } from "@/shared/ui/card";
import { getNameInitials } from "@/shared/lib/user";
import { cn } from "@/shared/lib/utils";
import { DragGhost } from "@/features/board/ui/DragGhost";

export interface TaskCardProps {
  task: Task;
  onClick?: (task: Task) => void;
  draggable?: boolean;
  /** This card is the actively dragged one (useBoardDnd state). A flip back
   * to `false` while the pointer is still down means an external cancel
   * (Escape) — the card drops its drag visuals and ignores further moves. */
  dragging?: boolean;
  onDragStart?: (task: Task) => void;
  onDragMove?: (x: number, y: number) => void;
  onDragEnd?: () => void;
  onDragCancel?: () => void;
  /** Клас кольорової смужки зверху картки — статусний колір колонки (scr01). */
  accentClass?: string;
  className?: string;
}

// Movement past this many pixels counts as a drag, not a tap/click — below
// it, releasing opens the edit dialog instead.
const DRAG_THRESHOLD_PX = 6;

interface TaskCardVisualProps extends React.ComponentProps<"div"> {
  task: Task;
  accentClass?: string;
  ref?: Ref<HTMLDivElement>;
}

/** The card's visual content only — accent stripe + title/assignee, no
 * interaction wiring. Shared by the real (interactive) card and its
 * DragGhost clone so the two never drift apart. */
function TaskCardVisual({ task, accentClass, className, ref, ...props }: TaskCardVisualProps) {
  return (
    <Card
      ref={ref}
      className={cn("gap-0 overflow-hidden rounded-2xl border bg-background p-0 shadow-none", className)}
      {...props}
    >
      <div aria-hidden className={cn("h-1 shrink-0", accentClass ?? "bg-muted")} />
      <CardContent className="flex min-h-[76px] flex-col justify-between gap-3 p-4 pt-3">
        <p className="text-[15px] leading-snug font-medium">{task.title}</p>
        <span className="flex justify-end">
          {task.assignee && (
            <span
              title={task.assignee}
              className="flex size-7 items-center justify-center rounded-full bg-muted text-[11px] font-semibold"
            >
              {getNameInitials(task.assignee)}
            </span>
          )}
        </span>
      </CardContent>
    </Card>
  );
}

/** A single task, shown inside a `Column`. The drag gesture runs on Pointer
 * Events (not HTML5 draggable/dragstart) — mobile Safari/Chrome never fire
 * native dragstart for a touch gesture, but they do fire pointerdown/move/up
 * for both touch and mouse alike. The card captures the pointer and reports
 * start/move/end up to `useBoardDnd` (wired at the page level).
 *
 * While dragging, the card itself stays put at reduced opacity (a
 * placeholder for where it came from) and a `DragGhost` clone — portalled to
 * `document.body` — follows the cursor instead. Moving the original card via
 * `transform` used to work until it crossed the columns row's
 * `overflow-x-auto` edge, which clipped it into invisibility; the portal
 * sidesteps that ancestor entirely. */
export function TaskCard({
  task,
  onClick,
  draggable = true,
  dragging = false,
  onDragStart,
  onDragMove,
  onDragEnd,
  onDragCancel,
  accentClass,
  className,
}: TaskCardProps) {
  // Read-only rendering (SCR-05, AC-10): without a click handler the card is
  // plain content — no button semantics, focusability or pointer cursor; and
  // when not draggable, no drag wiring either.
  const clickable = onClick !== undefined;

  const cardRef = useRef<HTMLDivElement>(null);
  const ghostRef = useRef<HTMLDivElement>(null);
  const gestureStart = useRef<{ x: number; y: number } | null>(null);
  // The original card's on-screen rect at the moment the drag started — the
  // ghost is sized to match it and its translate is computed relative to it.
  const cardRect = useRef<{ left: number; top: number; width: number; height: number } | null>(null);
  const draggingRef = useRef(false);
  const cancelledRef = useRef(false);
  const suppressClickRef = useRef(false);
  const [ghosting, setGhosting] = useState(false);

  const resetDragVisuals = () => {
    const el = cardRef.current;
    if (el) {
      el.style.pointerEvents = "";
      el.style.opacity = "";
    }
    setGhosting(false);
  };

  // External cancel (Escape in useBoardDnd): the hook already dropped the
  // drag state — reset the visuals and ignore moves until the pointer is up.
  useEffect(() => {
    if (!dragging && draggingRef.current) {
      draggingRef.current = false;
      if (gestureStart.current) cancelledRef.current = true;
      resetDragVisuals();
    }
  }, [dragging]);

  function handlePointerDown(e: React.PointerEvent<HTMLDivElement>) {
    if (e.button !== 0) return;
    // A gesture starting on a nested button must not be captured at the card
    // level — capture would redirect the button's synthesized click here.
    if ((e.target as HTMLElement).closest("button")) return;
    gestureStart.current = { x: e.clientX, y: e.clientY };
    draggingRef.current = false;
    cancelledRef.current = false;
    suppressClickRef.current = false;
    e.currentTarget.setPointerCapture(e.pointerId);
  }

  function handlePointerMove(e: React.PointerEvent<HTMLDivElement>) {
    const start = gestureStart.current;
    if (!start || cancelledRef.current) return;
    const dx = e.clientX - start.x;
    const dy = e.clientY - start.y;
    if (!draggingRef.current) {
      if (Math.hypot(dx, dy) <= DRAG_THRESHOLD_PX) return;
      draggingRef.current = true;
      const el = cardRef.current;
      if (el) {
        const rect = el.getBoundingClientRect();
        cardRect.current = { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
        // pointer-events:none keeps the original out of elementFromPoint
        // (the ghost is already pointer-events:none), so useBoardDnd
        // resolves the column underneath the cursor; the captured pointer
        // still delivers events here (capture bypasses hit testing).
        el.style.pointerEvents = "none";
        el.style.opacity = "0.4";
      }
      setGhosting(true);
      onDragStart?.(task);
    }
    const rect = cardRect.current;
    if (ghostRef.current && rect) {
      ghostRef.current.style.transform = `translate(${rect.left + dx}px, ${rect.top + dy}px)`;
    }
    onDragMove?.(e.clientX, e.clientY);
  }

  function handlePointerUp() {
    if (!gestureStart.current) return;
    const wasDragging = draggingRef.current;
    const wasCancelled = cancelledRef.current;
    gestureStart.current = null;
    draggingRef.current = false;
    cancelledRef.current = false;
    if (wasDragging || wasCancelled) {
      suppressClickRef.current = true;
      resetDragVisuals();
      if (wasDragging) onDragEnd?.();
    }
  }

  function handlePointerCancel() {
    if (!gestureStart.current) return;
    const wasDragging = draggingRef.current;
    gestureStart.current = null;
    draggingRef.current = false;
    cancelledRef.current = false;
    resetDragVisuals();
    if (wasDragging) onDragCancel?.();
  }

  function handleClick() {
    if (suppressClickRef.current) {
      suppressClickRef.current = false;
      return;
    }
    onClick?.(task);
  }

  return (
    <>
      <TaskCardVisual
        ref={cardRef}
        task={task}
        accentClass={accentClass}
        className={cn(
          "transition-colors",
          clickable && "cursor-pointer hover:bg-accent/50",
          // touch-action:none is what lets pointermove fire for a touch drag
          // instead of scrolling the page.
          draggable && "touch-none select-none",
          className,
        )}
        role={clickable ? "button" : undefined}
        tabIndex={clickable ? 0 : undefined}
        onClick={clickable ? handleClick : undefined}
        onPointerDown={draggable ? handlePointerDown : undefined}
        onPointerMove={draggable ? handlePointerMove : undefined}
        onPointerUp={draggable ? handlePointerUp : undefined}
        onPointerCancel={draggable ? handlePointerCancel : undefined}
      />
      {ghosting && cardRect.current && (
        <DragGhost ref={ghostRef} width={cardRect.current.width} height={cardRect.current.height}>
          {/* data-slot overridden so this floating clone never matches a
           * `[data-slot="card"]` lookup (tests, `cardByTitle`) alongside the
           * real, stationary card it's cloned from. */}
          <TaskCardVisual
            task={task}
            accentClass={accentClass}
            className="h-full w-full"
            data-slot="drag-ghost-card"
          />
        </DragGhost>
      )}
    </>
  );
}
