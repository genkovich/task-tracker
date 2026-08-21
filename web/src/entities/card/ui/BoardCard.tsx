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

export function BoardCard({ card, readOnly, onEdit, onDelete, onDragStart }: BoardCardProps) {
  return (
    <UiCard
      draggable={!readOnly}
      onDragStart={() => onDragStart?.(card)}
      onClick={() => !readOnly && onEdit?.(card)}
      className={`py-3 ${readOnly ? "" : "cursor-grab active:cursor-grabbing hover:shadow-md"}`}
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
