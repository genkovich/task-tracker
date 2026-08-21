import { useState } from "react";
import { useParams } from "react-router";
import type { Route } from "./+types/PublicBoardPage";
import { PublicBoardView } from "@/features/public-board-view/ui/PublicBoardView";
import NotFoundPage from "@/pages/not-found/ui/NotFoundPage";

export const meta: Route.MetaFunction = () => [{ title: "Board — Task Tracker" }];

export default function PublicBoardPage() {
  const { token } = useParams();
  const [notFound, setNotFound] = useState(false);

  // No token in the URL is itself a not-found (a malformed /b/ address) —
  // AC-05 treats it exactly like a disabled/never-valid one.
  if (!token || notFound) {
    return <NotFoundPage />;
  }

  return (
    <main className="min-h-screen bg-background p-4 sm:p-6">
      <div className="mx-auto max-w-6xl space-y-4">
        <div className="flex items-center gap-2">
          <h1 className="text-lg font-semibold">Task Tracker</h1>
          <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
            View only
          </span>
        </div>
        <PublicBoardView token={token} onNotFound={() => setNotFound(true)} />
      </div>
    </main>
  );
}
