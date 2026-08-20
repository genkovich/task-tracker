import { act, render, renderHook, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider, useTheme, type Theme } from "../ThemeContext";

const STORAGE_KEY = "task-tracker-theme";

type MediaListener = (event: MediaQueryListEvent) => void;

interface MockMediaQueryList {
  matches: boolean;
  media: string;
  onchange: MediaListener | null;
  addEventListener: ReturnType<typeof vi.fn>;
  removeEventListener: ReturnType<typeof vi.fn>;
  addListener: ReturnType<typeof vi.fn>;
  removeListener: ReturnType<typeof vi.fn>;
  dispatchEvent: ReturnType<typeof vi.fn>;
}

function setupMatchMedia(initialMatches: boolean) {
  const listeners = new Set<MediaListener>();
  const mql: MockMediaQueryList = {
    matches: initialMatches,
    media: "(prefers-color-scheme: dark)",
    onchange: null,
    addEventListener: vi.fn((_: string, l: MediaListener) => listeners.add(l)),
    removeEventListener: vi.fn((_: string, l: MediaListener) => listeners.delete(l)),
    addListener: vi.fn((l: MediaListener) => listeners.add(l)),
    removeListener: vi.fn((l: MediaListener) => listeners.delete(l)),
    dispatchEvent: vi.fn(),
  };

  window.matchMedia = vi.fn().mockReturnValue(mql);

  return {
    mql,
    fire: (matches: boolean) => {
      mql.matches = matches;
      const event = { matches } as MediaQueryListEvent;
      for (const l of listeners) l(event);
    },
  };
}

beforeEach(() => {
  window.localStorage.clear();
  document.documentElement.className = "";
  document.documentElement.style.colorScheme = "";
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("ThemeProvider", () => {
  it("defaults to system theme and resolves from prefers-color-scheme", () => {
    setupMatchMedia(true);

    const { result } = renderHook(() => useTheme(), { wrapper: ThemeProvider });

    expect(result.current.theme).toBe("system");
    expect(result.current.resolvedTheme).toBe("dark");
  });

  it("applies dark class to <html> when resolved theme is dark", () => {
    setupMatchMedia(true);

    render(
      <ThemeProvider>
        <span>hello</span>
      </ThemeProvider>,
    );

    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe("dark");
  });

  it("reads stored preference from localStorage", () => {
    window.localStorage.setItem(STORAGE_KEY, "dark");
    setupMatchMedia(false);

    const { result } = renderHook(() => useTheme(), { wrapper: ThemeProvider });

    expect(result.current.theme).toBe("dark");
    expect(result.current.resolvedTheme).toBe("dark");
  });

  it("persists theme changes to localStorage and updates DOM", () => {
    setupMatchMedia(false);

    const { result } = renderHook(() => useTheme(), { wrapper: ThemeProvider });

    expect(document.documentElement.classList.contains("dark")).toBe(false);

    act(() => result.current.setTheme("dark"));

    expect(window.localStorage.getItem(STORAGE_KEY)).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("reacts to system theme change when theme is 'system'", () => {
    const { fire } = setupMatchMedia(false);

    const { result } = renderHook(() => useTheme(), { wrapper: ThemeProvider });
    expect(result.current.resolvedTheme).toBe("light");

    act(() => fire(true));

    expect(result.current.resolvedTheme).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("ignores invalid stored values and falls back to system", () => {
    window.localStorage.setItem(STORAGE_KEY, "neon");
    setupMatchMedia(false);

    const { result } = renderHook(() => useTheme(), { wrapper: ThemeProvider });

    expect(result.current.theme).toBe("system");
  });
});

describe("useTheme outside provider", () => {
  it("returns fallback context without throwing", () => {
    const { result } = renderHook(() => useTheme());

    expect(result.current.theme).toBe("system");
    expect(result.current.resolvedTheme).toBe("light");
    expect(typeof result.current.setTheme).toBe("function");
    expect(() => result.current.setTheme("dark" satisfies Theme)).not.toThrow();
  });

  it("does not render content twice when used as a child of nothing", () => {
    render(<span>{useTheme.name}</span>);
    expect(screen.getByText("useTheme")).toBeInTheDocument();
  });
});
