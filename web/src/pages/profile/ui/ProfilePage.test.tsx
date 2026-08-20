import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import ProfilePage from "./ProfilePage";

vi.mock("@/app/providers/auth", () => ({
  useAuth: () => ({
    user: {
      id: "u1",
      email: "jane@example.com",
      first_name: "Jane",
      last_name: "Doe",
      avatar_url: null,
      role: "member",
      position: "Engineer",
      department: "Platform",
      bio: "Hello world",
      timezone: "Europe/Kyiv",
    },
    isLoading: false,
    login: vi.fn(),
    logout: vi.fn(),
    setTokens: vi.fn(),
    fetchUser: vi.fn(),
  }),
}));

vi.mock("@/features/edit-profile/api/profileApi", () => ({
  profileApi: {
    updateProfile: vi.fn(),
  },
}));

describe("ProfilePage", () => {
  it("renders user info card and edit form", () => {
    render(
      <MemoryRouter>
        <ProfilePage />
      </MemoryRouter>,
    );

    expect(screen.getByText("Profile")).toBeInTheDocument();
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("Engineer · Platform")).toBeInTheDocument();
    expect(screen.getAllByText("Hello world").length).toBeGreaterThan(0);
    expect(screen.getByText("Edit profile")).toBeInTheDocument();
  });
});
