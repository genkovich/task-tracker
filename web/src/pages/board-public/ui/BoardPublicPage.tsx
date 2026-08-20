import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router";
import { Link2Off } from "lucide-react";

import { boardApi } from "@/features/board/api/boardApi";
import { usePublicBoardEvents } from "@/features/board/api/useBoardEvents";
import { ApiClientError } from "@/shared/api/client";
import type { PublicBoardState } from "@/features/board/api/types";
import { Column } from "@/features/board/ui/Column";
import { EmptyState } from "@/shared/ui/EmptyState";

/** SCR-05 public read-only board view (AC-09, AC-10): fetches via a
 * token-scoped `boardApi.getPublicBoard` and renders T15's `Column` in
 * read-only mode (no drag/edit/delete/quick-add affordances). On a 404
 * `board.link_invalid` response, renders the SCR-06 unavailable state
 * (AC-11) instead of the board. */
export default function BoardPublicPage() {
  const { token = "" } = useParams<{ token: string }>();
  const [board, setBoard] = useState<PublicBoardState | null>(null);
  const [unavailable, setUnavailable] = useState(false);

  const refetch = useCallback(() => {
    boardApi
      .getPublicBoard(token)
      .then((state) => setBoard(state))
      .catch((err) => {
        if (err instanceof ApiClientError && err.statusCode === 404) {
          setUnavailable(true);
          return;
        }
        throw err;
      });
  }, [token]);

  useEffect(() => {
    refetch();
  }, [refetch]);

  usePublicBoardEvents(token, refetch);

  if (unavailable) {
    return (
      <main className="flex h-screen items-center justify-center p-4">
        <EmptyState
          Icon={Link2Off}
          title="Цей лінк більше недоступний"
          description="Власник дошки відкликав цей public link, або він недійсний."
        />
      </main>
    );
  }

  if (!board) return null;

  return (
    <main className="flex h-screen flex-col gap-4 p-4">
      <h1 className="text-xl font-semibold">Дошка (лише перегляд)</h1>

      <div className="flex flex-1 gap-4 overflow-x-auto">
        {board.columns.map((column) => (
          <Column key={column.id} column={column} readOnly />
        ))}
      </div>
    </main>
  );
}
