import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { LayoutGridIcon } from "lucide-react";
import { Skeleton } from "@/shared/ui/skeleton";
import { EmptyState } from "@/shared/ui/EmptyState";
import { Button } from "@/shared/ui/button";
import { ApiClientError } from "@/shared/api/client";
import { COLUMNS, type Card, type ColumnStatus } from "@/entities/card/model/types";
import { BoardColumn } from "@/entities/card/ui/BoardColumn";
import { boardApi, subscribeToBoardEvents } from "../api/boardApi";

type LoadState = "loading" | "loaded" | "error";

interface BoardViewProps {
  readonly readOnly?: boolean;
  readonly onAddCard?: (status: ColumnStatus) => void;
  readonly onEditCard?: (card: Card) => void;
}

export function BoardView({ readOnly, onAddCard, onEditCard }: BoardViewProps) {
  const [cards, setCards] = useState<Card[]>([]);
  const [state, setState] = useState<LoadState>("loading");
  const draggedCard = useRef<Card | null>(null);
  const loadedOnce = useRef(false);

  const load = useCallback(async () => {
    try {
      const items = await boardApi.listCards();
      setCards(items);
      setState("loaded");
      loadedOnce.current = true;
    } catch {
      // Only the *first* load blanks the screen — a refetch failure on an
      // already-loaded board keeps showing the last known state and just
      // toasts (screens.md SCR-01 error state).
      if (loadedOnce.current) {
        toast.error("Couldn't refresh the board — showing the last known state");
      } else {
        setState("error");
      }
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    return subscribeToBoardEvents(load);
  }, [load]);

  async function handleDrop(target: ColumnStatus) {
    const card = draggedCard.current;
    draggedCard.current = null;
    if (readOnly || !card || card.column_status === target) return;

    const previous = cards;
    setCards((cur) => cur.map((c) => (c.id === card.id ? { ...c, column_status: target } : c)));

    try {
      await boardApi.moveCard(card.id, target);
    } catch (err) {
      if (err instanceof ApiClientError && err.statusCode === 404) {
        // The card was deleted by someone else while this drag was in
        // flight — the delete wins (AC-15). Silent no-op, not a
        // user-facing error: just drop it from the board.
        setCards((cur) => cur.filter((c) => c.id !== card.id));
        return;
      }
      setCards(previous);
      toast.error("Could not save the move — try again");
    }
  }

  // Pointer Events drive the drag end to end (see BoardCard/BoardColumn) —
  // this window-level listener resolves *where* the pointer was released,
  // since a captured pointer keeps delivering events to the card element
  // even once the finger/cursor has moved off it.
  useEffect(() => {
    if (readOnly) return;

    function onPointerUp(e: PointerEvent) {
      if (!draggedCard.current) return;
      const el = document.elementFromPoint(e.clientX, e.clientY);
      const columnEl = el?.closest("[data-column-status]");
      const target = columnEl?.getAttribute("data-column-status") as ColumnStatus | null;
      if (target) {
        handleDrop(target);
      } else {
        draggedCard.current = null;
      }
    }

    window.addEventListener("pointerup", onPointerUp);
    return () => window.removeEventListener("pointerup", onPointerUp);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [readOnly, cards]);

  async function handleDelete(card: Card) {
    const previous = cards;
    setCards((cur) => cur.filter((c) => c.id !== card.id));
    try {
      await boardApi.deleteCard(card.id);
    } catch {
      setCards(previous);
      toast.error("Could not delete the card — try again");
    }
  }

  if (state === "loading") {
    return (
      <div className="flex gap-4" data-testid="board-loading">
        {COLUMNS.map((c) => (
          <div key={c.status} className="flex-1 space-y-3">
            <Skeleton className="h-5 w-24" />
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ))}
      </div>
    );
  }

  if (state === "error") {
    return (
      <EmptyState
        Icon={LayoutGridIcon}
        title="Couldn't load the board"
        description="Check your connection and try again."
        action={
          <Button onClick={load} variant="outline">
            Retry
          </Button>
        }
      />
    );
  }

  if (cards.length === 0) {
    return (
      <EmptyState
        Icon={LayoutGridIcon}
        title="No cards yet"
        description={
          readOnly ? "The team hasn't added any cards yet." : "Add the first card to get started."
        }
        action={
          !readOnly && onAddCard ? (
            <Button onClick={() => onAddCard("todo")}>Add card</Button>
          ) : undefined
        }
      />
    );
  }

  return (
    // Stacked on narrow (phone) viewports, side-by-side from sm: up — a
    // fixed-width 3-column row is wider than a phone screen (3×min-w-64 +
    // gaps ≈ 800px), which would force horizontal scrolling and make touch
    // drag unreachable for a column off-screen. Mobile-first per ux-flows.md.
    <div className="flex flex-col gap-4 sm:flex-row">
      {COLUMNS.map(({ status, label }) => (
        <BoardColumn
          key={status}
          status={status}
          label={label}
          cards={cards.filter((c) => c.column_status === status)}
          readOnly={readOnly}
          onAddCard={onAddCard}
          onEditCard={onEditCard}
          onDeleteCard={handleDelete}
          onDragStart={(card) => (draggedCard.current = card)}
        />
      ))}
    </div>
  );
}
