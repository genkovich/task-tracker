import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { KanbanSquare, LayoutDashboard, Plus } from "lucide-react";

import type { Route } from "./+types/DashboardPage";
import { boardApi } from "@/features/board/api/boardApi";
import type { BoardSummary } from "@/features/board/api/types";
import { showApiError } from "@/shared/lib/showApiError";
import { Button } from "@/shared/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog";
import { EmptyState } from "@/shared/ui/EmptyState";
import { Input } from "@/shared/ui/input";

export const meta: Route.MetaFunction = () => [{ title: "Dashboard — Task Tracker" }];

/** Живий дашборд (boards BRD-01/BRD-02): список усіх дощок із кількістю
 * задач і створення нової. Клік по дошці відкриває /board/:boardId. */
export default function DashboardPage() {
  const navigate = useNavigate();
  const [boards, setBoards] = useState<BoardSummary[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [creating, setCreating] = useState(false);

  const refetch = useCallback(() => {
    boardApi
      .listBoards()
      .then((list) => {
        setBoards(list);
        setFailed(false);
      })
      .catch(() => setFailed(true));
  }, []);

  useEffect(() => {
    refetch();
  }, [refetch]);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold tracking-tight">Your boards</h1>
        <Button className="rounded-full gap-1.5" onClick={() => setCreating(true)}>
          <Plus className="size-4" />
          <span className="hidden sm:inline">New board</span>
        </Button>
      </div>

      {failed ? (
        <div className="min-h-[520px] rounded-xl bg-muted">
          <EmptyState
            Icon={LayoutDashboard}
            title="Couldn't load boards"
            description="Something went wrong while loading your boards."
            action={
              <Button className="rounded-full" onClick={refetch}>
                Try again
              </Button>
            }
            className="min-h-[520px] justify-center"
          />
        </div>
      ) : boards === null ? (
        <div className="flex min-h-[520px] items-center justify-center rounded-xl bg-muted">
          <div className="size-6 animate-spin rounded-full border-2 border-muted-foreground/25 border-t-muted-foreground" />
        </div>
      ) : boards.length === 0 ? (
        <div className="min-h-[520px] rounded-xl bg-muted">
          <EmptyState
            Icon={LayoutDashboard}
            title="No boards yet"
            description="Create your first board to start organizing tasks."
            action={
              <Button className="rounded-full" onClick={() => setCreating(true)}>
                Create board
              </Button>
            }
            className="min-h-[520px] justify-center"
          />
        </div>
      ) : (
        <ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {boards.map((board) => (
            <li key={board.id}>
              <button
                type="button"
                onClick={() => navigate(`/board/${board.id}`)}
                className="flex w-full items-center gap-4 rounded-xl border bg-card p-5 text-left transition-colors hover:border-primary/40 hover:bg-accent"
              >
                <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                  <KanbanSquare className="size-5 text-muted-foreground" />
                </span>
                <span className="min-w-0">
                  <span className="block truncate font-medium">{board.name}</span>
                  <span className="block text-sm text-muted-foreground">
                    {formatTaskCount(board.task_count)}
                  </span>
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}

      <CreateBoardDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={(boardId) => navigate(`/board/${boardId}`)}
      />
    </div>
  );
}

function formatTaskCount(count: number): string {
  return count === 1 ? "1 задача" : `${count} задач`;
}

interface CreateBoardDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (boardId: string) => void;
}

/** Діалог створення дошки (BRD-02/BRD-03): непорожня назва обов'язкова —
 * порожня блокується ще до запиту, як у quick-add задачі. */
function CreateBoardDialog({ open, onOpenChange, onCreated }: CreateBoardDialogProps) {
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    const trimmed = name.trim();
    if (!trimmed) {
      setError("Назва обов'язкова");
      return;
    }

    setError(null);
    setSubmitting(true);
    try {
      const board = await boardApi.createBoard(trimmed);
      setName("");
      onOpenChange(false);
      onCreated(board.id);
    } catch (err) {
      showApiError(err);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          setName("");
          setError(null);
        }
        onOpenChange(next);
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Створити дошку</DialogTitle>
          <DialogDescription>
            Нова дошка одразу отримає колонки To Do, In Progress і Done.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          <Input
            autoFocus
            value={name}
            placeholder="Назва дошки"
            aria-label="Назва дошки"
            aria-invalid={error ? true : undefined}
            maxLength={200}
            onChange={(e) => {
              setName(e.target.value);
              if (error) setError(null);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                void handleSubmit();
              }
            }}
          />
          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            Скасувати
          </Button>
          <Button onClick={() => void handleSubmit()} disabled={submitting}>
            Створити
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
