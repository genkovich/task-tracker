import { apiClient, BASE_URL } from "@/shared/api/client";
import type { Card } from "@/entities/card/model/types";

interface CardPage {
  items: Card[];
}

export const publicBoardApi = {
  getBoard: async (token: string): Promise<Card[]> => {
    const page = await apiClient.get<CardPage>(`/public/boards/${token}`);
    return page.items;
  },
};

interface BoardEventMessage {
  type: string;
}

/**
 * Subscribes to the token-scoped SSE stream (ADR-0001, AC-08, AC-12).
 * onEvent fires for every card/link change; onLinkDisabled fires once, the
 * moment this viewer's own link is disabled, so the caller can switch to
 * the not-found view without waiting for a manual refresh.
 */
export function subscribeToPublicBoardEvents(
  token: string,
  onEvent: () => void,
  onLinkDisabled: () => void,
): () => void {
  const source = new EventSource(`${BASE_URL}/api/v1/public/boards/${token}/events`);
  source.onmessage = (e) => {
    let msg: BoardEventMessage;
    try {
      msg = JSON.parse(e.data);
    } catch {
      return;
    }
    if (msg.type === "link.disabled") {
      onLinkDisabled();
      source.close();
      return;
    }
    onEvent();
  };
  return () => source.close();
}
