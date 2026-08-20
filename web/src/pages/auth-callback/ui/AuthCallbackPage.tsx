import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { useAuth } from "@/app/providers/auth";
import { BASE_URL } from "@/shared/api/client";

export default function AuthCallbackPage() {
  const { setTokens, fetchUser } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  useEffect(() => {
    const code = searchParams.get("code");

    if (!code) {
      navigate("/login", { replace: true });
      return;
    }

    const exchangeCode = async () => {
      try {
        const res = await fetch(`${BASE_URL}/api/v1/auth/exchange`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ code }),
        });

        if (!res.ok) {
          navigate("/login", { replace: true });
          return;
        }

        const data = (await res.json()) as {
          access_token: string;
          refresh_token: string;
          expires_in: number;
        };

        setTokens(data.access_token, data.refresh_token);

        await fetchUser();
        navigate("/dashboard", { replace: true });
      } catch {
        localStorage.removeItem("access_token");
        localStorage.removeItem("refresh_token");
        navigate("/login", { replace: true });
      }
    };

    exchangeCode();
  }, [searchParams, setTokens, fetchUser, navigate]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/50 animate-in-fade">
      <p className="text-sm text-muted-foreground">Signing in...</p>
    </div>
  );
}
