import { describe, expect, it } from "vitest";
import { formatTimeForUser, getBrowserTimezone, tzShortLabel } from "../tz";

describe("tzShortLabel", () => {
  it("returns the last path segment", () => {
    expect(tzShortLabel("Europe/Kyiv")).toBe("Kyiv");
    expect(tzShortLabel("America/Los_Angeles")).toBe("Los Angeles");
    expect(tzShortLabel("UTC")).toBe("UTC");
  });
});

describe("formatTimeForUser", () => {
  it("formats in the user's timezone when set", () => {
    const date = new Date("2026-06-04T14:00:00Z");
    expect(
      formatTimeForUser(date, "yyyy-MM-dd HH:mm", "Europe/Kyiv", { browserTz: "Europe/Kyiv" }),
    ).toBe("2026-06-04 17:00");
  });

  it("appends the short label when user TZ differs from browser TZ", () => {
    const date = new Date("2026-06-04T14:00:00Z");
    expect(formatTimeForUser(date, "HH:mm", "Europe/Kyiv", { browserTz: "America/Los_Angeles" })).toBe(
      "17:00 (Kyiv)",
    );
  });

  it("does not append the label when user TZ matches browser TZ", () => {
    const date = new Date("2026-06-04T14:00:00Z");
    expect(formatTimeForUser(date, "HH:mm", "Europe/Kyiv", { browserTz: "Europe/Kyiv" })).toBe(
      "17:00",
    );
  });

  it("falls back to browser TZ when user TZ is null", () => {
    const date = new Date("2026-06-04T14:00:00Z");
    expect(formatTimeForUser(date, "HH:mm", null, { browserTz: "Europe/Berlin" })).toBe("16:00");
  });
});

describe("getBrowserTimezone", () => {
  it("returns a timezone string", () => {
    const tz = getBrowserTimezone();
    expect(typeof tz).toBe("string");
  });
});
