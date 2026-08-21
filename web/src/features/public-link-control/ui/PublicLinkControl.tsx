import { useEffect, useState } from "react";
import { toast } from "sonner";
import { CopyIcon } from "lucide-react";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { publicLinkApi, type PublicLink } from "../api/publicLinkApi";

function shareUrl(token: string): string {
  return `${window.location.origin}/b/${token}`;
}

export function PublicLinkControl() {
  const [link, setLink] = useState<PublicLink | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<"generating" | "disabling" | null>(null);

  useEffect(() => {
    publicLinkApi
      .getActiveLink()
      .then(setLink)
      .catch(() => setLink(null))
      .finally(() => setLoading(false));
  }, []);

  async function handleGenerate() {
    setBusy("generating");
    try {
      const created = await publicLinkApi.generateLink();
      setLink(created);
    } catch {
      toast.error("Could not create a public link — try again");
    } finally {
      setBusy(null);
    }
  }

  async function handleDisable() {
    if (!link) return;
    setBusy("disabling");
    try {
      await publicLinkApi.disableLink(link.id);
      setLink(null);
    } catch {
      toast.error("Could not disable the public link — try again");
    } finally {
      setBusy(null);
    }
  }

  async function handleCopy() {
    if (!link?.token) return;
    await navigator.clipboard.writeText(shareUrl(link.token));
    toast.success("Link copied");
  }

  if (loading) return null;

  if (!link) {
    return (
      <Button variant="outline" onClick={handleGenerate} disabled={busy === "generating"}>
        {busy === "generating" ? "Generating..." : "Get link"}
      </Button>
    );
  }

  return (
    <div className="flex items-center gap-2">
      {link.token ? (
        <>
          <Input readOnly value={shareUrl(link.token)} className="w-64 text-xs" />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Copy link"
            onClick={handleCopy}
          >
            <CopyIcon className="size-4" />
          </Button>
        </>
      ) : (
        // The token is shown only once, right after generation (minimal
        // exposure by design) — a page reload after that shows this instead.
        <span className="text-sm text-muted-foreground">Public link is active</span>
      )}
      <Button variant="outline" onClick={handleDisable} disabled={busy === "disabling"}>
        {busy === "disabling" ? "Disabling..." : "Disable link"}
      </Button>
    </div>
  );
}
