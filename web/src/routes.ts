import { type RouteConfig, route, index, layout } from "@react-router/dev/routes";

export default [
  // Guest landing: product name + the single Google button. Signed-in users
  // are bounced straight to the board.
  index("pages/login/ui/LoginPage.tsx"),
  route("b/:token", "pages/board-public/ui/BoardPublicPage.tsx"),
  route("auth/callback", "pages/auth-callback/ui/AuthCallbackPage.tsx"),
  layout("app/layouts/AuthGate.tsx", [
    route("board", "pages/board/ui/BoardIndexPage.tsx"),
    route("board/:boardId", "pages/board/ui/BoardPage.tsx"),
  ]),
  layout("app/layouts/ProtectedLayout.tsx", [
    route("dashboard", "pages/dashboard/ui/DashboardPage.tsx"),
    route("profile", "pages/profile/ui/ProfilePage.tsx"),
  ]),
  route("*", "pages/not-found/ui/NotFoundPage.tsx"),
] satisfies RouteConfig;
