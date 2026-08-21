import { Button } from "@/shared/ui/button";
import { PlusIcon } from "lucide-react";
import type { Card, ColumnStatus } from "@/entities/card/model/types";
import { BoardCard } from "./BoardCard";

interface BoardColumnProps {
  readonly status: ColumnStatus;
  readonly label: string;
  readonly cards: Card[];
  readonly readOnly?: boolean;
  readonly onAddCard?: (status: ColumnStatus) => void;
  readonly onEditCard?: (card: Card) => void;
  readonly onDeleteCard?: (card: Card) => void;
  readonly onDragStart?: (card: Card) => void;
  readonly onDrop?: (status: ColumnStatus) => void;
}

export function BoardColumn({
  status,
  label,
  cards,
  readOnly,
  onAddCard,
  onEditCard,
  onDeleteCard,
  onDragStart,
  onDrop,
}: BoardColumnProps) {
  return (
    <div
      className="flex min-w-64 flex-1 flex-col gap-3 rounded-lg bg-muted/40 p-3"
      onDragOver={(e) => !readOnly && e.preventDefault()}
      onDrop={() => !readOnly && onDrop?.(status)}
      data-testid={`column-${status}`}
    >
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold">{label}</h2>
        {!readOnly && onAddCard && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-6"
            aria-label={`Add card to ${label}`}
            onClick={() => onAddCard(status)}
          >
            <PlusIcon className="size-4" />
          </Button>
        )}
      </div>
      <div className="flex flex-col gap-2">
        {cards.map((card) => (
          <BoardCard
            key={card.id}
            card={card}
            readOnly={readOnly}
            onEdit={onEditCard}
            onDelete={onDeleteCard}
            onDragStart={onDragStart}
          />
        ))}
      </div>
    </div>
  );
}
