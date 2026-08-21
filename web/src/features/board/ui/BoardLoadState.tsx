import { CircleAlert } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { EmptyState } from "@/shared/ui/EmptyState";

/** SCR-01 loading state (Design/scr01-board-loading-*.png): the board area
 * shows a progress indicator instead of a blank screen while fetching. */
export function BoardLoading() {
  return (
    <div className="flex flex-1 items-center justify-center">
      <div
        className="size-6 animate-spin rounded-full border-2 border-muted-foreground/25 border-t-muted-foreground"
        role="status"
        aria-label="Завантаження дошки"
      />
    </div>
  );
}

/** SCR-01 error state (Design/scr01-board-error-*.png): a centered message
 * with a retry action instead of a white screen. */
export function BoardLoadError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex flex-1 items-center justify-center">
      <EmptyState
        Icon={CircleAlert}
        title="Не вдалося завантажити дошку"
        description="Перевірте з'єднання і спробуйте ще раз."
        action={
          <Button className="rounded-full px-5" onClick={onRetry}>
            Спробувати ще
          </Button>
        }
      />
    </div>
  );
}
