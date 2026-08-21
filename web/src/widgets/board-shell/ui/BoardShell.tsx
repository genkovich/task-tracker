import type { ReactNode } from "react";

export interface BoardShellProps {
  /** Дії поруч із заголовком «Дошка»: Share (SCR-01) або бейдж «Лише
   * перегляд» (SCR-05) — контекстні дії цієї дошки, не глобальний chrome. */
  actions?: ReactNode;
  children: ReactNode;
}

/** Board content frame (Design/scr01*, scr05*): the «Дошка» heading plus its
 * contextual actions, then the columns. No page chrome of its own (no
 * header/background/height) — the surrounding layout (ProtectedLayout for
 * the team-editor view, the guest page's own header for the public viewer)
 * owns that; this just composes the heading row and the board body. */
export function BoardShell({ actions, children }: BoardShellProps) {
  return (
    <div className="flex w-full flex-1 flex-col gap-6">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-3xl font-bold tracking-tight">Дошка</h1>
        <div className="flex items-center gap-2 sm:gap-3">{actions}</div>
      </div>
      {children}
    </div>
  );
}
