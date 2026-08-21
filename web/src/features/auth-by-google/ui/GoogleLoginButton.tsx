import { useAuth } from "@/app/providers/auth";
import { Button } from "@/shared/ui/button";
import { GoogleIcon } from "@/shared/ui/google-icon";

export function GoogleLoginButton() {
  const { login } = useAuth();

  return (
    <Button size="lg" className="w-full rounded-full gap-2 sm:w-auto" onClick={login}>
      <GoogleIcon />
      Sign in with Google
    </Button>
  );
}
