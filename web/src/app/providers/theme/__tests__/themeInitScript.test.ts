import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { applyStoredTheme, themeInitScript } from "../themeInitScript";

const STORAGE_KEY = "task-tracker-theme";

function resetDom() {
  document.documentElement.className = "";
  document.documentElement.style.colorScheme = "";
}

beforeEach(() => {
  window.localStorage.clear();
  resetDom();
});

afterEach(() => {
  resetDom();
});

describe("applyStoredTheme", () => {
  it("adds .dark when stored theme is dark", () => {
    window.localStorage.setItem(STORAGE_KEY, "dark");

    applyStoredTheme();

    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe("dark");
  });

  it("does not add .dark when stored theme is light", () => {
    window.localStorage.setItem(STORAGE_KEY, "light");

    applyStoredTheme();

    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(document.documentElement.style.colorScheme).toBe("light");
  });

  it("falls back to system preference when stored theme is missing", () => {
    const original = window.matchMedia;
    window.matchMedia = (() => ({
      matches: true,
      media: "(prefers-color-scheme: dark)",
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
      onchange: null,
    })) as unknown as typeof window.matchMedia;

    applyStoredTheme();

    expect(document.documentElement.classList.contains("dark")).toBe(true);

    window.matchMedia = original;
  });

  it("ignores invalid stored values", () => {
    window.localStorage.setItem(STORAGE_KEY, "neon");
    const original = window.matchMedia;
    window.matchMedia = (() => ({
      matches: false,
      media: "(prefers-color-scheme: dark)",
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
      onchange: null,
    })) as unknown as typeof window.matchMedia;

    applyStoredTheme();

    expect(document.documentElement.classList.contains("dark")).toBe(false);
    window.matchMedia = original;
  });

  it("never throws even when storage access is denied", () => {
    const original = Object.getOwnPropertyDescriptor(window, "localStorage");
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      get() {
        throw new Error("denied");
      },
    });

    expect(() => applyStoredTheme()).not.toThrow();

    if (original) Object.defineProperty(window, "localStorage", original);
  });
});

describe("themeInitScript", () => {
  it("contains the IIFE wrapper and storage key", () => {
    expect(themeInitScript).toMatch(/^\(/);
    expect(themeInitScript).toContain("task-tracker-theme");
    expect(themeInitScript).toContain("classList.toggle");
  });
});
