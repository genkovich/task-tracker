import type { Route } from "./+types/DashboardPage";
import { useAuth } from "@/app/providers/auth";
import { getDisplayName } from "@/shared/lib/user";

export const meta: Route.MetaFunction = () => [{ title: "Dashboard — Task Tracker" }];

// Навмисно без дизайну: голе привітання. Оформлення — справа проєкту.
export default function DashboardPage() {
  const { user } = useAuth();

  if (!user) return null;

  return <h1>Hello, {getDisplayName(user)}!</h1>;
}
