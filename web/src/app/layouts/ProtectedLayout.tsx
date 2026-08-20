import { useState } from "react";
import { Navigate, Outlet } from "react-router";
import { useAuth } from "@/app/providers/auth";
import { AppSidebar } from "@/widgets/app-sidebar/ui/AppSidebar";
import { TopBar } from "@/widgets/top-bar/ui/TopBar";
import { BottomTabs } from "@/widgets/bottom-tabs/ui/BottomTabs";
import { Sheet, SheetContent } from "@/shared/ui/sheet";
import { Toaster } from "@/shared/ui/sonner";
import { TooltipProvider } from "@/shared/ui/tooltip";

export default function ProtectedLayout() {
  const { user, isLoading } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="size-6 animate-spin rounded-full border-2 border-muted-foreground/25 border-t-muted-foreground" />
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" replace />;
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
          <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-6 md:px-6 md:py-8 pb-20 md:pb-8">
            <Outlet />
          </main>
          <BottomTabs className="md:hidden" />
        </div>
      </div>
      <Toaster />
    </TooltipProvider>
  );
}
