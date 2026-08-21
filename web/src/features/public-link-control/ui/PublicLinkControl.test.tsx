import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PublicLinkControl } from "./PublicLinkControl";

const mockGetActiveLink = vi.fn();
const mockGenerateLink = vi.fn();
const mockDisableLink = vi.fn();

vi.mock("../api/publicLinkApi", () => ({
  publicLinkApi: {
    getActiveLink: () => mockGetActiveLink(),
    generateLink: () => mockGenerateLink(),
    disableLink: (...args: unknown[]) => mockDisableLink(...args),
  },
}));

beforeEach(() => {
  vi.clearAllMocks();
});

describe("PublicLinkControl", () => {
  it("shows 'Get link' when no link is active (AC-09 precondition)", async () => {
    mockGetActiveLink.mockResolvedValue(null);
    render(<PublicLinkControl />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Get link" })).toBeInTheDocument();
    });
  });

  it("generates a link and shows it, valid until disabled (AC-09)", async () => {
    const user = userEvent.setup();
    mockGetActiveLink.mockResolvedValue(null);
    mockGenerateLink.mockResolvedValue({
      id: "l1",
      token: "abc123",
      disabled_at: null,
      created_at: "2026-08-21T10:00:00Z",
    });

    render(<PublicLinkControl />);
    await waitFor(() => screen.getByRole("button", { name: "Get link" }));

    await user.click(screen.getByRole("button", { name: "Get link" }));

    await waitFor(() => {
      expect(mockGenerateLink).toHaveBeenCalled();
    });
    expect(screen.getByDisplayValue(/\/b\/abc123$/)).toBeInTheDocument();
  });

  it("disables the active link (AC-04)", async () => {
    const user = userEvent.setup();
    mockGetActiveLink.mockResolvedValue({
      id: "l1",
      token: "abc123",
      disabled_at: null,
      created_at: "2026-08-21T10:00:00Z",
    });
    mockDisableLink.mockResolvedValue({});

    render(<PublicLinkControl />);
    await waitFor(() => screen.getByRole("button", { name: "Disable link" }));

    await user.click(screen.getByRole("button", { name: "Disable link" }));

    await waitFor(() => {
      expect(mockDisableLink).toHaveBeenCalledWith("l1");
    });
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Get link" })).toBeInTheDocument();
    });
  });

  it("shows a status label (no link input) when an active link was fetched without a token", async () => {
    mockGetActiveLink.mockResolvedValue({
      id: "l1",
      disabled_at: null,
      created_at: "2026-08-21T10:00:00Z",
    });

    render(<PublicLinkControl />);

    await waitFor(() => {
      expect(screen.getByText("Public link is active")).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Disable link" })).toBeInTheDocument();
  });
});
