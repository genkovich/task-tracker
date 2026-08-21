export type ColumnStatus = "todo" | "in_progress" | "done";

export const COLUMNS: { status: ColumnStatus; label: string }[] = [
  { status: "todo", label: "To Do" },
  { status: "in_progress", label: "In Progress" },
  { status: "done", label: "Done" },
];

export interface Card {
  id: string;
  name: string;
  assignee: string | null;
  column_status: ColumnStatus;
  created_at: string;
  updated_at: string;
}
