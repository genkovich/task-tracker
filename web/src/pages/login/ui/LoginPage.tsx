import { GoogleLoginButton } from "@/features/auth-by-google/ui/GoogleLoginButton";
import { Wordmark } from "@/shared/ui/Wordmark";

const MONO = "font-[family-name:'JetBrains_Mono',monospace]";

export default function LoginPage() {
  return (
    <div
      className="dark relative flex min-h-screen items-center justify-center overflow-hidden bg-[#09090B] px-4 text-[#FAFAFA]"
      style={{ fontFamily: "'Space Grotesk', sans-serif" }}
    >
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 z-0 opacity-60"
        style={{
          background: [
            "radial-gradient(circle at 20% 20%, rgba(245,158,11,0.10), transparent 45%)",
            "radial-gradient(circle at 80% 30%, rgba(34,211,238,0.10), transparent 45%)",
            "radial-gradient(circle at 50% 80%, rgba(236,72,153,0.07), transparent 45%)",
          ].join(", "),
        }}
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 z-10"
        style={{
          background: `url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)' opacity='0.03'/%3E%3C/svg%3E")`,
        }}
      />

      <div className="relative z-20 w-full max-w-sm">
        <Wordmark
          to="/"
          ariaLabel="my.app — back to landing"
          className="mb-10 justify-center text-lg"
        />

        <div className="rounded-xl border border-[#27272A] bg-[#0F0F12]/80 p-8 shadow-[0_0_60px_-30px_rgba(34,211,238,0.4)] backdrop-blur-xl">
          <div className="mb-6 text-center">
            <h1 className="text-2xl font-semibold tracking-tight">Sign in</h1>
            <p className={`mt-1 text-sm text-zinc-400 ${MONO}`}>// continue with your workspace</p>
          </div>
          <GoogleLoginButton />
          <p className="mt-6 text-center text-xs text-zinc-400">
            New here? Create an account by signing in.
          </p>
        </div>

        <p
          className={`mt-6 text-center text-[11px] uppercase tracking-widest text-zinc-400 ${MONO}`}
        >
          things that actually ship
        </p>
      </div>
    </div>
  );
}
