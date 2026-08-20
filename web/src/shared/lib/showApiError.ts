import { ApiClientError } from "@/shared/api/client";
import { toast } from "sonner";

export function showApiError(err: unknown) {
  if (err instanceof ApiClientError) {
    toast.error(err.message);
  } else {
    toast.error("Something went wrong");
  }
}
