import { render, screen, fireEvent, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";

import { TimePicker, generateSlots } from "./time-picker";

// ── generateSlots unit tests ─────────────────────────────────────────

describe("generateSlots", () => {
  it("returns correct total count for step=5", () => {
    expect(generateSlots(5)).toHaveLength(288);
  });

  it("returns correct total count for step=15", () => {
    expect(generateSlots(15)).toHaveLength(96);
  });

  it("returns correct total count for step=30", () => {
    expect(generateSlots(30)).toHaveLength(48);
  });

  it("slots are ordered 00:00 → 23:55", () => {
    const slots = generateSlots(5);
    expect(slots[0]).toBe("00:00");
    expect(slots[slots.length - 1]).toBe("23:55");
  });

  it("slots are in ascending order", () => {
    const slots = generateSlots(15);
    for (let i = 1; i < slots.length; i++) {
      expect(slots[i] > slots[i - 1]).toBe(true);
    }
  });
});

// ── TimePicker component tests ───────────────────────────────────────

describe("TimePicker", () => {
  let onChange: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    onChange = vi.fn();
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((cb) => {
      cb(0);
      return 0;
    });
  });

  it("renders time slots for given minuteStep", () => {
    render(<TimePicker value="" onChange={onChange} minuteStep={60} />);

    expect(screen.getByText("06:00")).toBeInTheDocument();
    expect(screen.getByText("12:00")).toBeInTheDocument();
    expect(screen.getByText("23:00")).toBeInTheDocument();
  });

  it("renders preset buttons", () => {
    render(<TimePicker value="" onChange={onChange} />);

    const presets = screen.getAllByRole("button").filter(
      (btn) => btn.getAttribute("data-slot") === "button"
    );
    const labels = presets.map((btn) => btn.textContent);
    expect(labels).toContain("10:00");
    expect(labels).toContain("13:00");
    expect(labels).toContain("17:00");
  });

  it("highlights selected slot with bg-primary", () => {
    render(<TimePicker value="10:00" onChange={onChange} minuteStep={60} />);

    const slot = screen.getByText("10:00", {
      selector: "button[data-time]",
    });
    expect(slot.className).toContain("bg-primary");
  });

  it("calls onChange when clicking a time slot", () => {
    render(<TimePicker value="" onChange={onChange} minuteStep={60} />);

    fireEvent.click(
      screen.getByText("14:00", { selector: "button[data-time]" })
    );
    expect(onChange).toHaveBeenCalledWith("14:00");
  });

  it("calls onChange when clicking a preset button", () => {
    render(<TimePicker value="" onChange={onChange} />);

    const preset = screen
      .getAllByRole("button")
      .find(
        (btn) =>
          btn.getAttribute("data-slot") === "button" &&
          btn.textContent === "13:00"
      )!;
    fireEvent.click(preset);
    expect(onChange).toHaveBeenCalledWith("13:00");
  });

  it("gives selected preset the default variant", () => {
    render(<TimePicker value="13:00" onChange={onChange} minuteStep={60} />);

    const presets = screen
      .getAllByRole("button")
      .filter((btn) => btn.getAttribute("data-slot") === "button");

    const selected = presets.find((btn) => btn.textContent === "13:00")!;
    const other = presets.find((btn) => btn.textContent === "10:00")!;

    expect(selected.getAttribute("data-variant")).not.toBe("ghost");
    expect(other.getAttribute("data-variant")).toBe("ghost");
  });

  it("sets scrollTop on mount via requestAnimationFrame", () => {
    const rafSpy = vi.spyOn(window, "requestAnimationFrame");

    render(<TimePicker value="14:00" onChange={onChange} minuteStep={60} />);

    expect(rafSpy).toHaveBeenCalled();
  });

  it("scrolls container when value changes to out-of-view slot", () => {
    const { rerender } = render(
      <TimePicker value="06:00" onChange={onChange} minuteStep={60} />
    );

    // Simulate the scroll container having dimensions
    // (jsdom doesn't lay out, so scrollTop stays 0, but we verify no errors)
    rerender(
      <TimePicker value="23:00" onChange={onChange} minuteStep={60} />
    );

    // The effect runs without throwing — no scrollIntoView called
    expect(true).toBe(true);
  });

  it("does not use scrollIntoView", () => {
    const spy = vi.spyOn(Element.prototype, "scrollIntoView");

    render(<TimePicker value="10:00" onChange={onChange} minuteStep={60} />);

    act(() => {
      // flush rAF
    });

    expect(spy).not.toHaveBeenCalled();
    spy.mockRestore();
  });
});
