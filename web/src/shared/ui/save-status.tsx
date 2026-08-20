import { Check } from "lucide-react";

interface SaveStatusProps {
  status: "idle" | "saving" | "saved" | "error";
}

export function SaveStatus({ status }: SaveStatusProps) {
  if (status === "idle") return null;

  if (status === "saving") {
    return (
      <span className="text-xs text-muted-foreground animate-pulse">
        Saving...
      </span>
    );
  }

  if (status === "saved") {
    return (
      <span className="flex items-center gap-1 text-xs text-green-600 dark:text-green-500">
        <Check className="size-3" />
        Saved
      </span>
    );
  }

  return (
    <span className="text-xs text-destructive">Failed to save</span>
  );
}