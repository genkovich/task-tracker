import { useEffect } from "react";
import { useNavigate } from "react-router";

import type { Route } from "./+types/BoardIndexPage";
import { boardApi } from "@/features/board/api/boardApi";

export const meta: Route.MetaFunction = () => [{ title: "Дошка — Task Tracker" }];

// ProtectedLayout reads this via useMatches() to drop its default centered
// max-w-5xl column, matching BoardPage — this route only ever shows a brief
// loading spinner before redirecting, but it still renders inside the same
// full-width frame.
export const handle = { fullWidth: true };

/** Легасі-адреса /board без ідентифікатора (boards BRD-07): відкриває першу
 * дошку, а коли дощок немає (чи список не вдалось отримати) — дашборд. */
export default function BoardIndexPage() {
  const navigate = useNavigate();

  useEffect(() => {
    let cancelled = false;
    boardApi
      .listBoards()
      .then((boards) => {
        if (cancelled) return;
        navigate(boards.length > 0 ? `/board/${boards[0].id}` : "/dashboard", { replace: true });
      })
      .catch(() => {
        if (!cancelled) navigate("/dashboard", { replace: true });
      });
    return () => {
      cancelled = true;
    };
  }, [navigate]);

  return (
    <div className="flex flex-1 items-center justify-center">
      <div
        className="size-6 animate-spin rounded-full border-2 border-muted-foreground/25 border-t-muted-foreground"
        role="status"
        aria-label="Завантаження дошки"
      />
    </div>
  );
}
