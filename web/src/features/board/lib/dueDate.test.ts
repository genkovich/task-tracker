import { describe, it, expect } from "vitest";
import { formatDueDate, isOverdue, todayISO } from "./dueDate";

describe("todayISO", () => {
  it("uses the viewer's own calendar day, not UTC", () => {
    // 23:30 local on the 1st is still the 1st, whatever UTC says.
    const lateEvening = new Date(2026, 8, 1, 23, 30, 0);
    expect(todayISO(lateEvening)).toBe("2026-09-01");
  });

  it("zero-pads month and day", () => {
    expect(todayISO(new Date(2026, 0, 5, 12, 0, 0))).toBe("2026-01-05");
  });
});

// TSK-06: a deadline strictly before today is overdue; today itself is not.
describe("isOverdue", () => {
  const now = new Date(2026, 8, 10, 12, 0, 0); // 2026-09-10

  it("flags a day in the past", () => {
    expect(isOverdue("2026-09-09", now)).toBe(true);
    expect(isOverdue("2025-12-31", now)).toBe(true);
  });

  it("does not flag today or the future", () => {
    expect(isOverdue("2026-09-10", now)).toBe(false);
    expect(isOverdue("2026-09-11", now)).toBe(false);
    expect(isOverdue("2027-01-01", now)).toBe(false);
  });
});

describe("formatDueDate", () => {
  const now = new Date(2026, 8, 10, 12, 0, 0);

  it("drops the year when the deadline is this year", () => {
    expect(formatDueDate("2026-09-01", now)).toBe("1 вер");
    expect(formatDueDate("2026-01-15", now)).toBe("15 січ");
  });

  it("keeps the year when it differs — otherwise a 2027 deadline reads as this year", () => {
    expect(formatDueDate("2027-03-02", now)).toBe("2 бер 2027");
  });

  it("returns the raw value it cannot parse rather than inventing a date", () => {
    expect(formatDueDate("not-a-date", now)).toBe("not-a-date");
  });
});
