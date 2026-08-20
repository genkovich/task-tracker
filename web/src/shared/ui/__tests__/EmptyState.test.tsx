import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { Building2 } from "lucide-react";
import { EmptyState } from "../EmptyState";

describe("EmptyState", () => {
  it("renders title and icon", () => {
    render(<EmptyState Icon={Building2} title="No groups yet" />);
    expect(screen.getByText("No groups yet")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(
      <EmptyState
        Icon={Building2}
        title="No groups yet"
        description="Create the first group to organise your stream."
      />,
    );
    expect(screen.getByText(/create the first group/i)).toBeInTheDocument();
  });

  it("renders an action when provided", () => {
    render(
      <EmptyState
        Icon={Building2}
        title="No groups yet"
        action={<button type="button">Create group</button>}
      />,
    );
    expect(screen.getByRole("button", { name: /create group/i })).toBeInTheDocument();
  });
});
