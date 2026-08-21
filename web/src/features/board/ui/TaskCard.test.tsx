import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { TaskCard } from "./TaskCard";
import type { Task } from "@/features/board/api/types";

const task: Task = {
  id: "task-1",
  column_id: "col-1",
  title: "Write the report",
  assignee: null,
  priority: "medium",
  due_date: null,
  has_description: false,
  comment_count: 0,
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

// tasks T10 — the card's markers (TSK-03/TSK-05/TSK-06/TSK-08). The card
// shows markers, never content: no description text and no comment text ever
// reaches a column.
describe("TaskCard — markers", () => {
  const card = (overrides: Partial<Task> = {}): Task => ({ ...task, ...overrides });

  it("draws a distinct priority marker for each priority (TSK-03)", () => {
    const seen = new Set<string>();

    for (const priority of ["low", "medium", "high"] as const) {
      const { container, unmount } = render(<TaskCard task={card({ priority })} />);
      const dot = container.querySelector(`[data-priority="${priority}"]`);
      expect(dot).not.toBeNull();
      seen.add(dot!.className);
      unmount();
    }

    expect(seen.size).toBe(3);
  });

  it("shows a due-date badge, marked overdue only for a past deadline (TSK-05/TSK-06)", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 8, 10, 12, 0, 0)); // 2026-09-10

    const future = render(<TaskCard task={card({ due_date: "2026-09-20" })} />);
    const futureBadge = future.container.querySelector('[data-slot="due-badge"]')!;
    expect(futureBadge).toHaveTextContent("20 вер");
    expect(futureBadge.getAttribute("data-overdue")).toBeNull();
    expect(futureBadge.className).not.toContain("destructive");
    future.unmount();

    const past = render(<TaskCard task={card({ due_date: "2026-09-01" })} />);
    const pastBadge = past.container.querySelector('[data-slot="due-badge"]')!;
    expect(pastBadge.getAttribute("data-overdue")).toBe("true");
    expect(pastBadge.className).toContain("destructive");
    past.unmount();

    vi.useRealTimers();
  });

  it("omits the due-date badge when there is no deadline", () => {
    const { container } = render(<TaskCard task={card({ due_date: null })} />);
    expect(container.querySelector('[data-slot="due-badge"]')).toBeNull();
  });

  it("shows the description marker only when the task has one (TSK-01)", () => {
    const without = render(<TaskCard task={card({ has_description: false })} />);
    expect(without.container.querySelector('[data-slot="has-description"]')).toBeNull();
    without.unmount();

    const withDescription = render(<TaskCard task={card({ has_description: true })} />);
    expect(withDescription.container.querySelector('[data-slot="has-description"]')).not.toBeNull();
  });

  it("shows the comment count only when there are comments (TSK-08)", () => {
    const none = render(<TaskCard task={card({ comment_count: 0 })} />);
    expect(none.container.querySelector('[data-slot="comment-count"]')).toBeNull();
    none.unmount();

    const some = render(<TaskCard task={card({ comment_count: 3 })} />);
    expect(some.container.querySelector('[data-slot="comment-count"]')).toHaveTextContent("3");
  });

  // The ghost is a clone of the same visual, so a marker added to one must
  // appear in the other — that is the whole point of sharing TaskCardVisual.
  it("carries the markers into the drag ghost too", () => {
    const { container } = render(
      <TaskCard
        task={card({ priority: "high", comment_count: 2, has_description: true })}
        onClick={vi.fn()}
      />,
    );

    dragPast(container.querySelector('[data-slot="card"]') as HTMLElement);

    const ghost = document.querySelector('[data-slot="drag-ghost-card"]');
    expect(ghost).not.toBeNull();
    expect(ghost!.querySelector('[data-priority="high"]')).not.toBeNull();
    expect(ghost!.querySelector('[data-slot="comment-count"]')).toHaveTextContent("2");
    expect(ghost!.querySelector('[data-slot="has-description"]')).not.toBeNull();
  });
});
