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
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Редагувати task</DialogTitle>
          <DialogDescription>
            Змініть назву або виконавця й збережіть, або видаліть task.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="edit-task-title">Назва</Label>
            <Input
              id="edit-task-title"
              value={title}
              aria-invalid={error ? true : undefined}
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
            <Label htmlFor="edit-task-assignee">Виконавець</Label>
            <Input
              id="edit-task-assignee"
              value={assignee}
              onChange={(e) => setAssignee(e.target.value)}
            />
          </div>
        </div>

        <DialogFooter className="sm:justify-between">
          <Button
            type="button"
            variant="destructive"
            onClick={handleDelete}
            disabled={deleting || saving}
          >
            Видалити
          </Button>
          <Button type="button" onClick={handleSave} disabled={saving || deleting}>
            Зберегти
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
