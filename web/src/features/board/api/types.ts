// Mirrors docs/features/board/contracts/openapi.yaml components/schemas,
// extended by docs/features/boards/contracts/openapi.yaml (multi-board).

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
  id: string;
  name: string;
  created_at: string;
  columns: Column[];
  public_link: PublicLink | null;
}

// One dashboard row (boards BRD-01).
export interface BoardSummary {
  id: string;
  name: string;
  created_at: string;
  task_count: number;
}

export interface BoardCreate {
  name: string;
}

// Public-viewer view of the board (AC-09, AC-10) — same columns/tasks shape,
// no `public_link` (irrelevant to a viewer already holding one).
export interface PublicBoardState {
  columns: Column[];
}

export interface TaskCreate {
  // The board whose leftmost column receives the task (boards BRD-08) —
  // the column itself is still chosen by the server.
  board_id: string;
  title: string;
  assignee?: string | null;
}

export interface TaskUpdate {
  title?: string;
  assignee?: string | null;
}
