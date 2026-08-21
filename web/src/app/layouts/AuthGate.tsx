import { Navigate, Outlet } from "react-router";
import { useAuth } from "@/app/providers/auth";

// Chrome-less auth guard: the board owns its full-width layout, so it only
// needs the redirect half of ProtectedLayout, not the sidebar/topbar shell.
export default function AuthGate() {
  const { user, isLoading } = useAuth();

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

  return <Outlet />;
}
