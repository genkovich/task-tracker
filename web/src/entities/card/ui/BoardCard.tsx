import { useRef } from "react";
import { Card as UiCard, CardContent } from "@/shared/ui/card";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { Trash2Icon } from "lucide-react";
import type { Card } from "@/entities/card/model/types";

interface BoardCardProps {
  readonly card: Card;
  readonly readOnly?: boolean;
  readonly onEdit?: (card: Card) => void;
  readonly onDelete?: (card: Card) => void;
  readonly onDragStart?: (card: Card) => void;
}

// Movement past this many pixels counts as a drag, not a tap — below it,
// releasing opens the edit dialog instead.
const DRAG_THRESHOLD_PX = 6;

export function BoardCard({ card, readOnly, onEdit, onDelete, onDragStart }: BoardCardProps) {
  // Pointer Events (not HTML5 draggable/dragstart) — mobile Safari/Chrome
  // never fire native dragstart for a touch gesture, but they do fire
  // pointerdown/pointermove/pointerup for both touch and mouse alike
  // (spec §6 NFR: drag must work by touch as well as by mouse).
  const pointerStart = useRef<{ x: number; y: number } | null>(null);
  const draggingRef = useRef(false);

  function handlePointerDown(e: React.PointerEvent<HTMLDivElement>) {
    if (readOnly) return;
    // Starting a gesture on the delete button (or any nested button) must
    // not capture the pointer at the card level — capture redirects every
    // later event, including the synthesized click, to the capturing
    // element, which would silently turn "tap delete" into "open edit".
    if ((e.target as HTMLElement).closest("button")) return;
    pointerStart.current = { x: e.clientX, y: e.clientY };
    draggingRef.current = false;
    e.currentTarget.setPointerCapture(e.pointerId);
  }

  function handlePointerMove(e: React.PointerEvent<HTMLDivElement>) {
    if (readOnly || !pointerStart.current || draggingRef.current) return;
    const dx = e.clientX - pointerStart.current.x;
    const dy = e.clientY - pointerStart.current.y;
    if (Math.hypot(dx, dy) > DRAG_THRESHOLD_PX) {
      draggingRef.current = true;
      onDragStart?.(card);
    }
  }

  function handlePointerUp() {
    pointerStart.current = null;
    // draggingRef is intentionally left as-is here — handleClick (which
    // fires right after pointerup) reads it to suppress opening the edit
    // dialog when this gesture was actually a drag, then resets it.
  }

  function handleClick() {
    if (readOnly) return;
    if (draggingRef.current) {
      draggingRef.current = false;
      return;
    }
    onEdit?.(card);
  }

  return (
    <UiCard
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onClick={handleClick}
      className={`touch-none py-3 ${readOnly ? "" : "cursor-grab active:cursor-grabbing hover:shadow-md"}`}
      data-testid={`card-${card.id}`}
    >
      <CardContent className="flex items-start justify-between gap-2 px-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">{card.name}</p>
          {card.assignee && (
            <Badge variant="secondary" className="mt-1">
              {card.assignee}
            </Badge>
          )}
        </div>
        {!readOnly && onDelete && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={`Delete ${card.name}`}
            className="size-6 shrink-0"
            onClick={(e) => {
              e.stopPropagation();
              onDelete(card);
            }}
          >
            <Trash2Icon className="size-4" />
          </Button>
        )}
      </CardContent>
    </UiCard>
  );
}
