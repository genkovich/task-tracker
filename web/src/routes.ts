import { type RouteConfig, route, index, layout } from "@react-router/dev/routes";

export default [
  index("pages/board/ui/BoardPage.tsx"),
  route("login", "pages/login/ui/LoginPage.tsx"),
  route("auth/callback", "pages/auth-callback/ui/AuthCallbackPage.tsx"),
  layout("app/layouts/ProtectedLayout.tsx", [
    route("dashboard", "pages/dashboard/ui/DashboardPage.tsx"),
    route("profile", "pages/profile/ui/ProfilePage.tsx"),
  ]),
  route("*", "pages/not-found/ui/NotFoundPage.tsx"),
] satisfies RouteConfig;
