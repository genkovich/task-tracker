import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { apiClient, ApiClientError, setOnUnauthorized, BASE_URL, validateBaseUrl } from "./client";

describe("validateBaseUrl", () => {
  it("accepts valid http URL", () => {
    expect(validateBaseUrl("http://localhost:8080")).toBe("http://localhost:8080");
  });

  it("accepts valid https URL", () => {
    expect(validateBaseUrl("https://api.example.com")).toBe("https://api.example.com");
  });

  it("throws on invalid URL without protocol", () => {
    expect(() => validateBaseUrl("localhost:8080")).toThrow("Invalid BASE_URL");
  });

  it("throws on empty string", () => {
    expect(() => validateBaseUrl("")).toThrow("Invalid BASE_URL");
  });

  it("throws on ftp URL", () => {
    expect(() => validateBaseUrl("ftp://files.example.com")).toThrow("Invalid BASE_URL");
  });

  it("throws on relative path", () => {
    expect(() => validateBaseUrl("/api")).toThrow("Invalid BASE_URL");
  });
});

describe("apiClient", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("makes a GET request with auth header", async () => {
    localStorage.setItem("access_token", "test-token");

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ id: 1 }),
    });

    const result = await apiClient.get("/users");

    expect(globalThis.fetch).toHaveBeenCalledWith(
      `${BASE_URL}/api/v1/users`,
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer test-token",
        }),
      }),
    );
    expect(result).toEqual({ id: 1 });
  });

  // Review 2026-08-21 root K: public viewer routes are anonymous by design —
  // an editor's bearer token must never leak onto them.
  it("does not attach the Authorization header on /public/ paths", async () => {
    localStorage.setItem("access_token", "test-token");

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ columns: [] }),
    });

    await apiClient.get("/public/some-token/board");

    const [, options] = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect((options.headers as Record<string, string>).Authorization).toBeUndefined();
  });

  it("throws ApiClientError on non-401 error", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      statusText: "Not Found",
      json: () =>
        Promise.resolve({
          error: { code: "user.not_found", message: "user not found" },
        }),
    });

    await expect(apiClient.get("/users/123")).rejects.toThrow(ApiClientError);
  });

  describe("token refresh on 401", () => {
    it("refreshes token and retries on 401", async () => {
      localStorage.setItem("access_token", "expired-token");
      localStorage.setItem("refresh_token", "valid-refresh");

      let callCount = 0;
      globalThis.fetch = vi.fn().mockImplementation((url: string) => {
        if (typeof url === "string" && url.includes("/auth/refresh")) {
          return Promise.resolve({
            ok: true,
            status: 200,
            json: () =>
              Promise.resolve({
                access_token: "new-access",
                refresh_token: "new-refresh",
              }),
          });
        }

        callCount++;
        if (callCount === 1) {
          return Promise.resolve({
            ok: false,
            status: 401,
            statusText: "Unauthorized",
            json: () =>
              Promise.resolve({
                error: { code: "auth.invalid_token", message: "invalid token" },
              }),
          });
        }

        return Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.resolve({ data: "success" }),
        });
      });

      const result = await apiClient.get("/auth/me");
      expect(result).toEqual({ data: "success" });
      expect(localStorage.getItem("access_token")).toBe("new-access");
      expect(localStorage.getItem("refresh_token")).toBe("new-refresh");
    });

    it("calls onUnauthorized and clears tokens when refresh fails", async () => {
      localStorage.setItem("access_token", "expired-token");
      localStorage.setItem("refresh_token", "invalid-refresh");

      const onUnauthorized = vi.fn();
      setOnUnauthorized(onUnauthorized);

      globalThis.fetch = vi.fn().mockImplementation((url: string) => {
        if (typeof url === "string" && url.includes("/auth/refresh")) {
          return Promise.resolve({ ok: false, status: 401 });
        }

        return Promise.resolve({
          ok: false,
          status: 401,
          statusText: "Unauthorized",
          json: () =>
            Promise.resolve({
              error: { code: "auth.invalid_token", message: "invalid token" },
            }),
        });
      });

      await expect(apiClient.get("/auth/me")).rejects.toThrow(ApiClientError);
      expect(onUnauthorized).toHaveBeenCalled();
      expect(localStorage.getItem("access_token")).toBeNull();
      expect(localStorage.getItem("refresh_token")).toBeNull();
    });
  });
});
