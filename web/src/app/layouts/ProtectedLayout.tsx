import { useState } from "react";
import { Navigate, Outlet, useMatches } from "react-router";
import { useAuth } from "@/app/providers/auth";
import { AppSidebar } from "@/widgets/app-sidebar/ui/AppSidebar";
import { TopBar } from "@/widgets/top-bar/ui/TopBar";
import { BottomTabs } from "@/widgets/bottom-tabs/ui/BottomTabs";
import { Sheet, SheetContent } from "@/shared/ui/sheet";
import { TooltipProvider } from "@/shared/ui/tooltip";
import { cn } from "@/shared/lib/utils";

/** A route module opts out of the default centered `max-w-5xl` column by
 * exporting `handle = { fullWidth: true }` (board.ts routes) — the board's
 * three columns need the full frame width (~1100px+). */
interface RouteHandle {
  fullWidth?: boolean;
}

export default function ProtectedLayout() {
  const { user, isLoading } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const matches = useMatches();
  const fullWidth = matches.some((m) => (m.handle as RouteHandle | undefined)?.fullWidth);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="size-6 animate-spin rounded-full border-2 border-muted-foreground/25 border-t-muted-foreground" />
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/" replace />;
  }

  return (
    <TooltipProvider>
      <div className="flex min-h-screen">
        {/* Desktop sidebar */}
        <AppSidebar className="hidden md:flex" />

        {/* Mobile sidebar sheet */}
        <Sheet open={mobileMenuOpen} onOpenChange={setMobileMenuOpen}>
          <SheetContent side="left" className="w-[220px] p-0">
            <AppSidebar className="flex w-full border-r-0" />
          </SheetContent>
        </Sheet>

        <div className="flex flex-1 flex-col min-w-0">
          <TopBar onMenuClick={() => setMobileMenuOpen(true)} />
          <main
            className={cn(
              "w-full flex-1 px-4 py-6 md:px-6 md:py-8 pb-20 md:pb-8",
              !fullWidth && "mx-auto max-w-5xl",
            )}
          >
            <Outlet />
          </main>
          <BottomTabs className="md:hidden" />
        </div>
      </div>
    </TooltipProvider>
  );
}
