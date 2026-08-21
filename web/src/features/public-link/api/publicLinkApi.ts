import { apiClient } from "@/shared/api/client";

export interface PublicLink {
  token: string;
  created_at: string;
}

export const publicLinkApi = {
  issue: () => apiClient.post<PublicLink>("/board/public-link", {}),

  revoke: () => apiClient.delete<void>("/board/public-link"),
};
