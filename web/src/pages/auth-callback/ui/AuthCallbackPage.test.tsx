import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import AuthCallbackPage from "./AuthCallbackPage";

const mockNavigate = vi.fn();
const mockSetTokens = vi.fn();
const mockFetchUser = vi.fn();

vi.mock("react-router", async () => {
  const actual = await vi.importActual("react-router");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useSearchParams: () => [currentSearchParams],
  };
});

vi.mock("@/app/providers/auth", () => ({
  useAuth: () => ({
    user: null,
    isLoading: false,
    login: vi.fn(),
    logout: vi.fn(),
    setTokens: mockSetTokens,
    fetchUser: mockFetchUser,
  }),
}));

let currentSearchParams: URLSearchParams;

beforeEach(() => {
  vi.clearAllMocks();
  mockFetchUser.mockResolvedValue(undefined);
  currentSearchParams = new URLSearchParams();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("AuthCallbackPage", () => {
  it("exchanges code and redirects to dashboard on success", async () => {
    currentSearchParams = new URLSearchParams("code=test-auth-code-123");

    const mockResponse = {
      ok: true,
      json: () =>
        Promise.resolve({
          access_token: "abc123",
          refresh_token: "def456",
          expires_in: 900,
        }),
    };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(mockResponse));

    render(
      <MemoryRouter>
        <AuthCallbackPage />
      </MemoryRouter>,
    );

    expect(screen.getByText("Signing in...")).toBeInTheDocument();

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining("/api/v1/auth/exchange"),
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ code: "test-auth-code-123" }),
        }),
      );
    });

    await waitFor(() => {
      expect(mockSetTokens).toHaveBeenCalledWith("abc123", "def456");
    });

    await waitFor(() => {
      expect(mockFetchUser).toHaveBeenCalled();
    });

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/dashboard", {
        replace: true,
      });
    });
  });

  it("redirects to login when no code in query params", async () => {
    currentSearchParams = new URLSearchParams();

    render(
      <MemoryRouter>
        <AuthCallbackPage />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/login", { replace: true });
    });
  });

  it("redirects to login when exchange fails", async () => {
    currentSearchParams = new URLSearchParams("code=test-auth-code-123");

    const mockResponse = {
      ok: false,
      status: 401,
      json: () =>
        Promise.resolve({
          error: { code: "auth.invalid_exchange_code", message: "invalid" },
        }),
    };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(mockResponse));

    render(
      <MemoryRouter>
        <AuthCallbackPage />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/login", { replace: true });
    });
  });

  it("redirects to login when fetchUser fails after exchange", async () => {
    currentSearchParams = new URLSearchParams("code=test-auth-code-123");

    const mockResponse = {
      ok: true,
      json: () =>
        Promise.resolve({
          access_token: "abc123",
          refresh_token: "def456",
          expires_in: 900,
        }),
    };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(mockResponse));
    mockFetchUser.mockRejectedValue(new Error("Unauthorized"));

    render(
      <MemoryRouter>
        <AuthCallbackPage />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/login", { replace: true });
    });
  });
});
