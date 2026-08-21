import { apiClient } from "@/shared/api/client";
import type { BoardState, PublicBoardState, Task, TaskCreate, TaskUpdate } from "./types";

export const boardApi = {
  getBoard: () => apiClient.get<BoardState>("/board"),

  // Public viewer (AC-09, AC-11) — token-scoped, read-only board fetch;
  // rejects with an ApiClientError (404 `board.link_invalid`) for an
  // invalid/revoked token, per contracts/openapi.yaml PublicLinkInvalid.
  getPublicBoard: (token: string) =>
    apiClient.get<PublicBoardState>(`/public/${encodeURIComponent(token)}/board`),

  createTask: (data: TaskCreate) => apiClient.post<Task>("/tasks", data),

  editTask: (taskId: string, data: TaskUpdate) =>
    apiClient.patch<Task>(`/tasks/${encodeURIComponent(taskId)}`, data),

  // No Idempotency-Key here: the contract doesn't define one, the API's CORS
  // AllowedHeaders would fail the preflight, and a move retry is naturally
  // idempotent server-side (same task, same column, last-write-wins).
  moveTask: (taskId: string, columnId: string) =>
    apiClient.post<Task>(`/tasks/${encodeURIComponent(taskId)}/move`, { column_id: columnId }),

  deleteTask: (taskId: string) => apiClient.delete<void>(`/tasks/${encodeURIComponent(taskId)}`),
};
