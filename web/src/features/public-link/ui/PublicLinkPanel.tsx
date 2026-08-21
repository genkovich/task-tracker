import { useEffect, useState } from "react";
import { Check, Copy, Share2 } from "lucide-react";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
  PopoverHeader,
  PopoverTitle,
} from "@/shared/ui/popover";
import { Button } from "@/shared/ui/button";
import { ApiClientError } from "@/shared/api/client";
import { publicLinkApi, type PublicLink } from "../api/publicLinkApi";
import { noLink, activeLink, type PublicLinkState } from "../model/publicLinkState";

export interface PublicLinkPanelProps {
  /** The board this panel issues/revokes the link for (boards BRD-06). */
  boardId: string;
  /** The board's current link from `BoardState.public_link` (AC-08) — seeds
   * the panel so an already-issued link survives a page reload. */
  publicLink?: PublicLink | null;
}

/** SCR-04 public-link panel: попап від кнопки «Поділитись» у шапці борда
 * (Design/scr04-share-link-*). Стани: немає лінка → «Отримати лінк»;
 * активний → URL із копіюванням + «Відкликати лінк». */
export function PublicLinkPanel({ boardId, publicLink = null }: PublicLinkPanelProps) {
  const [state, setState] = useState<PublicLinkState>(publicLink ? activeLink(publicLink) : noLink);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [copied, setCopied] = useState(false);

  // A board refetch can deliver a fresh `public_link` (issued or revoked in
  // another tab) — adopt the prop change. Local issue/revoke transitions
  // are untouched: they don't change the prop, so this never re-runs.
  useEffect(() => {
    setState(publicLink ? activeLink(publicLink) : noLink);
  }, [publicLink]);

  async function handleIssue() {
    setError(null);
    setPending(true);
    try {
      const link = await publicLinkApi.issue(boardId);
      setState(activeLink(link));
    } catch (err) {
      setError(err instanceof ApiClientError ? err.message : "failed to issue public link");
    } finally {
      setPending(false);
    }
  }

  async function handleRevoke() {
    setError(null);
    setPending(true);
    try {
      await publicLinkApi.revoke(boardId);
      setState(noLink);
    } catch (err) {
      setError(err instanceof ApiClientError ? err.message : "failed to revoke public link");
    } finally {
      setPending(false);
    }
  }

  async function handleCopy(url: string) {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Кліпборд недоступний (нема дозволу/не secure context) — URL і так
      // видно текстом, тож мовчки лишаємо ручне копіювання.
    }
  }

  // Must match the viewer route in routes.ts (`b/:token`), not the API path.
  const publicUrl =
    state.status === "active" ? `${window.location.origin}/b/${state.link.token}` : null;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="secondary"
          size="sm"
          aria-label="Поділитись"
          className="h-9 rounded-full bg-white/10 hover:bg-white/15 max-sm:size-9 max-sm:bg-white max-sm:p-0 max-sm:text-neutral-900 max-sm:hover:bg-white/90 sm:px-4"
        >
          <Share2 aria-hidden />
          <span className="max-sm:hidden">Поділитись</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="end"
        sideOffset={8}
        className="dark w-[min(22rem,calc(100vw-2rem))] rounded-2xl border-white/10 p-5 font-sans [color-scheme:dark]"
      >
        <PopoverHeader>
          <PopoverTitle className="text-lg font-semibold">Публічне посилання</PopoverTitle>
        </PopoverHeader>

        {state.status === "none" ? (
          <div className="mt-3 flex flex-col gap-3">
            <p className="text-[15px] font-semibold">Публічного лінка ще немає.</p>
            <p className="text-sm text-muted-foreground">
              Будь-хто з лінком зможе переглядати цю дошку, але не редагувати.
            </p>
            <Button
              size="sm"
              className="self-start rounded-full px-4"
              disabled={pending}
              onClick={handleIssue}
            >
              Отримати лінк
            </Button>
          </div>
        ) : (
          <div className="mt-3 flex flex-col gap-3">
            <div className="flex items-center gap-1 rounded-xl bg-white/[0.06] py-1 pr-1 pl-3">
              <p className="min-w-0 flex-1 truncate text-sm">{publicUrl}</p>
              <Button
                variant="ghost"
                size="icon-sm"
                className="rounded-lg text-muted-foreground hover:text-foreground"
                aria-label="Скопіювати лінк"
                onClick={() => publicUrl && void handleCopy(publicUrl)}
              >
                {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
              </Button>
            </div>
            <Button
              size="sm"
              variant="destructive"
              className="self-start rounded-full px-4 dark:bg-destructive dark:hover:bg-destructive/90"
              disabled={pending}
              onClick={handleRevoke}
            >
              Відкликати лінк
            </Button>
            <p className="text-sm text-muted-foreground">
              Завжди показує актуальну дошку, лише для перегляду.
            </p>
          </div>
        )}

        {error && <p className="text-destructive mt-3 text-sm">{error}</p>}
      </PopoverContent>
    </Popover>
  );
}
