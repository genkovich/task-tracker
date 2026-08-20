const STORAGE_KEY = "task-tracker-theme";

export function applyStoredTheme(): void {
  if (typeof document === "undefined") return;
  let stored: string | null = null;
  try {
    stored = window.localStorage.getItem(STORAGE_KEY);
  } catch {
    stored = null;
  }
  const theme = stored === "light" || stored === "dark" ? stored : "system";
  const dark =
    theme === "dark" ||
    (theme === "system" &&
      typeof window !== "undefined" &&
      typeof window.matchMedia === "function" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);
  document.documentElement.classList.toggle("dark", dark);
  document.documentElement.style.colorScheme = dark ? "dark" : "light";
}

/**
 * Inline-able IIFE that runs before React mounts so the initial HTML
 * already carries the right theme class. Prevents the React hydration
 * mismatch (#418) that fires when the post-mount effect toggles the
 * class after hydration finishes.
 *
 * Kept as a string so we can serialise it into <script> inside <head>
 * via React Router's Layout.
 */
export const themeInitScript = `(function(){try{var k="${STORAGE_KEY}";var s=null;try{s=window.localStorage.getItem(k);}catch(e){}var t=s==="light"||s==="dark"?s:"system";var d=t==="dark"||(t==="system"&&window.matchMedia&&window.matchMedia("(prefers-color-scheme: dark)").matches);document.documentElement.classList.toggle("dark",d);document.documentElement.style.colorScheme=d?"dark":"light";}catch(e){}})();`;
