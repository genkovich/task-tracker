import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { GoogleLoginButton } from "./GoogleLoginButton";

vi.mock("@/app/providers/auth", () => ({
  useAuth: () => ({
    user: null,
    isLoading: false,
    login: vi.fn(),
    logout: vi.fn(),
    setTokens: vi.fn(),
    fetchUser: vi.fn(),
  }),
}));

describe("GoogleLoginButton", () => {
  it("renders the login button", () => {
    render(<GoogleLoginButton />);
    expect(screen.getByRole("button", { name: /sign in with google/i })).toBeInTheDocument();
  });
});
