import { useState } from "react";
import type { Route } from "./+types/BoardPage";
import { Toaster } from "@/shared/ui/sonner";
import { BoardView } from "@/features/board-view/ui/BoardView";
import { CardFormDialog } from "@/features/card-form/ui/CardFormDialog";
import { PublicLinkControl } from "@/features/public-link-control/ui/PublicLinkControl";
import type { Card, ColumnStatus } from "@/entities/card/model/types";

export const meta: Route.MetaFunction = () => [{ title: "Board — Task Tracker" }];

type DialogState = { mode: "add"; column: ColumnStatus } | { mode: "edit"; card: Card } | null;

export default function BoardPage() {
  const [dialog, setDialog] = useState<DialogState>(null);

  return (
    <main className="min-h-screen bg-background p-4 sm:p-6">
      <div className="mx-auto max-w-6xl space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold">Task Tracker</h1>
          <PublicLinkControl />
        </div>
        <BoardView
          onAddCard={(column) => setDialog({ mode: "add", column })}
          onEditCard={(card) => setDialog({ mode: "edit", card })}
        />
      </div>
      <CardFormDialog
        open={dialog !== null}
        card={dialog?.mode === "edit" ? dialog.card : undefined}
        onClose={() => setDialog(null)}
        onSaved={() => {
          // The board's own SSE subscription picks up the resulting
          // card.created/card.updated event and refetches — no explicit
          // refetch call needed here.
        }}
      />
      <Toaster />
    </main>
  );
}
