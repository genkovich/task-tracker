import { useEffect, useState } from "react";
import { Trash2 } from "lucide-react";

import { useAuth } from "@/app/providers/auth";
import type { TaskComment } from "@/features/board/api/types";
import { getDisplayName } from "@/shared/lib/user";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Textarea } from "@/shared/ui/textarea";

export interface TaskCommentsProps {
  comments: TaskComment[];
  /** SCR-07 (TSK-12): the thread renders as plain text — no form, no delete. */
  readOnly?: boolean;
  onAdd: (author: string, body: string) => Promise<void>;
  onDelete: (commentId: string) => Promise<void>;
}

function formatCommentTime(createdAt: string): string {
  const date = new Date(createdAt);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString("uk-UA", {
    day: "numeric",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/** The task's comment thread (SCR-03/SCR-07): oldest first, with an add form
 * for the editor only. The author field is free text pre-filled with the
 * signed-in person's name — there is no account behind a comment (ADR-0001),
 * so this is a convenience, not an identity. */
export function TaskComments({ comments, readOnly, onAdd, onDelete }: TaskCommentsProps) {
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const handleDelete = async (commentId: string) => {
    setDeletingId(commentId);
    try {
      await onDelete(commentId);
    } catch {
      // Reported by the caller; the row simply stays where it was.
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <section className="flex flex-col gap-3" aria-label="Коментарі">
      <h3 className="text-sm font-semibold">Коментарі</h3>

      {comments.length === 0 ? (
        <p className="text-sm text-muted-foreground">Коментарів ще немає.</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {comments.map((comment) => (
            <li key={comment.id} data-slot="comment" className="rounded-xl bg-muted/50 p-3">
              <div className="flex items-baseline justify-between gap-2">
                <span className="text-sm font-medium">{comment.author}</span>
                <span className="flex items-center gap-2">
                  <time className="text-[11px] text-muted-foreground" dateTime={comment.created_at}>
                    {formatCommentTime(comment.created_at)}
                  </time>
                  {!readOnly && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      className="rounded-full text-muted-foreground hover:text-destructive"
                      aria-label={`Видалити коментар від ${comment.author}`}
                      disabled={deletingId === comment.id}
                      onClick={() => void handleDelete(comment.id)}
                    >
                      <Trash2 />
                    </Button>
                  )}
                </span>
              </div>
              <p className="mt-1 text-sm whitespace-pre-wrap">{comment.body}</p>
            </li>
          ))}
        </ul>
      )}

      {!readOnly && <CommentForm onAdd={onAdd} />}
    </section>
  );
}

/** The add form, split out so the whole thing — including its `useAuth` call —
 * simply does not exist in the viewer's tree: a guest page must not depend on
 * the auth context to render its own content (TSK-12). */
function CommentForm({ onAdd }: { onAdd: TaskCommentsProps["onAdd"] }) {
  const { user } = useAuth();
  const defaultAuthor = user ? getDisplayName(user) : "";

  const [author, setAuthor] = useState(defaultAuthor);
  const [body, setBody] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // The signed-in user resolves asynchronously (auth/me), so the prefill has
  // to catch up once it lands — but never over something already typed.
  useEffect(() => {
    setAuthor((current) => (current === "" ? defaultAuthor : current));
  }, [defaultAuthor]);

  const handleSubmit = async () => {
    const trimmedAuthor = author.trim();
    const trimmedBody = body.trim();
    if (!trimmedAuthor) {
      setError("Вкажіть автора");
      return;
    }
    if (!trimmedBody) {
      setError("Коментар не може бути порожнім");
      return;
    }

    setError(null);
    setSubmitting(true);
    try {
      await onAdd(trimmedAuthor, trimmedBody);
      // Cleared only on success. A rejected post has already surfaced its
      // toast; wiping up to 2000 typed characters on top of that would turn a
      // retryable failure into lost work.
      setBody("");
    } catch {
      // The caller reported it; keeping the text is this component's job.
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <Label htmlFor="task-comment-author" className="text-sm">
        Автор
      </Label>
      <Input
        id="task-comment-author"
        value={author}
        className="h-10 rounded-xl"
        onChange={(e) => {
          setAuthor(e.target.value);
          if (error) setError(null);
        }}
      />
      <Label htmlFor="task-comment-body" className="text-sm">
        Новий коментар
      </Label>
      <Textarea
        id="task-comment-body"
        value={body}
        rows={3}
        className="rounded-xl"
        aria-invalid={error ? true : undefined}
        onChange={(e) => {
          setBody(e.target.value);
          if (error) setError(null);
        }}
      />
      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}
      <Button
        type="button"
        className="self-start rounded-full px-4"
        disabled={submitting}
        onClick={() => void handleSubmit()}
      >
        Додати коментар
      </Button>
    </div>
  );
}
