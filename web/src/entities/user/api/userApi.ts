import { apiClient } from "@/shared/api/client";
import type { CurrentUser, User } from "../model/types";

export interface UsersListResponse {
  users: User[];
  has_next: boolean;
  has_prev: boolean;
}

export const userApi = {
  getById: (id: string) => apiClient.get<User>(`/users/${id}`),

  list: (params?: { after?: string; before?: string; limit?: number }) => {
    const search = new URLSearchParams();
    if (params?.after) search.set("after", params.after);
    if (params?.before) search.set("before", params.before);
    if (params?.limit) search.set("limit", String(params.limit));
    const qs = search.toString();
    return apiClient.get<UsersListResponse>(`/users${qs ? `?${qs}` : ""}`);
  },

  create: (data: {
    email: string;
    first_name?: string;
    last_name?: string;
    role?: string;
  }) => apiClient.post<User>("/users", data),

  update: (
    id: string,
    data: {
      email: string;
      first_name?: string;
      last_name?: string;
      avatar_url?: string;
      role: string;
    },
  ) => apiClient.put<User>(`/users/${id}`, data),

  delete: (id: string) => apiClient.delete(`/users/${id}`),

  getCurrentUser: () => apiClient.get<CurrentUser>("/auth/me"),
};
