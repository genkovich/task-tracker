import "@testing-library/jest-dom/vitest";

// Polyfills for jsdom (required by cmdk)
globalThis.ResizeObserver ??= class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

Element.prototype.scrollIntoView ??= function () {};
Element.prototype.hasPointerCapture ??= function () { return false; };
Element.prototype.setPointerCapture ??= function () {};
Element.prototype.releasePointerCapture ??= function () {};
