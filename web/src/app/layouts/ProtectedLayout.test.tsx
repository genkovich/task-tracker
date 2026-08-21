import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { createRoutesStub } from "react-router";

vi.mock("@/app/providers/auth", () => ({
  useAuth: vi.fn(),
}));

vi.mock("@/app/providers/theme", () => ({
  useTheme: () => ({ theme: "system", resolvedTheme: "light", setTheme: vi.fn() }),
}));

import { useAuth } from "@/app/providers/auth";
import ProtectedLayout from "./ProtectedLayout";

const mockedUseAuth = vi.mocked(useAuth);

const authenticatedUser = {
  id: "123",
  email: "test@test.com",
  first_name: "Test",
  last_name: "User",
  avatar_url: null,
  role: "member" as const,
  position: null,
  department: null,
  bio: null,
  timezone: null,
};

const baseAuth = {
  user: authenticatedUser,
  isAdmin: false,
  isLoading: false,
  login: vi.fn(),
  logout: vi.fn(),
  setTokens: vi.fn(),
  fetchUser: vi.fn(),
};

// ProtectedLayout reads useMatches() (route `handle`, A1 fullWidth) — a hook
// that only works inside a data router. createRoutesStub builds one from a
// plain route tree, matching how @react-router/dev wires the real app more
// closely than the classic <MemoryRouter><Routes> harness this test used
// before.
function renderLayout() {
  const Stub = createRoutesStub([
    {
      Component: ProtectedLayout,
      children: [{ path: "/board", Component: () => <div>Dashboard content</div> }],
    },
    { path: "/", Component: () => <div>Login page</div> },
  ]);
  return render(<Stub initialEntries={["/board"]} />);
}

describe("ProtectedLayout", () => {
  it("shows a spinner while auth is loading", () => {
    mockedUseAuth.mockReturnValue({ ...baseAuth, user: null, isLoading: true });
    renderLayout();

    expect(screen.queryByText("Dashboard content")).not.toBeInTheDocument();
    expect(screen.queryByText("Login page")).not.toBeInTheDocument();
  });

  it("redirects to / when unauthenticated", () => {
    mockedUseAuth.mockReturnValue({ ...baseAuth, user: null });
    renderLayout();

    expect(screen.getByText("Login page")).toBeInTheDocument();
  });

  it("renders the outlet for an authenticated user", () => {
    mockedUseAuth.mockReturnValue(baseAuth);
    renderLayout();

    expect(screen.getByText("Dashboard content")).toBeInTheDocument();
  });
});
