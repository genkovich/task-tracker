import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { LayoutGridIcon } from "lucide-react";
import { Skeleton } from "@/shared/ui/skeleton";
import { EmptyState } from "@/shared/ui/EmptyState";
import { Button } from "@/shared/ui/button";
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

  const load = useCallback(async () => {
    try {
      const items = await boardApi.listCards();
      setCards(items);
      setState("loaded");
    } catch {
      setState("error");
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
    } catch {
      setCards(previous);
      toast.error("Could not save the move — try again");
    }
  }

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
    <div className="flex gap-4">
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
          onDrop={handleDrop}
        />
      ))}
    </div>
  );
}
