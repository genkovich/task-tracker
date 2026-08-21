import { useCallback, useEffect, useRef, useState } from "react";
import { useParams } from "react-router";
import { Eye, Link2Off } from "lucide-react";

import { boardApi } from "@/features/board/api/boardApi";
import { usePublicBoardEvents } from "@/features/board/api/useBoardEvents";
import { showApiError } from "@/shared/lib/showApiError";
import { ApiClientError } from "@/shared/api/client";
import type { PublicBoardState } from "@/features/board/api/types";
import { BoardLoadError, BoardLoading } from "@/features/board/ui/BoardLoadState";
import { Column } from "@/features/board/ui/Column";
import { BoardShell } from "@/widgets/board-shell/ui/BoardShell";

/** SCR-05 public read-only board view (AC-09, AC-10): fetches via a
 * token-scoped `boardApi.getPublicBoard` and renders T15's `Column` in
 * read-only mode (no drag/edit/delete/quick-add affordances). On a 404
 * `board.link_invalid` response, renders the SCR-06 unavailable state
 * (AC-11) instead of the board. */
export default function BoardPublicPage() {
  const { token = "" } = useParams<{ token: string }>();
  const [board, setBoard] = useState<PublicBoardState | null>(null);
  const [unavailable, setUnavailable] = useState(false);
  const [failed, setFailed] = useState(false);
  const boardRef = useRef<PublicBoardState | null>(null);

  const refetch = useCallback(() => {
    boardApi
      .getPublicBoard(token)
      .then((state) => {
        boardRef.current = state;
        setBoard(state);
        setFailed(false);
      })
      .catch((err) => {
        // A 404 means the link was revoked/invalid — SCR-06 always, even
        // over an already-rendered board (not a stale-state case).
        if (err instanceof ApiClientError && err.statusCode === 404) {
          setUnavailable(true);
          return;
        }
        // The full error screen is for the initial load only; a failed
        // refetch keeps the stale board on screen and surfaces a toast.
        if (boardRef.current === null) {
          setFailed(true);
          return;
        }
        showApiError(err);
      });
  }, [token]);

  useEffect(() => {
    refetch();
  }, [refetch]);

  usePublicBoardEvents(token, refetch);

  // SCR-06 (Design/scr06-link-unavailable-*): центрований стан на чорному,
  // без шапки борда — глядач за мертвим лінком не має бачити chrome.
  if (unavailable) {
    return (
      <main className="dark flex min-h-screen flex-col items-center justify-center gap-2 bg-background px-6 text-center font-sans text-foreground [color-scheme:dark]">
        <span className="mb-3 flex size-16 items-center justify-center rounded-full bg-muted">
          <Link2Off aria-hidden className="size-7 text-muted-foreground" />
        </span>
        <h1 className="text-2xl font-bold tracking-tight">Цей лінк більше недоступний</h1>
        <p className="text-muted-foreground">Власник дошки відкликав це публічне посилання.</p>
      </main>
    );
  }

  // A guest link has no ProtectedLayout (no sidebar/topbar), so this page
  // brings its own minimal chrome — same forced-dark treatment as LoginPage
  // (Design/scr05* renders dark regardless of the visitor's own theme
  // preference; BoardShell itself is theme-neutral now, it no longer
  // supplies this).
  return (
    <div className="dark flex min-h-screen flex-col bg-background font-sans text-foreground [color-scheme:dark]">
      <header className="flex h-16 shrink-0 items-center px-4 sm:px-6">
        <span className="text-[17px] font-bold tracking-tight">Task Tracker</span>
      </header>
      <main className="flex w-full flex-1 flex-col gap-6 px-4 py-6 sm:px-8 sm:py-8">
        <BoardShell
          actions={
            <span className="flex h-9 items-center gap-2 rounded-full bg-muted px-4 text-sm font-medium text-muted-foreground">
              <Eye aria-hidden className="size-4" />
              Лише перегляд
            </span>
          }
        >
          {failed ? (
            <BoardLoadError onRetry={refetch} />
          ) : !board ? (
            <BoardLoading />
          ) : (
            <div className="flex-1 rounded-3xl sm:bg-muted/50 sm:p-5">
              <div className="flex flex-col gap-8 sm:flex-row sm:gap-5 sm:overflow-x-auto sm:pb-1">
                {board.columns.map((column) => (
                  <Column key={column.id} column={column} readOnly />
                ))}
              </div>
            </div>
          )}
        </BoardShell>
      </main>
    </div>
  );
}
