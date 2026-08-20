import { apiClient } from "@/shared/api/client";
import type { TokenResponse } from "../model/types";

export const authApi = {
  refreshTokens: (refreshToken: string) =>
    apiClient.post<TokenResponse>("/auth/refresh", {
      refresh_token: refreshToken,
    }),
};
