import { LogOut, UserRound } from "lucide-react";

import { useAuth } from "@/app/providers/auth";
import { getDisplayName, getInitials } from "@/shared/lib/user";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";

/** Аватар користувача в шапці борда (SCR-01): ініціали на темному колі, як у
 * прототипі. Клік відкриває маленьке меню з іменем і «Вийти» — той самий
 * вихід, що в TopBar світлих сторінок (useAuth().logout чистить токени і
 * веде на лендінг). Без користувача — нейтральна іконка без меню. */
export function BoardUserBadge() {
  const { user, logout } = useAuth();

  if (!user) {
    return (
      <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-white/10">
        <UserRound aria-hidden className="size-4 text-muted-foreground" />
      </span>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          aria-label="Відкрити меню користувача"
          title={getDisplayName(user)}
          className="flex size-9 shrink-0 items-center justify-center rounded-full bg-white/10 text-xs font-semibold transition-shadow hover:ring-2 hover:ring-white/20"
        >
          {getInitials(user)}
        </button>
      </DropdownMenuTrigger>
      {/* Контент рендериться порталом поза BoardShell, тож темна тема борда
       * на нього не поширюється — клас dark тут примусовий, як на самому
       * BoardShell. */}
      <DropdownMenuContent align="end" className="dark w-48 [color-scheme:dark]">
        <DropdownMenuLabel>{getDisplayName(user)}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={logout}>
          <LogOut className="mr-2 size-4" />
          Вийти
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
