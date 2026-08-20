import { describe, expect, it } from "vitest";
import { getDisplayName, getInitials } from "../user";

describe("getDisplayName", () => {
  it("joins first and last name", () => {
    expect(
      getDisplayName({ first_name: "Jane", last_name: "Doe", email: "jane@example.com" }),
    ).toBe("Jane Doe");
  });

  it("uses first name only when last name missing", () => {
    expect(
      getDisplayName({ first_name: "Jane", last_name: null, email: "jane@example.com" }),
    ).toBe("Jane");
  });

  it("falls back to email when no name", () => {
    expect(
      getDisplayName({ first_name: null, last_name: null, email: "jane@example.com" }),
    ).toBe("jane@example.com");
  });
});

describe("getInitials", () => {
  it("returns two letters when both names exist", () => {
    expect(getInitials({ first_name: "Jane", last_name: "Doe", email: "x@example.com" })).toBe(
      "JD",
    );
  });

  it("returns first-name initial when last name missing", () => {
    expect(getInitials({ first_name: "Jane", last_name: null, email: "x@example.com" })).toBe(
      "J",
    );
  });

  it("falls back to email initial", () => {
    expect(getInitials({ first_name: null, last_name: null, email: "alice@example.com" })).toBe(
      "A",
    );
  });
});
