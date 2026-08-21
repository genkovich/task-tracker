import { useState } from "react";

import { boardApi } from "@/features/board/api/boardApi";
import type { Task } from "@/features/board/api/types";
import { showApiError } from "@/shared/lib/showApiError";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";

export interface QuickAddTaskProps {
  /** Дошка, в найлівішу колонку якої створюється задача (boards BRD-08). */
  boardId: string;
  onCreated: (task: Task) => void;
  /** Закрити форму (Esc або «Скасувати») — станом open володіє Column (scr02). */
  onCancel?: () => void;
}

/** Inline quick-add form for the leftmost column (SCR-02).
 *
 * AC-01: a non-empty title creates the task and hands it back via `onCreated`
 * so the caller can show it immediately.
 * AC-02: an empty title shows an inline required-name error and never calls
 * the API.
 */
export function QuickAddTask({ boardId, onCreated, onCancel }: QuickAddTaskProps) {
  const [title, setTitle] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      setError("Назва обов'язкова");
      return;
    }

    setError(null);
    setSubmitting(true);
    try {
      const created = await boardApi.createTask({ board_id: boardId, title: trimmedTitle });
      onCreated(created);
      setTitle("");
    } catch (err) {
      showApiError(err);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex flex-col gap-3 rounded-2xl border border-white/10 bg-background p-3">
      <Input
        autoFocus
        value={title}
        placeholder="Назва задачі"
        aria-label="Назва задачі"
        aria-invalid={error ? true : undefined}
        className="h-10 rounded-xl"
        onChange={(e) => {
          setTitle(e.target.value);
          if (error) setError(null);
        }}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            void handleSubmit();
          }
          if (e.key === "Escape") {
            e.preventDefault();
            onCancel?.();
          }
        }}
      />
      {error && (
        <p className="text-destructive text-sm" role="alert">
          {error}
        </p>
      )}
      <div className="flex items-center justify-end gap-2">
        {onCancel && (
          <Button type="button" variant="ghost" size="sm" className="rounded-full" onClick={onCancel}>
            Скасувати
          </Button>
        )}
        <Button
          type="button"
          size="sm"
          className="rounded-full px-4"
          onClick={() => void handleSubmit()}
          disabled={submitting}
        >
          Додати
        </Button>
      </div>
    </div>
  );
}
