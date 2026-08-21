import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { LayoutGridIcon } from "lucide-react";
import { Skeleton } from "@/shared/ui/skeleton";
import { EmptyState } from "@/shared/ui/EmptyState";
import { Button } from "@/shared/ui/button";
import { COLUMNS, type Card } from "@/entities/card/model/types";
import { BoardColumn } from "@/entities/card/ui/BoardColumn";
import { ApiClientError } from "@/shared/api/client";
import { publicBoardApi, subscribeToPublicBoardEvents } from "../api/publicBoardApi";

type ViewState = "loading" | "board" | "error" | "not-found";

interface PublicBoardViewProps {
  readonly token: string;
  readonly onNotFound: () => void;
}

export function PublicBoardView({ token, onNotFound }: PublicBoardViewProps) {
  const [cards, setCards] = useState<Card[]>([]);
  const [state, setState] = useState<ViewState>("loading");
  const loadedOnce = useRef(false);

  const load = useCallback(async () => {
    try {
      const items = await publicBoardApi.getBoard(token);
      setCards(items);
      setState("board");
      loadedOnce.current = true;
    } catch (err) {
      // A disabled or never-valid token resolves as a plain-text 404
      // (AC-05) — the apiClient still surfaces it as an ApiClientError
      // with statusCode 404, even though the body isn't its usual JSON
      // envelope. Any other failure (network, 5xx) is a transient error.
      if (err instanceof ApiClientError && err.statusCode === 404) {
        setState("not-found");
        return;
      }
      // Only the *first* load blanks the screen — a background refetch
      // failure on an already-loaded board keeps the last known state and
      // just toasts (screens.md SCR-02 error state), so a live viewer's
      // board doesn't disappear mid-demo over one flaky poll.
      if (loadedOnce.current) {
        toast.error("Connection issue — the board may be stale");
      } else {
        setState("error");
      }
    }
  }, [token]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (state !== "board") return;
    return subscribeToPublicBoardEvents(
      token,
      () => load(),
      () => setState("not-found"),
    );
  }, [state, token, load]);

  useEffect(() => {
    if (state === "not-found") onNotFound();
  }, [state, onNotFound]);

  if (state === "loading") {
    return (
      <div className="flex gap-4" data-testid="public-board-loading">
        {COLUMNS.map((c) => (
          <div key={c.status} className="flex-1 space-y-3">
            <Skeleton className="h-5 w-24" />
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

  if (state === "not-found") {
    // The parent handles the actual not-found render (AC-05/AC-12 reuse
    // the app's existing NotFoundPage); nothing to show here.
    return null;
  }

  if (cards.length === 0) {
    return (
      <EmptyState
        Icon={LayoutGridIcon}
        title="No cards yet"
        description="The team hasn't added any cards yet."
      />
    );
  }

  return (
    // Stacked on narrow viewports — see BoardView's identical layout note.
    <div className="flex flex-col gap-4 sm:flex-row">
      {COLUMNS.map(({ status, label }) => (
        <BoardColumn
          key={status}
          status={status}
          label={label}
          cards={cards.filter((c) => c.column_status === status)}
          readOnly
        />
      ))}
    </div>
  );
}
