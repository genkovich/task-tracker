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

  it("renders the product name as the page heading", () => {
    renderPage();

    expect(screen.getByRole("heading", { name: "Task Tracker" })).toBeInTheDocument();
  });

  it("renders a one-line value proposition", () => {
    renderPage();

    expect(
      screen.getByText("Every board, every task, one shared workspace."),
    ).toBeInTheDocument();
  });

  it("renders the Google sign-in button", () => {
    renderPage();

    expect(screen.getByRole("button", { name: /sign in with google/i })).toBeInTheDocument();
  });
});
