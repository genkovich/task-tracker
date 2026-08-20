import { apiClient, ApiClientError, BASE_URL } from "@/shared/api/client";
import type { CurrentUser } from "@/entities/user/model/types";

export interface UpdateProfileData {
  first_name: string | null;
  last_name: string | null;
  position: string | null;
  department: string | null;
  bio: string | null;
  timezone: string | null;
}

async function uploadAvatar(file: File): Promise<CurrentUser> {
  const form = new FormData();
  form.append("avatar", file);

  const token = localStorage.getItem("access_token");
  const res = await fetch(`${BASE_URL}/api/v1/users/me/avatar`, {
    method: "POST",
    body: form,
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => null);
    const code = body?.error?.code ?? "unknown";
    const message = body?.error?.message ?? `Upload failed (${res.status})`;
    throw new ApiClientError(code, message, res.status);
  }
  return res.json();
}

export const profileApi = {
  updateProfile: (data: UpdateProfileData) => apiClient.put<CurrentUser>("/users/me", data),
  uploadAvatar,
  deleteAvatar: () => apiClient.delete<CurrentUser>("/users/me/avatar"),
};
