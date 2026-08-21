import { useCallback, useEffect, useState } from "react";

import { ApiClientError } from "@/shared/api/client";
import { boardApi } from "@/features/board/api/boardApi";
import type { Task, TaskDetail, TaskPriority } from "@/features/board/api/types";
import { formatDueDate, isOverdue } from "@/features/board/lib/dueDate";
import { TaskComments } from "@/features/board/ui/TaskComments";
import { showApiError } from "@/shared/lib/showApiError";
import { cn } from "@/shared/lib/utils";
import { Button } from "@/shared/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Textarea } from "@/shared/ui/textarea";

export interface TaskDetailsModalProps {
  /** The card the modal was opened from — its title fills the header while
   * the detail request is still in flight, so the dialog never opens blank. */
  task: Task;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
  onDeleted: (taskId: string) => void;
  /** Present only for a viewer holding a public link, and that is exactly
   * what puts the dialog in read-only mode (SCR-07, TSK-12): an editor never
   * has a token, a viewer never has anything else. */
  publicToken?: string;
  /** The token turned out to be dead, or the task is not on its board
   * (TSK-13). Only the page can answer that honestly — SCR-06 replaces the
   * whole screen, not just the dialog — so the modal reports and steps back. */
  onLinkInvalid?: () => void;
}

const PRIORITIES: { value: TaskPriority; label: string }[] = [
  { value: "low", label: "Низький" },
  { value: "medium", label: "Звичайний" },
  { value: "high", label: "Високий" },
];

const PRIORITY_LABEL: Record<TaskPriority, string> = {
  low: "Низький",
  medium: "Звичайний",
  high: "Високий",
};

// The native control on purpose, not a custom popup: three fixed options, and
// on a phone (the viewer's device at a workshop) the platform picker beats
// anything hand-rolled. Styled with the same tokens as Input.
const SELECT_CLASS =
  "h-10 w-full rounded-xl border border-input bg-transparent px-3 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30";

/** SCR-03 / SCR-07 — the task's details. One component for both audiences:
 * the editor gets fields, the viewer gets the same content as text, because
 * "the viewer sees what the editor sees, minus every way to change it" is
 * easier to keep true in one file than in two. */
export function TaskDetailsModal({
  task,
  open,
  onOpenChange,
  onSaved,
  onDeleted,
  publicToken,
  onLinkInvalid,
}: TaskDetailsModalProps) {
  const readOnly = publicToken !== undefined;

  const [detail, setDetail] = useState<TaskDetail | null>(null);
  const [loadFailed, setLoadFailed] = useState(false);
  const [title, setTitle] = useState(task.title);
  const [assignee, setAssignee] = useState(task.assignee ?? "");
  const [description, setDescription] = useState("");
  const [priority, setPriority] = useState<TaskPriority>(task.priority);
  const [dueDate, setDueDate] = useState(task.due_date ?? "");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);

  // `resetForm` is the whole point of the parameter: the first load seeds the
  // fields from the server, but a re-read after a comment must NOT — the
  // person may have typed a new description in the meantime, and refilling the
  // fields would throw it away without a word.
  const load = useCallback(
    async ({ resetForm }: { resetForm: boolean }) => {
      try {
        const loaded = publicToken
          ? await boardApi.getPublicTask(publicToken, task.id)
          : await boardApi.getTask(task.id);
        setDetail(loaded);
        setLoadFailed(false);
        if (resetForm) {
          setTitle(loaded.task.title);
          setAssignee(loaded.task.assignee ?? "");
          setDescription(loaded.task.description);
          setPriority(loaded.task.priority);
          setDueDate(loaded.task.due_date ?? "");
        }
      } catch (err) {
        // A viewer whose link died mid-session belongs on the honest
        // "link is gone" screen (TSK-13), not on a generic load failure —
        // the page owns that state, so hand the decision up.
        if (publicToken !== undefined && err instanceof ApiClientError && err.statusCode === 404) {
          onLinkInvalid?.();
          return;
        }
        setLoadFailed(true);
        showApiError(err);
      }
    },
    [publicToken, task.id, onLinkInvalid],
  );

  useEffect(() => {
    void load({ resetForm: true });
  }, [load]);

  const handleSave = async () => {
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      setError("Назва обов'язкова");
      return;
    }

    setError(null);
    setSaving(true);
    try {
      await boardApi.editTask(task.id, {
        title: trimmedTitle,
        assignee: assignee.trim() || null,
        description: description.trim(),
        priority,
        due_date: dueDate || null,
      });
      onSaved();
      onOpenChange(false);
    } catch (err) {
      showApiError(err);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await boardApi.deleteTask(task.id);
      onDeleted(task.id);
      onOpenChange(false);
    } catch (err) {
      showApiError(err);
    } finally {
      setDeleting(false);
    }
  };

  // Both re-throw after surfacing the error: TaskComments only knows whether
  // it may clear the box the person typed into, and a swallowed rejection
  // would read to it as success.
  const handleAddComment = async (author: string, body: string) => {
    try {
      await boardApi.addComment(task.id, { author, body });
      await load({ resetForm: false });
      onSaved();
    } catch (err) {
      showApiError(err);
      throw err;
    }
  };

  const handleDeleteComment = async (commentId: string) => {
    try {
      await boardApi.deleteComment(task.id, commentId);
      await load({ resetForm: false });
      onSaved();
    } catch (err) {
      showApiError(err);
      throw err;
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {/* SCR-03 (Design/scr03-edit-task-*): темна модалка з великим радіусом,
       * Delete зліва, Cancel/Save справа; хрестика в прототипі немає. */}
      <DialogContent
        showCloseButton={readOnly}
        className="max-h-[85vh] overflow-y-auto rounded-3xl bg-popover p-6 font-sans text-popover-foreground sm:max-w-lg"
      >
        <DialogHeader className="text-left">
          <DialogTitle className="text-xl font-semibold">
            {readOnly ? (detail?.task.title ?? task.title) : "Деталі задачі"}
          </DialogTitle>
          <DialogDescription className="sr-only">
            {readOnly
              ? "Опис, пріоритет, дедлайн і коментарі задачі — лише перегляд."
              : "Змініть деталі задачі й збережіть, додайте коментар або видаліть задачу."}
          </DialogDescription>
        </DialogHeader>

        {loadFailed ? (
          <p className="text-sm text-muted-foreground" role="alert">
            Не вдалося завантажити деталі задачі.
          </p>
        ) : readOnly ? (
          <ReadOnlyDetails detail={detail} fallback={task} />
        ) : (
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="task-title" className="text-sm">
                Назва
              </Label>
              <Input
                id="task-title"
                value={title}
                aria-invalid={error ? true : undefined}
                className="h-10 rounded-xl"
                onChange={(e) => {
                  setTitle(e.target.value);
                  if (error) setError(null);
                }}
              />
              {error && (
                <p className="text-sm text-destructive" role="alert">
                  {error}
                </p>
              )}
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="task-description" className="text-sm">
                Опис
              </Label>
              <Textarea
                id="task-description"
                value={description}
                rows={5}
                className="rounded-xl"
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>

            <div className="flex flex-col gap-4 sm:flex-row">
              <div className="flex flex-1 flex-col gap-2">
                <Label htmlFor="task-priority" className="text-sm">
                  Пріоритет
                </Label>
                <select
                  id="task-priority"
                  value={priority}
                  className={SELECT_CLASS}
                  onChange={(e) => setPriority(e.target.value as TaskPriority)}
                >
                  {PRIORITIES.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>

              <div className="flex flex-1 flex-col gap-2">
                <Label htmlFor="task-due-date" className="text-sm">
                  Дедлайн
                </Label>
                <Input
                  id="task-due-date"
                  type="date"
                  value={dueDate}
                  className="h-10 rounded-xl"
                  onChange={(e) => setDueDate(e.target.value)}
                />
              </div>
            </div>

            <div className="flex flex-col gap-2">
              <Label htmlFor="task-assignee" className="text-sm">
                Виконавець
              </Label>
              <Input
                id="task-assignee"
                value={assignee}
                className="h-10 rounded-xl"
                onChange={(e) => setAssignee(e.target.value)}
              />
            </div>
          </div>
        )}

        {!loadFailed && (
          <TaskComments
            comments={detail?.comments ?? []}
            readOnly={readOnly}
            onAdd={handleAddComment}
            onDelete={handleDeleteComment}
          />
        )}

        {/* Save is gated on the detail having actually arrived: the fields
         * start from the card, which carries no description, so saving
         * before (or instead of) a successful load would quietly wipe the
         * description the card only knows exists. */}
        {!readOnly && detail && (
          <DialogFooter className="mt-2 flex-row items-center sm:justify-between">
            <Button
              type="button"
              variant="destructive"
              className="rounded-full px-4 dark:bg-destructive dark:hover:bg-destructive/90"
              onClick={() => void handleDelete()}
              disabled={deleting || saving}
            >
              Видалити
            </Button>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="ghost"
                className="rounded-full px-4"
                onClick={() => onOpenChange(false)}
                disabled={saving || deleting}
              >
                Скасувати
              </Button>
              <Button
                type="button"
                className="rounded-full px-4"
                onClick={() => void handleSave()}
                disabled={saving || deleting}
              >
                Зберегти
              </Button>
            </div>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}

/** The viewer's half of SCR-07: the same fields, rendered as text. There is
 * deliberately no input, no select and no button here — read-only is a fact
 * about what the markup contains, not a flag a handler checks. */
function ReadOnlyDetails({ detail, fallback }: { detail: TaskDetail | null; fallback: Task }) {
  const priority = detail?.task.priority ?? fallback.priority;
  const dueDate = detail?.task.due_date ?? fallback.due_date;
  const assignee = detail?.task.assignee ?? fallback.assignee;
  const description = detail?.task.description ?? "";
  const overdue = dueDate !== null && isOverdue(dueDate);

  return (
    <div className="flex flex-col gap-4">
      <dl className="flex flex-wrap gap-x-6 gap-y-2 text-sm">
        <div className="flex items-center gap-2">
          <dt className="text-muted-foreground">Пріоритет</dt>
          <dd className="font-medium">{PRIORITY_LABEL[priority]}</dd>
        </div>
        {dueDate && (
          <div className="flex items-center gap-2">
            <dt className="text-muted-foreground">Дедлайн</dt>
            <dd className={cn("font-medium", overdue && "text-destructive")}>
              {formatDueDate(dueDate)}
            </dd>
          </div>
        )}
        {assignee && (
          <div className="flex items-center gap-2">
            <dt className="text-muted-foreground">Виконавець</dt>
            <dd className="font-medium">{assignee}</dd>
          </div>
        )}
      </dl>

      <div className="flex flex-col gap-1">
        <h3 className="text-sm font-semibold">Опис</h3>
        <p className="text-sm whitespace-pre-wrap text-muted-foreground">
          {description || "Опису немає."}
        </p>
      </div>
    </div>
  );
}
