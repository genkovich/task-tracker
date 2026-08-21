// Mirrors docs/features/board/contracts/openapi.yaml components/schemas.

export interface Task {
  id: string;
  column_id: string;
  title: string;
  assignee: string | null;
  created_at: string;
  updated_at: string;
}

export interface Column {
  id: string;
  name: string;
  position: number;
  tasks: Task[];
}

export interface PublicLink {
  token: string;
  created_at: string;
}

export interface BoardState {
  columns: Column[];
  public_link: PublicLink | null;
}

// Public-viewer view of the board (AC-09, AC-10) — same columns/tasks shape,
// no `public_link` (irrelevant to a viewer already holding one).
export interface PublicBoardState {
  columns: Column[];
}

export interface TaskCreate {
  title: string;
  assignee?: string | null;
}

export interface TaskUpdate {
  title?: string;
  assignee?: string | null;
}
