import { LayoutDashboard, Plus } from "lucide-react";
import type { Route } from "./+types/DashboardPage";
import { Button } from "@/shared/ui/button";
import { EmptyState } from "@/shared/ui/EmptyState";

export const meta: Route.MetaFunction = () => [{ title: "Dashboard — Task Tracker" }];

export default function DashboardPage() {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold tracking-tight">Your boards</h1>
        <Button className="rounded-full gap-1.5">
          <Plus className="size-4" />
          <span className="hidden sm:inline">New board</span>
        </Button>
      </div>

      <div className="min-h-[520px] rounded-xl bg-muted">
        <EmptyState
          Icon={LayoutDashboard}
          title="No boards yet"
          description="Create your first board to start organizing tasks."
          action={<Button className="rounded-full">Create board</Button>}
          className="min-h-[520px] justify-center"
        />
      </div>
    </div>
  );
}
