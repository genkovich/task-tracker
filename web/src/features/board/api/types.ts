// Mirrors docs/features/board/contracts/openapi.yaml components/schemas,
// extended by docs/features/boards/contracts/openapi.yaml (multi-board) and
// docs/features/tasks/contracts/openapi.yaml (rich tasks).

export type TaskPriority = "low" | "medium" | "high";

// A task as it appears on the board (contract `TaskCard`). Deliberately no
// `description`: the board state is refetched on every SSE event, so it
// carries only whether there is one, plus the comment count — enough for the
// card's markers, nothing more.
export interface Task {
  id: string;
  column_id: string;
  title: string;
  assignee: string | null;
  priority: TaskPriority;
  /** Calendar day, `YYYY-MM-DD`, never a timestamp. */
  due_date: string | null;
  has_description: boolean;
  comment_count: number;
  created_at: string;
  updated_at: string;
}

// The full task row (contract `Task`) — what create/edit return and what the
// detail view shows.
export interface TaskRecord {
  id: string;
  column_id: string;
  title: string;
  assignee: string | null;
  description: string;
  priority: TaskPriority;
  due_date: string | null;
  created_at: string;
  updated_at: string;
}

export interface TaskComment {
  id: string;
  task_id: string;
  author: string;
  body: string;
  created_at: string;
}

// Contract `TaskDetail` — the same payload for the editor and the viewer;
// only the surrounding UI differs (tasks TSK-12).
export interface TaskDetail {
  task: TaskRecord;
  comments: TaskComment[];
}

export interface CommentCreate {
  author: string;
  body: string;
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
  description?: string;
  priority?: TaskPriority;
  due_date?: string | null;
}

// Not a merge-patch (tasks contract): a field left out takes its zero value,
// which is how a due date or an assignee gets cleared.
export interface TaskUpdate {
  title: string;
  assignee?: string | null;
  description?: string;
  priority?: TaskPriority;
  due_date?: string | null;
}
