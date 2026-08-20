import { describe, it, expect } from "vitest";
import { toZonedDate } from "./date";

describe("toZonedDate", () => {
  // 2024-06-15T12:00:00Z (noon UTC)
  const utcNoon = "2024-06-15T12:00:00Z";

  it("returns correct hours for UTC", () => {
    const result = toZonedDate(utcNoon, "UTC");
    expect(result.getHours()).toBe(12);
    expect(result.getMinutes()).toBe(0);
    expect(result.getDate()).toBe(15);
  });

  it("shifts hours for Europe/Kyiv (UTC+3 in summer)", () => {
    const result = toZonedDate(utcNoon, "Europe/Kyiv");
    expect(result.getHours()).toBe(15);
    expect(result.getMinutes()).toBe(0);
    expect(result.getDate()).toBe(15);
  });

  it("shifts hours for America/New_York (UTC-4 in summer)", () => {
    const result = toZonedDate(utcNoon, "America/New_York");
    expect(result.getHours()).toBe(8);
    expect(result.getMinutes()).toBe(0);
    expect(result.getDate()).toBe(15);
  });

  it("handles date boundary crossing (UTC midnight → next day in positive offset)", () => {
    // 2024-06-15T22:30:00Z → June 16 01:30 in Europe/Kyiv
    const lateUtc = "2024-06-15T22:30:00Z";
    const result = toZonedDate(lateUtc, "Europe/Kyiv");
    expect(result.getDate()).toBe(16);
    expect(result.getHours()).toBe(1);
    expect(result.getMinutes()).toBe(30);
  });

  it("handles date boundary crossing (UTC midnight → previous day in negative offset)", () => {
    // 2024-06-15T02:00:00Z → June 14 22:00 in America/New_York
    const earlyUtc = "2024-06-15T02:00:00Z";
    const result = toZonedDate(earlyUtc, "America/New_York");
    expect(result.getDate()).toBe(14);
    expect(result.getHours()).toBe(22);
  });

  it("accepts Date object as input", () => {
    const date = new Date("2024-06-15T12:00:00Z");
    const result = toZonedDate(date, "UTC");
    expect(result.getHours()).toBe(12);
  });

  it("preserves minutes and seconds", () => {
    const result = toZonedDate("2024-06-15T12:45:30Z", "UTC");
    expect(result.getHours()).toBe(12);
    expect(result.getMinutes()).toBe(45);
    expect(result.getSeconds()).toBe(30);
  });

  it("handles Asia/Kolkata (UTC+5:30 — half-hour offset)", () => {
    const result = toZonedDate(utcNoon, "Asia/Kolkata");
    expect(result.getHours()).toBe(17);
    expect(result.getMinutes()).toBe(30);
  });
});
