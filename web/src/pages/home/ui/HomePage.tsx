import { Navigate } from "react-router";
import type { Route } from "./+types/HomePage";
import { useAuth } from "@/app/providers/auth";
import { GoogleLoginButton } from "@/features/auth-by-google/ui/GoogleLoginButton";

export const meta: Route.MetaFunction = () => [{ title: "Task Tracker" }];

// Навмисно без дизайну: гола кнопка логіну. Оформлення — справа проєкту.
export default function HomePage() {
  const { user, isLoading } = useAuth();

  if (isLoading) return null;

  if (user) {
    return <Navigate to="/dashboard" replace />;
  }

  return (
    <main className="flex min-h-screen items-center justify-center">
      <div className="w-full max-w-xs text-center">
        <h1 className="mb-6 text-2xl">Task Tracker</h1>
        <GoogleLoginButton />
      </div>
    </main>
  );
}
