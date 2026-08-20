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

export interface TaskCreate {
  title: string;
  assignee?: string | null;
}

export interface TaskUpdate {
  title?: string;
  assignee?: string | null;
}
