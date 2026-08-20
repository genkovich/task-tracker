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

  it("closes the connection on unmount", () => {
    const { unmount } = renderHook(() => useBoardEvents(vi.fn()));
    const source = MockEventSource.instances[0];

    unmount();

    expect(source.closed).toBe(true);
  });
});
