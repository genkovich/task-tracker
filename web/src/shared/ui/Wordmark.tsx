import { Link } from "react-router";

const MONO = "font-[family-name:'JetBrains_Mono',monospace]";

interface WordmarkProps {
  readonly to?: string;
  readonly className?: string;
  readonly ariaLabel?: string;
}

export function Wordmark({ to, className = "", ariaLabel = "my.app" }: WordmarkProps) {
  const content = (
    <>
      <span className="text-amber-500">my</span>
      <span className="text-pink-500">.</span>
      <span className="text-cyan-400">app</span>
    </>
  );

  const baseClass = `inline-flex font-bold tracking-tight ${MONO} ${className}`;

  if (to) {
    return (
      <Link to={to} aria-label={ariaLabel} className={baseClass}>
        {content}
      </Link>
    );
  }

  return (
    <span aria-label={ariaLabel} className={baseClass}>
      {content}
    </span>
  );
}
