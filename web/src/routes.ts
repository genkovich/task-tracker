import { type RouteConfig, route, index, layout } from "@react-router/dev/routes";

export default [
  index("pages/home/ui/HomePage.tsx"),
  route("login", "pages/login/ui/LoginPage.tsx"),
  route("auth/callback", "pages/auth-callback/ui/AuthCallbackPage.tsx"),
  // No login required — team members edit the board with no accounts
  // (spec §3 Non-goals), so these routes stay outside ProtectedLayout.
  route("board", "pages/board/ui/BoardPage.tsx"),
  route("b/:token", "pages/public-board/ui/PublicBoardPage.tsx"),
  layout("app/layouts/ProtectedLayout.tsx", [
    route("dashboard", "pages/dashboard/ui/DashboardPage.tsx"),
    route("profile", "pages/profile/ui/ProfilePage.tsx"),
  ]),
  route("*", "pages/not-found/ui/NotFoundPage.tsx"),
] satisfies RouteConfig;
