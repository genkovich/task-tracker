import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import LoginPage from "./LoginPage";

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

describe("LoginPage", () => {
  function renderPage() {
    return render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>,
    );
  }

  it("renders the my.app wordmark", () => {
    renderPage();

    expect(screen.getByText("my")).toBeInTheDocument();
    expect(screen.getByText("app")).toBeInTheDocument();
  });

  it("has a sign-in heading with a sentence-case label", () => {
    renderPage();

    expect(screen.getByRole("heading", { name: /sign in/i })).toBeInTheDocument();
  });

  it("renders the Google sign-in button", () => {
    renderPage();

    expect(screen.getByRole("button", { name: /sign in with google/i })).toBeInTheDocument();
  });

  it("links back to the landing page from the wordmark", () => {
    renderPage();

    const wordmarkLink = screen.getByRole("link", { name: /my\.app/i });
    expect(wordmarkLink).toHaveAttribute("href", "/");
  });
});
