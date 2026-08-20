import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";

import { Stepper } from "./stepper";
import type { StepConfig } from "./types";

const steps: StepConfig[] = [
  { id: "org", title: "Stream" },
  { id: "team", title: "Group" },
  { id: "invite", title: "Invite", description: "Optional" },
];

describe("Stepper", () => {
  it("renders all step titles", () => {
    render(<Stepper steps={steps} currentStep={0} />);

    expect(screen.getByText("Stream")).toBeInTheDocument();
    expect(screen.getByText("Group")).toBeInTheDocument();
    expect(screen.getByText("Invite")).toBeInTheDocument();
  });

  it("renders step description when provided", () => {
    render(<Stepper steps={steps} currentStep={2} />);

    expect(screen.getByText("Optional")).toBeInTheDocument();
  });

  it("marks completed steps with checkmark", () => {
    render(<Stepper steps={steps} currentStep={2} />);

    const completedSteps = screen.getAllByTestId("step-completed");
    expect(completedSteps).toHaveLength(2);
  });

  it("marks current step as active", () => {
    render(<Stepper steps={steps} currentStep={1} />);

    const activeStep = screen.getByTestId("step-active");
    expect(activeStep).toBeInTheDocument();
  });

  it("marks future steps as upcoming", () => {
    render(<Stepper steps={steps} currentStep={0} />);

    const upcomingSteps = screen.getAllByTestId("step-upcoming");
    expect(upcomingSteps).toHaveLength(2);
  });

  it("shows step numbers for upcoming steps", () => {
    render(<Stepper steps={steps} currentStep={0} />);

    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("shows step number for active step", () => {
    render(<Stepper steps={steps} currentStep={0} />);

    const activeStep = screen.getByTestId("step-active");
    expect(activeStep).toHaveTextContent("1");
  });

  it("renders connecting lines between steps", () => {
    render(<Stepper steps={steps} currentStep={1} />);

    const connectors = screen.getAllByTestId("step-connector");
    expect(connectors).toHaveLength(2);
  });

  it("marks connector as completed when preceding step is done", () => {
    render(<Stepper steps={steps} currentStep={2} />);

    const connectors = screen.getAllByTestId("step-connector");
    const completedConnectors = connectors.filter(
      (c) => c.getAttribute("data-completed") === "true",
    );
    expect(completedConnectors).toHaveLength(2);
  });

  it("handles single step", () => {
    const singleStep: StepConfig[] = [{ id: "only", title: "Single step" }];
    render(<Stepper steps={singleStep} currentStep={0} />);

    expect(screen.getByText("Single step")).toBeInTheDocument();
    expect(screen.queryAllByTestId("step-connector")).toHaveLength(0);
  });

  it("handles all steps completed (currentStep >= steps.length)", () => {
    render(<Stepper steps={steps} currentStep={3} />);

    const completedSteps = screen.getAllByTestId("step-completed");
    expect(completedSteps).toHaveLength(3);
  });

  it("handles negative currentStep (all upcoming)", () => {
    render(<Stepper steps={steps} currentStep={-1} />);

    const upcomingSteps = screen.getAllByTestId("step-upcoming");
    expect(upcomingSteps).toHaveLength(3);
    expect(screen.queryByTestId("step-active")).not.toBeInTheDocument();
    expect(screen.queryByTestId("step-completed")).not.toBeInTheDocument();
  });

  it("applies custom className", () => {
    const { container } = render(
      <Stepper steps={steps} currentStep={0} className="my-custom-class" />,
    );

    expect(container.firstChild).toHaveClass("my-custom-class");
  });

  // ── Accessibility ──────────────────────────────────────────────────

  it("has role=list on container and role=listitem on steps", () => {
    render(<Stepper steps={steps} currentStep={0} />);

    expect(screen.getByRole("list")).toBeInTheDocument();
    expect(screen.getAllByRole("listitem")).toHaveLength(3);
  });

  it("has aria-label on container", () => {
    render(<Stepper steps={steps} currentStep={0} />);

    expect(screen.getByRole("list")).toHaveAttribute("aria-label");
  });

  it("has aria-label on step indicators", () => {
    render(<Stepper steps={steps} currentStep={1} />);

    const completed = screen.getByTestId("step-completed");
    expect(completed).toHaveAttribute("aria-label");
    expect(completed.getAttribute("aria-label")).toContain("completed");

    const active = screen.getByTestId("step-active");
    expect(active).toHaveAttribute("aria-label");
    expect(active.getAttribute("aria-label")).toContain("active");

    const upcoming = screen.getByTestId("step-upcoming");
    expect(upcoming).toHaveAttribute("aria-label");
    expect(upcoming.getAttribute("aria-label")).toContain("pending");
  });

  it("has aria-hidden on connectors", () => {
    render(<Stepper steps={steps} currentStep={1} />);

    const connectors = screen.getAllByTestId("step-connector");
    connectors.forEach((c) => {
      expect(c).toHaveAttribute("aria-hidden", "true");
    });
  });

  it("has data-slot=stepper on root", () => {
    const { container } = render(<Stepper steps={steps} currentStep={0} />);

    expect(container.firstChild).toHaveAttribute("data-slot", "stepper");
  });
});
