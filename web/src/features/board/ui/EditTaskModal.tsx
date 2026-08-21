import { useState } from "react";

import { boardApi } from "@/features/board/api/boardApi";
import type { Task } from "@/features/board/api/types";
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
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";

export interface EditTaskModalProps {
  task: Task;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: (task: Task) => void;
  onDeleted: (taskId: string) => void;
}

export function EditTaskModal({
  task,
  open,
  onOpenChange,
  onSaved,
  onDeleted,
}: EditTaskModalProps) {
  const [title, setTitle] = useState(task.title);
  const [assignee, setAssignee] = useState(task.assignee ?? "");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const handleSave = async () => {
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      setError("Назва обов'язкова");
      return;
    }

    setError(null);
    setSaving(true);
    try {
      const updated = await boardApi.editTask(task.id, {
        title: trimmedTitle,
        assignee: assignee.trim() || null,
      });
      onSaved(updated);
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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {/* SCR-03 (Design/scr03-edit-task-*): темна модалка з великим радіусом,
       * Delete зліва, Cancel/Save справа; хрестика в прототипі немає. */}
      <DialogContent
        showCloseButton={false}
        className="dark rounded-3xl border-white/10 bg-popover p-6 font-sans text-popover-foreground [color-scheme:dark] sm:max-w-md"
      >
        <DialogHeader className="text-left">
          <DialogTitle className="text-xl font-semibold">Редагувати задачу</DialogTitle>
          <DialogDescription className="sr-only">
            Змініть назву або виконавця й збережіть, або видаліть задачу.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-task-title" className="text-sm">
              Назва
            </Label>
            <Input
              id="edit-task-title"
              value={title}
              aria-invalid={error ? true : undefined}
              className="h-10 rounded-xl"
              onChange={(e) => {
                setTitle(e.target.value);
                if (error) setError(null);
              }}
            />
            {error && (
              <p className="text-destructive text-sm" role="alert">
                {error}
              </p>
            )}
          </div>

          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-task-assignee" className="text-sm">
              Виконавець
            </Label>
            <Input
              id="edit-task-assignee"
              value={assignee}
              className="h-10 rounded-xl"
              onChange={(e) => setAssignee(e.target.value)}
            />
          </div>
        </div>

        <DialogFooter className="mt-2 flex-row items-center sm:justify-between">
          <Button
            type="button"
            variant="destructive"
            className="rounded-full px-4 dark:bg-destructive dark:hover:bg-destructive/90"
            onClick={handleDelete}
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
              onClick={handleSave}
              disabled={saving || deleting}
            >
              Зберегти
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
