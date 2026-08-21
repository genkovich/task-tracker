import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Button } from "@/shared/ui/button";
import { ApiClientError } from "@/shared/api/client";
import type { Card } from "@/entities/card/model/types";
import { cardFormApi } from "../api/cardFormApi";

interface CardFormDialogProps {
  readonly open: boolean;
  readonly card?: Card;
  readonly onClose: () => void;
  readonly onSaved: (card: Card) => void;
}

export function CardFormDialog({ open, card, onClose, onSaved }: CardFormDialogProps) {
  const isEdit = Boolean(card);
  const [name, setName] = useState("");
  const [assignee, setAssignee] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      setName(card?.name ?? "");
      setAssignee(card?.assignee ?? "");
      setError(null);
    }
  }, [open, card]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);

    try {
      const saved = isEdit
        ? await cardFormApi.updateCard(card!.id, name, assignee || null)
        : await cardFormApi.createCard(name, assignee || null);
      onSaved(saved);
      onClose();
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError("Something went wrong");
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit card" : "Add card"}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Change the card's name and/or assignee."
              : "Give the card a name and, optionally, an assignee."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="card-name">Name</Label>
            <Input
              id="card-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={200}
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="card-assignee">Assignee (optional)</Label>
            <Input
              id="card-assignee"
              value={assignee}
              onChange={(e) => setAssignee(e.target.value)}
              maxLength={100}
            />
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <DialogFooter>
            <Button type="submit" disabled={saving}>
              {saving ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
