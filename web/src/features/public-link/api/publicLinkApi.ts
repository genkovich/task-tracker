import { apiClient } from "@/shared/api/client";

export interface PublicLink {
  token: string;
  created_at: string;
}

// Per-board public link (boards BRD-06): each board issues and revokes its
// own link independently.
export const publicLinkApi = {
  issue: (boardId: string) =>
    apiClient.post<PublicLink>(`/boards/${encodeURIComponent(boardId)}/public-link`, {}),

  revoke: (boardId: string) =>
    apiClient.delete<void>(`/boards/${encodeURIComponent(boardId)}/public-link`),
};
