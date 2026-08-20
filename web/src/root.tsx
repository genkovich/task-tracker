import {
  isRouteErrorResponse,
  Link,
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
} from "react-router";

import type { Route } from "./+types/root";
import { Providers } from "@/app/providers";
import { Button } from "@/shared/ui/button";
import { Wordmark } from "@/shared/ui/Wordmark";
import "@fontsource-variable/jetbrains-mono";
import "@fontsource-variable/space-grotesk";
import "@/app/styles/global.css";

const MONO_FONT = "font-[family-name:'JetBrains_Mono',monospace]";

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <script src="/theme-init.js" />
        <Meta />
        <Links />
      </head>
      <body>
        {children}
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function App() {
  return (
    <Providers>
      <Outlet />
    </Providers>
  );
}

export function HydrateFallback() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-background">
      <div
        className="size-8 animate-spin rounded-full border-2 border-muted-foreground/25 border-t-muted-foreground"
        role="status"
        aria-label="Loading"
      />
    </main>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  let title = "Something went wrong";
  let details = "An unexpected error occurred. Try refreshing the page.";
  let stack: string | undefined;
  let isNotFound = false;

  if (isRouteErrorResponse(error)) {
    isNotFound = error.status === 404;
    title = isNotFound ? "Page not found" : `Error ${error.status}`;
    details = isNotFound
      ? "The page you are looking for doesn't exist or has moved."
      : error.statusText || details;
  } else if (import.meta.env.DEV && error && error instanceof Error) {
    details = error.message;
    stack = error.stack;
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 py-12">
      <div className="w-full max-w-md text-center">
        <Wordmark to="/" className="mb-10 text-base" />
        {isNotFound && (
          <p
            className={`text-7xl font-bold tracking-tight text-cyan-400 ${MONO_FONT}`}
            aria-hidden="true"
          >
            404
          </p>
        )}
        <h1 className="mt-6 text-2xl font-semibold tracking-tight">{title}</h1>
        <p className="mt-3 text-sm text-muted-foreground">{details}</p>
        <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <Button asChild>
            <Link to="/dashboard">Go to dashboard</Link>
          </Button>
          <Button asChild variant="outline">
            <Link to="/">Back home</Link>
          </Button>
        </div>
        {stack && (
          <pre className="mt-8 w-full overflow-x-auto rounded-lg bg-muted p-4 text-left text-xs">
            <code>{stack}</code>
          </pre>
        )}
      </div>
    </main>
  );
}
