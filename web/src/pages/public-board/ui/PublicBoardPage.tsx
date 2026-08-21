import { useEffect } from "react";
import { useNavigate, useParams } from "react-router";
import type { Route } from "./+types/PublicBoardPage";
import { PublicBoardView } from "@/features/public-board-view/ui/PublicBoardView";

export const meta: Route.MetaFunction = () => [
  { title: "Board — Task Tracker" },
  // Deliberately not indexed / archived — spec §6.1: the public view shows
  // the team's current work, not something meant for search or long-term
  // public discovery.
  { name: "robots", content: "noindex, nofollow" },
];

export default function PublicBoardPage() {
  const { token } = useParams();
  const navigate = useNavigate();

  // No token in the URL is itself a not-found (a malformed /b/ address) —
  // AC-05 treats it exactly like a disabled/never-valid one. Navigate to
  // the app's own catch-all route rather than importing NotFoundPage
  // directly — same-layer page-to-page imports are forbidden by this
  // repo's FSD convention (web/CLAUDE.md), and this is the mechanism
  // screens.md's SCR-03 actually specifies.
  useEffect(() => {
    if (!token) navigate("/not-found", { replace: true });
  }, [token, navigate]);

  if (!token) return null;

  return (
    <main className="min-h-screen bg-background p-4 sm:p-6">
      <div className="mx-auto max-w-6xl space-y-4">
        <div className="flex items-center gap-2">
          <h1 className="text-lg font-semibold">Task Tracker</h1>
          <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
            View only
          </span>
        </div>
        <PublicBoardView
          token={token}
          onNotFound={() => navigate("/not-found", { replace: true })}
        />
      </div>
    </main>
  );
}
