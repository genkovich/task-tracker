import type { ReactNode } from "react";

export interface BoardShellProps {
  /** Права частина шапки: Share + аватар (SCR-01) або бейдж «Лише перегляд» (SCR-05). */
  actions?: ReactNode;
  children: ReactNode;
}

/** Спільний chrome борд-екранів (Design/scr01*, scr05*): темна сторінка,
 * шапка з брендом і діями праворуч, заголовок «Дошка». Прототип малює борд
 * темним незалежно від теми застосунку, тому клас `dark` тут примусовий —
 * той самий прийом, що на LoginPage. */
export function BoardShell({ actions, children }: BoardShellProps) {
  return (
    <div className="dark flex min-h-screen flex-col bg-background font-sans text-foreground [color-scheme:dark]">
      <header className="flex h-16 shrink-0 items-center justify-between border-b border-white/10 px-4 sm:px-6">
        <span className="text-[17px] font-bold tracking-tight">Task Tracker</span>
        <div className="flex items-center gap-2 sm:gap-3">{actions}</div>
      </header>
      <main className="flex w-full flex-1 flex-col gap-6 px-4 py-6 sm:px-8 sm:py-8">
        <h1 className="text-3xl font-bold tracking-tight">Дошка</h1>
        {children}
      </main>
    </div>
  );
}
