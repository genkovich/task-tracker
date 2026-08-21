import { apiClient } from "@/shared/api/client";
import type { Card } from "@/entities/card/model/types";

export const cardFormApi = {
  createCard: (name: string, assignee: string | null) =>
    apiClient.post<Card>("/cards", { name, assignee }),
  updateCard: (id: string, name: string, assignee: string | null) =>
    apiClient.patch<Card>(`/cards/${id}`, { name, assignee }),
};
