import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { TaskCard } from "./TaskCard";
import type { Task } from "@/features/board/api/types";

const task: Task = {
  id: "task-1",
  column_id: "col-1",
  title: "Write the report",
  assignee: null,
  created_at: "2026-08-20T00:00:00Z",
  updated_at: "2026-08-20T00:00:00Z",
};

// Review 2026-08-21 root K: the read-only card (SCR-05, AC-10) still carried
// role="button", tabIndex, a pointer cursor and a drag handler — interactive
// affordances the public viewer must not expose.
describe("TaskCard — read-only rendering (AC-10)", () => {
  it("drops button semantics, focusability, pointer cursor and drag wiring when non-interactive", () => {
    const { container } = render(<TaskCard task={task} draggable={false} />);

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(container.querySelector("[tabindex]")).toBeNull();
    expect(container.querySelector(".cursor-pointer")).toBeNull();

    // Starting a drag must not put the task id into the data transfer.
    const card = screen.getByText("Write the report").closest("div")!;
    const setData = vi.fn();
    fireEvent.dragStart(card, { dataTransfer: { setData } });
    expect(setData).not.toHaveBeenCalled();
  });

  it("keeps button semantics and drag wiring for the editor view", () => {
    render(<TaskCard task={task} onClick={vi.fn()} />);

    expect(screen.getByRole("button")).toBeInTheDocument();
    expect(screen.getByRole("button")).toHaveAttribute("tabindex", "0");
    expect(screen.getByRole("button")).toHaveAttribute("draggable", "true");
  });
});
