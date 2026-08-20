import { useState } from "react";

import { boardApi } from "@/features/board/api/boardApi";
import type { Task } from "@/features/board/api/types";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";

export interface QuickAddTaskProps {
  onCreated: (task: Task) => void;
}

/** Inline quick-add form for the leftmost column (SCR-02).
 *
 * AC-01: a non-empty title creates the task and hands it back via `onCreated`
 * so the caller can show it immediately.
 * AC-02: an empty title shows an inline required-name error and never calls
 * the API.
 */
export function QuickAddTask({ onCreated }: QuickAddTaskProps) {
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
      const created = await boardApi.createTask({ title: trimmedTitle });
      onCreated(created);
      setTitle("");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <Label htmlFor="quick-add-task-title">Назва</Label>
      <Input
        id="quick-add-task-title"
        value={title}
        placeholder="Назва task"
        aria-invalid={error ? true : undefined}
        onChange={(e) => {
          setTitle(e.target.value);
          if (error) setError(null);
        }}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            void handleSubmit();
          }
        }}
      />
      {error && (
        <p className="text-destructive text-sm" role="alert">
          {error}
        </p>
      )}
      <Button type="button" size="sm" onClick={() => void handleSubmit()} disabled={submitting}>
        Додати task
      </Button>
    </div>
  );
}
