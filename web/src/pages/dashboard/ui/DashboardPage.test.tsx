import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import DashboardPage from "./DashboardPage";

describe("DashboardPage", () => {
  it("renders the boards heading and the new-board action", () => {
    render(<DashboardPage />);

    expect(screen.getByRole("heading", { name: "Your boards" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /new board/i })).toBeInTheDocument();
  });

  it("shows the empty state when there are no boards", () => {
    render(<DashboardPage />);

    expect(screen.getByText("No boards yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create board/i })).toBeInTheDocument();
  });
});
