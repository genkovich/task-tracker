import { UserRound } from "lucide-react";

import { useAuth } from "@/app/providers/auth";
import { getDisplayName, getInitials } from "@/shared/lib/user";

/** Аватар-коло користувача в шапці борда (SCR-01): ініціали на темному колі,
 * як у прототипі; без користувача — нейтральна іконка. */
export function BoardUserBadge() {
  const { user } = useAuth();

  return (
    <span
      title={user ? getDisplayName(user) : undefined}
      className="flex size-9 shrink-0 items-center justify-center rounded-full bg-white/10 text-xs font-semibold"
    >
      {user ? (
        getInitials(user)
      ) : (
        <UserRound aria-hidden className="size-4 text-muted-foreground" />
      )}
    </span>
  );
}
