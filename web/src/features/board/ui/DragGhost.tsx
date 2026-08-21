import type { ReactNode, Ref } from "react";
import { createPortal } from "react-dom";

export interface DragGhostProps {
  width: number;
  height: number;
  children: ReactNode;
  ref?: Ref<HTMLDivElement>;
}

/** Floating clone of a dragged card, portalled to `document.body` so an
 * ancestor's `overflow-x-auto` (the columns row) never clips it mid-drag —
 * previously the dragged card moved via its own `transform`, which the
 * scrollable row's clipping context cut off as soon as it crossed the row's
 * edge. Positioned via a `transform: translate(x, y)` written directly to
 * this node by the caller on every pointer move (no re-render per move,
 * matching TaskCard's existing imperative-style drag). `pointer-events:none`
 * keeps it out of `elementFromPoint` hit-testing, same as the dragged
 * original always was. */
export function DragGhost({ width, height, children, ref }: DragGhostProps) {
  return createPortal(
    <div
      ref={ref}
      aria-hidden
      className="pointer-events-none fixed top-0 left-0 z-50 rounded-2xl shadow-lg"
      style={{ width, height }}
    >
      {children}
    </div>,
    document.body,
  );
}
