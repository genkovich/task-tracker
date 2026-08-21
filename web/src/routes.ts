import { type RouteConfig, route, index, layout } from "@react-router/dev/routes";

export default [
  // Guest landing: product name + the single Google button. Signed-in users
  // are bounced straight to the board.
  index("pages/login/ui/LoginPage.tsx"),
  route("b/:token", "pages/board-public/ui/BoardPublicPage.tsx"),
  route("auth/callback", "pages/auth-callback/ui/AuthCallbackPage.tsx"),
  layout("app/layouts/ProtectedLayout.tsx", [
    route("dashboard", "pages/dashboard/ui/DashboardPage.tsx"),
    route("profile", "pages/profile/ui/ProfilePage.tsx"),
    // The board routes export `handle = { fullWidth: true }` — ProtectedLayout
    // reads it via useMatches() to drop its default max-w-5xl centering, since
    // the three-column board needs the full frame width (~1100px+).
    route("board", "pages/board/ui/BoardIndexPage.tsx"),
    route("board/:boardId", "pages/board/ui/BoardPage.tsx"),
  ]),
  route("*", "pages/not-found/ui/NotFoundPage.tsx"),
] satisfies RouteConfig;
