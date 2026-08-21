import { apiClient } from "@/shared/api/client";
import type {
  BoardState,
  BoardSummary,
  CommentCreate,
  PublicBoardState,
  TaskComment,
  TaskCreate,
  TaskDetail,
  TaskRecord,
  TaskUpdate,
} from "./types";

export const boardApi = {
  // Dashboard (boards BRD-01/BRD-02).
  listBoards: () => apiClient.get<BoardSummary[]>("/boards"),

  createBoard: (name: string) => apiClient.post<BoardState>("/boards", { name }),

  getBoard: (boardId: string) =>
    apiClient.get<BoardState>(`/boards/${encodeURIComponent(boardId)}`),

  // Public viewer (AC-09, AC-11) — token-scoped, read-only board fetch;
  // rejects with an ApiClientError (404 `board.link_invalid`) for an
  // invalid/revoked token, per contracts/openapi.yaml PublicLinkInvalid.
  getPublicBoard: (token: string) =>
    apiClient.get<PublicBoardState>(`/public/${encodeURIComponent(token)}/board`),

  createTask: (data: TaskCreate) => apiClient.post<TaskRecord>("/tasks", data),

  editTask: (taskId: string, data: TaskUpdate) =>
    apiClient.patch<TaskRecord>(`/tasks/${encodeURIComponent(taskId)}`, data),

  // No Idempotency-Key here: the contract doesn't define one, the API's CORS
  // AllowedHeaders would fail the preflight, and a move retry is naturally
  // idempotent server-side (same task, same column, last-write-wins).
  moveTask: (taskId: string, columnId: string) =>
    apiClient.post<TaskRecord>(`/tasks/${encodeURIComponent(taskId)}/move`, {
      column_id: columnId,
    }),

  deleteTask: (taskId: string) => apiClient.delete<void>(`/tasks/${encodeURIComponent(taskId)}`),

  // Task details (tasks TSK-01/TSK-08) — the full task plus its comments.
  getTask: (taskId: string) => apiClient.get<TaskDetail>(`/tasks/${encodeURIComponent(taskId)}`),

  // The viewer's twin of getTask (TSK-12). Rejects with a 404
  // `board.link_invalid` both for a dead token and for a task that belongs to
  // another board (TSK-13) — the viewer cannot tell the two apart, by design.
  getPublicTask: (token: string, taskId: string) =>
    apiClient.get<TaskDetail>(
      `/public/${encodeURIComponent(token)}/tasks/${encodeURIComponent(taskId)}`,
    ),

  addComment: (taskId: string, data: CommentCreate) =>
    apiClient.post<TaskComment>(`/tasks/${encodeURIComponent(taskId)}/comments`, data),

  deleteComment: (taskId: string, commentId: string) =>
    apiClient.delete<void>(
      `/tasks/${encodeURIComponent(taskId)}/comments/${encodeURIComponent(commentId)}`,
    ),
};
