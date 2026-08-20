import { apiClient } from "@/shared/api/client";
import type { BoardState, PublicBoardState, Task, TaskCreate, TaskUpdate } from "./types";

export const boardApi = {
  getBoard: () => apiClient.get<BoardState>("/board"),

  // Public viewer (AC-09, AC-11) — token-scoped, read-only board fetch;
  // rejects with an ApiClientError (404 `board.link_invalid`) for an
  // invalid/revoked token, per contracts/openapi.yaml PublicLinkInvalid.
  getPublicBoard: (token: string) => apiClient.get<PublicBoardState>(`/public/${token}/board`),

  createTask: (data: TaskCreate) => apiClient.post<Task>("/tasks", data),

  editTask: (taskId: string, data: TaskUpdate) => apiClient.patch<Task>(`/tasks/${taskId}`, data),

  moveTask: (taskId: string, columnId: string) =>
    apiClient.post<Task>(
      `/tasks/${taskId}/move`,
      { column_id: columnId },
      { "Idempotency-Key": crypto.randomUUID() },
    ),

  deleteTask: (taskId: string) => apiClient.delete<void>(`/tasks/${taskId}`),
};
