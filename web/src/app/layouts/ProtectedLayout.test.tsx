import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, type UIMatch } from "react-router";

vi.mock("@/app/providers/auth", () => ({
  useAuth: vi.fn(),
}));

vi.mock("@/app/providers/theme", () => ({
  useTheme: () => ({ theme: "system", resolvedTheme: "light", setTheme: vi.fn() }),
}));

// useMatches() only works inside a data router. A real one (e.g.
// createRoutesStub) routes every navigation — even a plain client-side one
// in a memory router — through React Router 7's internal Request/fetch
// pipeline, which in this repo's Node+jsdom test combination throws
// `RequestInit: Expected signal to be an instance of AbortSignal`: jsdom
// installs its own AbortController/AbortSignal on the global scope, and
// that's a different class than the one Node's native Request validates
// against — a cross-realm instanceof mismatch (version-sensitive: it
// happened to pass on Node 23 locally but broke in CI's Node 24). This
// layout has no loaders/actions of its own — it only reads route `handle`
// — so it doesn't need real data-router navigation machinery; mock
// useMatches() directly instead and keep the plain MemoryRouter harness.
const mockedUseMatches = vi.fn<() => UIMatch[]>(() => []);
vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return { ...actual, useMatches: () => mockedUseMatches() };
});

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

function renderLayout() {
  return render(
    <MemoryRouter initialEntries={["/board"]}>
      <Routes>
        <Route element={<ProtectedLayout />}>
          <Route path="/board" element={<div>Dashboard content</div>} />
        </Route>
        <Route path="/" element={<div>Login page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mockedUseMatches.mockReturnValue([]);
});

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

// A1: a board route opts out of the default centered column by exporting
// `handle = { fullWidth: true }` — ProtectedLayout reads it via
// useMatches() and drops max-w-5xl from <main> when any matched route
// carries it.
describe("ProtectedLayout — fullWidth route handle (A1)", () => {
  it("centers <main> with max-w-5xl when no matched route opts into fullWidth", () => {
    mockedUseAuth.mockReturnValue(baseAuth);
    mockedUseMatches.mockReturnValue([]);
    renderLayout();

    expect(screen.getByText("Dashboard content").closest("main")).toHaveClass(
      "mx-auto",
      "max-w-5xl",
    );
  });

  it("drops max-w-5xl from <main> when a matched route's handle sets fullWidth", () => {
    mockedUseAuth.mockReturnValue(baseAuth);
    mockedUseMatches.mockReturnValue([
      {
        id: "board",
        pathname: "/board",
        params: {},
        data: undefined,
        loaderData: undefined,
        handle: { fullWidth: true },
      },
    ]);
    renderLayout();

    const main = screen.getByText("Dashboard content").closest("main");
    expect(main).not.toHaveClass("max-w-5xl");
    expect(main).not.toHaveClass("mx-auto");
  });
});
