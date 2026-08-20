import { describe, expect, it } from "vitest";
import { isSafeUrl } from "../isSafeUrl";

describe("isSafeUrl", () => {
  it("accepts http URLs", () => {
    expect(isSafeUrl("http://example.com")).toBe(true);
    expect(isSafeUrl("http://example.com/path?q=1")).toBe(true);
  });

  it("accepts https URLs", () => {
    expect(isSafeUrl("https://example.com")).toBe(true);
    expect(isSafeUrl("https://meet.google.com/abc-def")).toBe(true);
  });

  it("accepts mailto URLs", () => {
    expect(isSafeUrl("mailto:user@example.com")).toBe(true);
  });

  it("rejects javascript: URLs", () => {
    expect(isSafeUrl("javascript:alert(1)")).toBe(false);
    expect(isSafeUrl("JavaScript:alert(1)")).toBe(false);
    expect(isSafeUrl("javascript:void(0)")).toBe(false);
  });

  it("rejects data: URLs", () => {
    expect(isSafeUrl("data:text/html,<script>alert(1)</script>")).toBe(false);
    expect(isSafeUrl("data:image/png;base64,abc")).toBe(false);
  });

  it("rejects vbscript: URLs", () => {
    expect(isSafeUrl("vbscript:msgbox")).toBe(false);
  });

  it("rejects empty and malformed strings", () => {
    expect(isSafeUrl("")).toBe(false);
    expect(isSafeUrl("not a url")).toBe(false);
    expect(isSafeUrl("   ")).toBe(false);
  });

  it("rejects relative paths", () => {
    expect(isSafeUrl("/path/to/page")).toBe(false);
    expect(isSafeUrl("../parent")).toBe(false);
  });

  it("rejects ftp: URLs", () => {
    expect(isSafeUrl("ftp://files.example.com")).toBe(false);
  });
});
