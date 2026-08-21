import { apiClient, BASE_URL } from "@/shared/api/client";
import type { Card, ColumnStatus } from "@/entities/card/model/types";

interface CardPage {
  items: Card[];
  has_next: boolean;
  has_prev: boolean;
  next_cursor: string | null;
}

export const boardApi = {
  listCards: async (): Promise<Card[]> => {
    const page = await apiClient.get<CardPage>("/cards");
    return page.items;
  },
  moveCard: (id: string, columnStatus: ColumnStatus) =>
    apiClient.patch<Card>(`/cards/${id}/column`, { column_status: columnStatus }),
  deleteCard: (id: string) => apiClient.delete<void>(`/cards/${id}`),
};

/**
 * Subscribes to the board-wide SSE stream (ADR-0001). Returns an unsubscribe
 * function. The event payload itself is not consumed — every event just
 * means "something changed, refetch" (kept minimal by design).
 */
export function subscribeToBoardEvents(onEvent: () => void): () => void {
  const source = new EventSource(`${BASE_URL}/api/v1/cards/events`);
  source.onmessage = () => onEvent();
  return () => source.close();
}
