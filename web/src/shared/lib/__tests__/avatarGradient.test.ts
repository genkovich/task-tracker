import { describe, expect, it } from "vitest";
import { getAvatarGradient } from "../avatarGradient";

describe("getAvatarGradient", () => {
  it("returns a deterministic gradient for the same input", () => {
    const a = getAvatarGradient("alice@example.com");
    const b = getAvatarGradient("alice@example.com");
    expect(a).toBe(b);
  });

  it("returns different gradients for different inputs", () => {
    const a = getAvatarGradient("alice@example.com");
    const b = getAvatarGradient("bob@example.com");
    expect(a).not.toBe(b);
  });

  it("normalises case and whitespace before hashing", () => {
    const a = getAvatarGradient("Alice@Example.com ");
    const b = getAvatarGradient(" alice@example.com");
    expect(a).toBe(b);
  });

  it("returns a CSS linear-gradient string", () => {
    const g = getAvatarGradient("test@example.com");
    expect(g.startsWith("linear-gradient(135deg, hsl(")).toBe(true);
    expect(g.endsWith("))")).toBe(true);
  });

  it("falls back to a neutral gradient on empty input", () => {
    expect(getAvatarGradient("")).toContain("linear-gradient");
  });
});
