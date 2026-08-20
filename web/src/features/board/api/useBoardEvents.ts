import { useEffect } from "react";
import { BASE_URL } from "@/shared/api/client";

/**
 * Subscribes to the board's SSE event stream (`GET /api/v1/board/events`,
 * ADR-0002) and calls `onStateChanged` whenever a `board.state_changed`
 * signal arrives, so the caller can refetch the board state.
 */
export function useBoardEvents(onStateChanged: () => void): void {
  useBoardEventsAt(`${BASE_URL}/api/v1/board/events`, onStateChanged);
}

/**
 * Public-viewer counterpart of `useBoardEvents` (AC-09) — subscribes to
 * `GET /api/v1/public/{token}/events`, the token-scoped SSE channel closed
 * synchronously by the server when the link is revoked (AC-11, ADR-0002).
 */
export function usePublicBoardEvents(token: string, onStateChanged: () => void): void {
  useBoardEventsAt(`${BASE_URL}/api/v1/public/${token}/events`, onStateChanged);
}

function useBoardEventsAt(url: string, onStateChanged: () => void): void {
  useEffect(() => {
    const source = new EventSource(url);

    const handleStateChanged = () => onStateChanged();
    source.addEventListener("board.state_changed", handleStateChanged);

    return () => {
      source.removeEventListener("board.state_changed", handleStateChanged);
      source.close();
    };
  }, [url, onStateChanged]);
}
