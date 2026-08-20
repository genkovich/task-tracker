import { apiClient } from "@/shared/api/client";
import type { BoardState, Task, TaskCreate, TaskUpdate } from "./types";

export const boardApi = {
  getBoard: () => apiClient.get<BoardState>("/board"),

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
