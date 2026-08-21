import { useState } from "react";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
  PopoverHeader,
  PopoverTitle,
} from "@/shared/ui/popover";
import { Button } from "@/shared/ui/button";
import { Badge } from "@/shared/ui/badge";
import { ApiClientError } from "@/shared/api/client";
import { publicLinkApi } from "../api/publicLinkApi";
import { noLink, activeLink, type PublicLinkState } from "../model/publicLinkState";

export function PublicLinkPanel() {
  const [state, setState] = useState<PublicLinkState>(noLink);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function handleIssue() {
    setError(null);
    setPending(true);
    try {
      const link = await publicLinkApi.issue();
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
      await publicLinkApi.revoke();
      setState(noLink);
    } catch (err) {
      setError(err instanceof ApiClientError ? err.message : "failed to revoke public link");
    } finally {
      setPending(false);
    }
  }

  // Must match the viewer route in routes.ts (`b/:token`), not the API path.
  const publicUrl =
    state.status === "active" ? `${window.location.origin}/b/${state.link.token}` : null;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm">
          Поділитись
        </Button>
      </PopoverTrigger>
      <PopoverContent>
        <PopoverHeader>
          <PopoverTitle>Публічне посилання</PopoverTitle>
        </PopoverHeader>

        {state.status === "none" ? (
          <div className="mt-2 flex flex-col gap-2">
            <Badge variant="secondary">Немає активного лінка</Badge>
            <Button size="sm" disabled={pending} onClick={handleIssue}>
              Отримати лінк
            </Button>
          </div>
        ) : (
          <div className="mt-2 flex flex-col gap-2">
            <Badge>Активний лінк</Badge>
            <p className="text-sm break-all">{publicUrl}</p>
            <Button size="sm" variant="destructive" disabled={pending} onClick={handleRevoke}>
              Відкликати
            </Button>
          </div>
        )}

        {error && <p className="text-destructive mt-2 text-sm">{error}</p>}
      </PopoverContent>
    </Popover>
  );
}
