import { useEffect } from "react";
import { BASE_URL } from "@/shared/api/client";

/**
 * Subscribes to the board's SSE event stream (`GET /api/v1/board/events`,
 * ADR-0002) and calls `onStateChanged` whenever a `board.state_changed`
 * signal arrives, so the caller can refetch the board state.
 */
export function useBoardEvents(onStateChanged: () => void): void {
  useEffect(() => {
    const source = new EventSource(`${BASE_URL}/api/v1/board/events`);

    const handleStateChanged = () => onStateChanged();
    source.addEventListener("board.state_changed", handleStateChanged);

    return () => {
      source.removeEventListener("board.state_changed", handleStateChanged);
      source.close();
    };
  }, [onStateChanged]);
}
