import type { Route } from "./+types/BoardPage";
import { Toaster } from "@/shared/ui/sonner";
import { BoardView } from "@/features/board-view/ui/BoardView";

export const meta: Route.MetaFunction = () => [{ title: "Board — Task Tracker" }];

export default function BoardPage() {
  return (
    <main className="min-h-screen bg-background p-4 sm:p-6">
      <div className="mx-auto max-w-6xl space-y-4">
        <h1 className="text-lg font-semibold">Task Tracker</h1>
        <BoardView />
      </div>
      <Toaster />
    </main>
  );
}
