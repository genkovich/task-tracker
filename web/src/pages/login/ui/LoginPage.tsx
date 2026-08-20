import type { Route } from "./+types/LoginPage";
import { GoogleLoginButton } from "@/features/auth-by-google/ui/GoogleLoginButton";

export const meta: Route.MetaFunction = () => [{ title: "Sign in — Task Tracker" }];

export default function LoginPage() {
  return (
    <div className="dark flex min-h-screen items-center justify-center bg-background px-6 font-sans text-foreground">
      <div className="flex w-full max-w-sm flex-col items-center gap-8 text-center">
        <div className="flex flex-col items-center gap-3">
          <h1 className="text-3xl font-bold tracking-tight">Task Tracker</h1>
          <p className="text-base text-muted-foreground">
            Every board, every task, one shared workspace.
          </p>
        </div>
        <GoogleLoginButton />
      </div>
    </div>
  );
}
