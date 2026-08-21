import { apiClient } from "@/shared/api/client";

export interface PublicLink {
  id: string;
  token?: string;
  disabled_at: string | null;
  created_at: string;
}

interface ActiveLinkResponse {
  link: PublicLink | null;
}

export const publicLinkApi = {
  getActiveLink: async (): Promise<PublicLink | null> => {
    const resp = await apiClient.get<ActiveLinkResponse>("/public-links/active");
    return resp.link;
  },
  generateLink: () => apiClient.post<PublicLink>("/public-links", {}),
  disableLink: (id: string) => apiClient.post<PublicLink>(`/public-links/${id}/disable`, {}),
};
