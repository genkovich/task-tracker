import type { ApiError } from "./types";

export function validateBaseUrl(url: string): string {
  if (!/^https?:\/\//.test(url)) {
    throw new Error(`Invalid BASE_URL: "${url}". Must start with http:// or https://`);
  }
  return url;
}

export const BASE_URL = validateBaseUrl(import.meta.env.VITE_API_URL ?? "http://localhost:8080");

export class ApiClientError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly statusCode: number,
  ) {
    super(message);
    this.name = "ApiClientError";
  }
}

let onUnauthorized: (() => void) | null = null;

export function setOnUnauthorized(callback: () => void) {
  onUnauthorized = callback;
}

let refreshPromise: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  const refreshToken = localStorage.getItem("refresh_token");
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${BASE_URL}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) return false;

    const data = (await res.json()) as {
      access_token: string;
      refresh_token: string;
    };
    localStorage.setItem("access_token", data.access_token);
    localStorage.setItem("refresh_token", data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

async function doFetch(path: string, options?: RequestInit): Promise<Response> {
  const url = `${BASE_URL}/api/v1${path}`;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  // Public viewer routes are anonymous by design — never leak the editor's
  // bearer token onto them.
  const token = localStorage.getItem("access_token");
  if (token && !path.startsWith("/public/")) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  return fetch(url, {
    ...options,
    headers: {
      ...headers,
      ...(options?.headers as Record<string, string>),
    },
  });
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  let res = await doFetch(path, options);

  if (res.status === 401) {
    if (!refreshPromise) {
      refreshPromise = tryRefresh().finally(() => {
        refreshPromise = null;
      });
    }

    const refreshed = await refreshPromise;
    if (refreshed) {
      res = await doFetch(path, options);
      if (res.status === 401) {
        localStorage.removeItem("access_token");
        localStorage.removeItem("refresh_token");
        onUnauthorized?.();
        throw new ApiClientError("auth.unauthorized", "session expired", 401);
      }
    } else {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      onUnauthorized?.();
      throw new ApiClientError("auth.unauthorized", "session expired", 401);
    }
  }

  if (!res.ok) {
    let code = "unknown";
    let message = res.statusText;

    try {
      const body = (await res.json()) as ApiError;
      code = body.error.code;
      message = body.error.message;
    } catch {
      // use defaults
    }

    throw new ApiClientError(code, message, res.status);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

export const apiClient = {
  get: <T>(path: string) => request<T>(path),

  post: <T>(path: string, body: unknown, headers?: Record<string, string>) =>
    request<T>(path, {
      method: "POST",
      body: JSON.stringify(body),
      headers,
    }),

  put: <T>(path: string, body: unknown) =>
    request<T>(path, {
      method: "PUT",
      body: JSON.stringify(body),
    }),

  patch: <T>(path: string, body: unknown) =>
    request<T>(path, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),

  delete: <T>(path: string) =>
    request<T>(path, {
      method: "DELETE",
    }),
};
