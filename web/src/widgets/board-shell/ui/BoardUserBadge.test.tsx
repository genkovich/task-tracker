import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BoardUserBadge } from "./BoardUserBadge";

// useAuth is the only dependency — mocked so the test drives the two states
// (signed-in / no user) without a provider tree.
const mockLogout = vi.fn();
let mockUser: { first_name?: string; last_name?: string; email: string } | null = null;

vi.mock("@/app/providers/auth", () => ({
  useAuth: () => ({ user: mockUser, logout: mockLogout }),
}));

beforeEach(() => {
  vi.clearAllMocks();
  mockUser = { first_name: "Admin", last_name: "Example", email: "admin@example.com" };
});

describe("BoardUserBadge — user menu on the board header", () => {
  it("opens a menu with the user's name and a log out action on avatar click", async () => {
    const user = userEvent.setup();
    render(<BoardUserBadge />);

    await user.click(screen.getByRole("button", { name: "Відкрити меню користувача" }));

    expect(await screen.findByText("Admin Example")).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: /вийти/i })).toBeInTheDocument();
  });

  it("calls useAuth().logout when «Вийти» is selected", async () => {
    const user = userEvent.setup();
    render(<BoardUserBadge />);

    await user.click(screen.getByRole("button", { name: "Відкрити меню користувача" }));
    await user.click(await screen.findByRole("menuitem", { name: /вийти/i }));

    expect(mockLogout).toHaveBeenCalledTimes(1);
  });

  it("renders a plain badge without a menu when there is no user", () => {
    mockUser = null;
    render(<BoardUserBadge />);

    expect(
      screen.queryByRole("button", { name: "Відкрити меню користувача" }),
    ).not.toBeInTheDocument();
  });
});
