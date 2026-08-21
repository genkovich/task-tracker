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

// A pointer gesture past DRAG_THRESHOLD_PX (6) — pointerdown at the origin,
// then a move far enough to activate the drag.
function dragPast(card: HTMLElement, to = { x: 0, y: 40 }) {
  fireEvent.pointerDown(card, { clientX: 0, clientY: 0, pointerId: 1 });
  fireEvent.pointerMove(card, { clientX: to.x, clientY: to.y, pointerId: 1 });
}

// Review 2026-08-21 root K: the read-only card (SCR-05, AC-10) still carried
// role="button", tabIndex, a pointer cursor and a drag handler — interactive
// affordances the public viewer must not expose.
describe("TaskCard — read-only rendering (AC-10)", () => {
  it("drops button semantics, focusability, pointer cursor and drag wiring when non-interactive", () => {
    const onDragStart = vi.fn();
    const { container } = render(
      <TaskCard task={task} draggable={false} onDragStart={onDragStart} />,
    );

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(container.querySelector("[tabindex]")).toBeNull();
    expect(container.querySelector(".cursor-pointer")).toBeNull();
    // No touch-action lock either — a read-only card must not block scroll.
    expect(container.querySelector(".touch-none")).toBeNull();

    // A full drag gesture must not start a drag on a read-only card.
    const card = container.querySelector('[data-slot="card"]')!;
    dragPast(card as HTMLElement);
    expect(onDragStart).not.toHaveBeenCalled();
  });

  it("keeps button semantics and drag wiring for the editor view", () => {
    render(<TaskCard task={task} onClick={vi.fn()} />);

    expect(screen.getByRole("button")).toBeInTheDocument();
    expect(screen.getByRole("button")).toHaveAttribute("tabindex", "0");
    // The pointer-drag affordance: touch-action locked so touch moves drag
    // the card instead of scrolling the page.
    expect(screen.getByRole("button")).toHaveClass("touch-none");
  });
});

describe("TaskCard — pointer drag gesture", () => {
  it("does not start a drag below the movement threshold", () => {
    const onDragStart = vi.fn();
    render(<TaskCard task={task} onClick={vi.fn()} onDragStart={onDragStart} />);
    const card = screen.getByRole("button");

    fireEvent.pointerDown(card, { clientX: 0, clientY: 0, pointerId: 1 });
    fireEvent.pointerMove(card, { clientX: 2, clientY: 3, pointerId: 1 });

    expect(onDragStart).not.toHaveBeenCalled();
  });

  it("starts the drag once past the threshold and reports moves and the drop", () => {
    const onDragStart = vi.fn();
    const onDragMove = vi.fn();
    const onDragEnd = vi.fn();
    render(
      <TaskCard
        task={task}
        onClick={vi.fn()}
        onDragStart={onDragStart}
        onDragMove={onDragMove}
        onDragEnd={onDragEnd}
      />,
    );
    const card = screen.getByRole("button");

    dragPast(card);
    expect(onDragStart).toHaveBeenCalledTimes(1);
    expect(onDragStart).toHaveBeenCalledWith(task);
    expect(onDragMove).toHaveBeenCalledWith(0, 40);

    fireEvent.pointerMove(card, { clientX: 10, clientY: 80, pointerId: 1 });
    expect(onDragStart).toHaveBeenCalledTimes(1);
    expect(onDragMove).toHaveBeenLastCalledWith(10, 80);

    fireEvent.pointerUp(card, { clientX: 10, clientY: 80, pointerId: 1 });
    expect(onDragEnd).toHaveBeenCalledTimes(1);
  });

  it("opens via onClick on a tap (no movement), but not after a drag", () => {
    const onClick = vi.fn();
    const onDragEnd = vi.fn();
    render(<TaskCard task={task} onClick={onClick} onDragEnd={onDragEnd} />);
    const card = screen.getByRole("button");

    // A plain tap: pointerdown + pointerup with no meaningful movement.
    fireEvent.pointerDown(card, { clientX: 0, clientY: 0, pointerId: 1 });
    fireEvent.pointerUp(card, { clientX: 1, clientY: 1, pointerId: 1 });
    fireEvent.click(card);
    expect(onClick).toHaveBeenCalledWith(task);
    expect(onDragEnd).not.toHaveBeenCalled();

    // A drag: the synthesized click after pointerup must not open the modal.
    onClick.mockClear();
    dragPast(card);
    fireEvent.pointerUp(card, { clientX: 0, clientY: 40, pointerId: 1 });
    fireEvent.click(card);
    expect(onClick).not.toHaveBeenCalled();
    expect(onDragEnd).toHaveBeenCalledTimes(1);
  });

  it("reports a cancel (not a drop) on pointercancel", () => {
    const onDragEnd = vi.fn();
    const onDragCancel = vi.fn();
    render(
      <TaskCard task={task} onClick={vi.fn()} onDragEnd={onDragEnd} onDragCancel={onDragCancel} />,
    );
    const card = screen.getByRole("button");

    dragPast(card);
    fireEvent.pointerCancel(card, { pointerId: 1 });

    expect(onDragCancel).toHaveBeenCalledTimes(1);
    expect(onDragEnd).not.toHaveBeenCalled();
  });

  it("aborts the gesture when the dragging prop flips false mid-drag (Escape)", () => {
    const onDragStart = vi.fn();
    const onDragMove = vi.fn();
    const onDragEnd = vi.fn();
    const onClick = vi.fn();
    const { rerender } = render(
      <TaskCard
        task={task}
        onClick={onClick}
        dragging={false}
        onDragStart={onDragStart}
        onDragMove={onDragMove}
        onDragEnd={onDragEnd}
      />,
    );
    const card = screen.getByRole("button");

    dragPast(card);
    expect(onDragStart).toHaveBeenCalledTimes(1);
    // The page wiring flips `dragging` on via useBoardDnd state…
    rerender(
      <TaskCard
        task={task}
        onClick={onClick}
        dragging
        onDragStart={onDragStart}
        onDragMove={onDragMove}
        onDragEnd={onDragEnd}
      />,
    );
    // …and Escape flips it back off while the pointer is still down.
    rerender(
      <TaskCard
        task={task}
        onClick={onClick}
        dragging={false}
        onDragStart={onDragStart}
        onDragMove={onDragMove}
        onDragEnd={onDragEnd}
      />,
    );

    onDragMove.mockClear();
    fireEvent.pointerMove(card, { clientX: 0, clientY: 120, pointerId: 1 });
    expect(onDragMove).not.toHaveBeenCalled();
    expect(onDragStart).toHaveBeenCalledTimes(1); // no re-activation either

    // Releasing after the abort is neither a drop nor a click.
    fireEvent.pointerUp(card, { clientX: 0, clientY: 120, pointerId: 1 });
    fireEvent.click(card);
    expect(onDragEnd).not.toHaveBeenCalled();
    expect(onClick).not.toHaveBeenCalled();
  });
});
