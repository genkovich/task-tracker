import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { ProfileForm } from "./ProfileForm";
import type { CurrentUser } from "@/entities/user/model/types";

const mockFetchUser = vi.fn();

vi.mock("@/app/providers/auth", () => ({
  useAuth: () => ({
    user: null,
    isLoading: false,
    login: vi.fn(),
    logout: vi.fn(),
    setTokens: vi.fn(),
    fetchUser: mockFetchUser,
  }),
}));

const mockUpdateProfile = vi.fn();

vi.mock("@/features/edit-profile/api/profileApi", () => ({
  profileApi: {
    updateProfile: (...args: unknown[]) => mockUpdateProfile(...args),
  },
}));

const testUser: CurrentUser = {
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
};

beforeEach(() => {
  vi.clearAllMocks();
  mockUpdateProfile.mockResolvedValue(testUser);
  mockFetchUser.mockResolvedValue(undefined);
});

describe("ProfileForm", () => {
  it("renders form fields pre-filled from user", () => {
    render(
      <MemoryRouter>
        <ProfileForm user={testUser} />
      </MemoryRouter>,
    );

    expect(screen.getByLabelText("First name")).toHaveValue("Jane");
    expect(screen.getByLabelText("Last name")).toHaveValue("Doe");
    expect(screen.getByLabelText("Position")).toHaveValue("Engineer");
    expect(screen.getByLabelText("Department")).toHaveValue("Platform");
    expect(screen.getByLabelText("Bio")).toHaveValue("Hello world");
    // Timezone is now a combobox — verify it displays the value
    expect(screen.getByRole("combobox", { name: "Timezone" })).toHaveTextContent(
      "Europe/Kyiv",
    );
  });

  it("submits form and calls API then fetchUser", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ProfileForm user={testUser} />
      </MemoryRouter>,
    );

    await user.clear(screen.getByLabelText("Position"));
    await user.type(screen.getByLabelText("Position"), "Staff Engineer");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(mockUpdateProfile).toHaveBeenCalledWith({
        first_name: "Jane",
        last_name: "Doe",
        position: "Staff Engineer",
        department: "Platform",
        bio: "Hello world",
        timezone: "Europe/Kyiv",
      });
    });

    await waitFor(() => {
      expect(mockFetchUser).toHaveBeenCalled();
    });

    expect(screen.getByText("Saved")).toBeInTheDocument();
  });

  it("shows error on API failure", async () => {
    const user = userEvent.setup();
    mockUpdateProfile.mockRejectedValue(new Error("Something went wrong"));

    render(
      <MemoryRouter>
        <ProfileForm user={testUser} />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    });
  });

  it("renders empty fields for null user values", () => {
    const emptyUser: CurrentUser = {
      ...testUser,
      first_name: null,
      last_name: null,
      position: null,
      department: null,
      bio: null,
      timezone: null,
    };

    render(
      <MemoryRouter>
        <ProfileForm user={emptyUser} />
      </MemoryRouter>,
    );

    expect(screen.getByLabelText("First name")).toHaveValue("");
    expect(screen.getByLabelText("Last name")).toHaveValue("");
    expect(screen.getByLabelText("Position")).toHaveValue("");
    expect(screen.getByLabelText("Department")).toHaveValue("");
    expect(screen.getByLabelText("Bio")).toHaveValue("");
    // Timezone combobox shows placeholder when empty
    expect(screen.getByRole("combobox", { name: "Timezone" })).toHaveTextContent(
      "Select timezone...",
    );
  });

  it("has maxLength on text inputs", () => {
    render(
      <MemoryRouter>
        <ProfileForm user={testUser} />
      </MemoryRouter>,
    );

    expect(screen.getByLabelText("First name")).toHaveAttribute("maxlength", "100");
    expect(screen.getByLabelText("Last name")).toHaveAttribute("maxlength", "100");
    expect(screen.getByLabelText("Position")).toHaveAttribute("maxlength", "200");
    expect(screen.getByLabelText("Department")).toHaveAttribute("maxlength", "200");
  });

  it("selects timezone from combobox", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <ProfileForm user={testUser} />
      </MemoryRouter>,
    );

    // Open combobox
    await user.click(screen.getByRole("combobox", { name: "Timezone" }));

    // Type to filter
    const searchInput = screen.getByPlaceholderText("Search timezone...");
    await user.type(searchInput, "America/New_York");

    // Select an option
    await user.click(screen.getByRole("option", { name: "America/New_York" }));

    // Verify combobox shows selected value
    expect(screen.getByRole("combobox", { name: "Timezone" })).toHaveTextContent(
      "America/New_York",
    );
  });
});
