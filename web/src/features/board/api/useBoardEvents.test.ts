import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { BASE_URL } from "@/shared/api/client";
import { useBoardEvents } from "./useBoardEvents";

type Listener = (ev: MessageEvent) => void;

class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  closed = false;
  listeners = new Map<string, Listener[]>();
  onopen: ((ev: Event) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: Listener) {
    const existing = this.listeners.get(type) ?? [];
    existing.push(listener);
    this.listeners.set(type, existing);
  }

  removeEventListener(type: string, listener: Listener) {
    const existing = this.listeners.get(type) ?? [];
    this.listeners.set(
      type,
      existing.filter((l) => l !== listener),
    );
  }

  close() {
    this.closed = true;
  }

  emit(type: string, data = "") {
    const messageEvent = { type, data } as MessageEvent;
    for (const listener of this.listeners.get(type) ?? []) {
      listener(messageEvent);
    }
  }
}

beforeEach(() => {
  MockEventSource.instances = [];
  vi.stubGlobal("EventSource", MockEventSource);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("useBoardEvents", () => {
  it("opens an SSE connection to /api/v1/board/events", () => {
    renderHook(() => useBoardEvents(vi.fn()));

    expect(MockEventSource.instances).toHaveLength(1);
    expect(MockEventSource.instances[0].url).toBe(`${BASE_URL}/api/v1/board/events`);
  });

  it("triggers exactly one refetch when a board.state_changed message arrives", () => {
    const onStateChanged = vi.fn();
    renderHook(() => useBoardEvents(onStateChanged));

    const source = MockEventSource.instances[0];
    source.emit("board.state_changed");

    expect(onStateChanged).toHaveBeenCalledTimes(1);
  });

  it("does not refetch for unrelated event types", () => {
    const onStateChanged = vi.fn();
    renderHook(() => useBoardEvents(onStateChanged));

    const source = MockEventSource.instances[0];
    source.emit("message");
    source.emit("some.other.event");

    expect(onStateChanged).not.toHaveBeenCalled();
  });

  // Review 2026-08-21 root C — contracts/events.md "Retry / reconnect": the
  // browser's EventSource auto-reconnects, and events missed while
  // disconnected are recovered by a fresh GET. The hook must refetch when a
  // connection (re)opens and once on error, or a reconnect silently shows
  // stale state forever.
  it("refetches when the connection reopens after an error (reconnect recovery)", () => {
    const onStateChanged = vi.fn();
    renderHook(() => useBoardEvents(onStateChanged));

    const source = MockEventSource.instances[0];
    source.onopen?.(new Event("open"));
    onStateChanged.mockClear();

    source.onerror?.(new Event("error"));
    source.onopen?.(new Event("open"));

    expect(onStateChanged).toHaveBeenCalledTimes(2);
  });

  it("refetches only once across repeated errors while the server stays down", () => {
    const onStateChanged = vi.fn();
    renderHook(() => useBoardEvents(onStateChanged));

    const source = MockEventSource.instances[0];
    source.onopen?.(new Event("open"));
    onStateChanged.mockClear();

    // EventSource retries by itself; each failed attempt fires onerror. The
    // hook must not turn that retry loop into a refetch loop.
    source.onerror?.(new Event("error"));
    source.onerror?.(new Event("error"));
    source.onerror?.(new Event("error"));

    expect(onStateChanged).toHaveBeenCalledTimes(1);
  });

  it("closes the connection on unmount", () => {
    const { unmount } = renderHook(() => useBoardEvents(vi.fn()));
    const source = MockEventSource.instances[0];

    unmount();

    expect(source.closed).toBe(true);
  });
});
