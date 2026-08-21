import { Link } from "react-router";
import { Button } from "@/shared/ui/button";
import { Wordmark } from "@/shared/ui/Wordmark";

const MONO = "font-[family-name:'JetBrains_Mono',monospace]";

export default function NotFoundPage() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 py-12">
      <div className="w-full max-w-md text-center">
        <Wordmark to="/" className="mb-10 text-base" />
        <p
          className={`text-7xl font-bold tracking-tight text-cyan-400 ${MONO}`}
          aria-hidden="true"
        >
          404
        </p>
        <h1 className="mt-6 text-2xl font-semibold tracking-tight">Page not found</h1>
        <p className="mt-3 text-sm text-muted-foreground">
          The page you are looking for doesn&apos;t exist or has moved.
        </p>
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <Button asChild>
            <Link to="/board">Go to the board</Link>
          </Button>
          <Button asChild variant="outline">
            <Link to="/">Back home</Link>
          </Button>
        </div>
      </div>
    </main>
  );
}
