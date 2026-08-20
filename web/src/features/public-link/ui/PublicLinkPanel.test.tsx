import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PublicLinkPanel } from "./PublicLinkPanel";
import { ApiClientError } from "@/shared/api/client";

// api/model do not exist yet (RED phase) — mocked at the module boundary the
// component is expected to depend on (T17 "What": api/ calls issuePublicLink /
// revokePublicLink against POST/DELETE /api/v1/board/public-link).
const mockIssue = vi.fn();
const mockRevoke = vi.fn();

vi.mock("@/features/public-link/api/publicLinkApi", () => ({
  publicLinkApi: {
    issue: (...args: unknown[]) => mockIssue(...args),
    revoke: (...args: unknown[]) => mockRevoke(...args),
  },
}));

const issuedLink = {
  token: "opaque-token-abc123",
  created_at: "2026-08-20T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
});

async function openPanel(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: /поділитись|share/i }));
}

describe("PublicLinkPanel", () => {
  // AC-07 (US-05) happy path — spec.md §5: "team member запитує public link"
  // -> "система видає team member лінк, що завжди показує поточний стан board
  // лише на перегляд". DoD: "issuing a link from the no-link state renders the
  // returned URL".
  it("issuing a link from the no-link state renders the returned URL", async () => {
    const user = userEvent.setup();
    mockIssue.mockResolvedValue(issuedLink);

    render(<PublicLinkPanel />);
    await openPanel(user);

    // No-link state: an issue action, no revoke action yet.
    expect(
      screen.getByRole("button", { name: /отримати лінк|get link/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /відкликати|revoke/i }),
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /отримати лінк|get link/i }));

    await waitFor(() => {
      expect(mockIssue).toHaveBeenCalled();
    });

    // The observable outcome the AC names: the returned URL is shown to the
    // team member (not just the raw token, and not merely "success").
    await waitFor(() => {
      expect(screen.getByText(/opaque-token-abc123/)).toBeInTheDocument();
    });

    // Now in active-link state: a revoke action is available.
    expect(
      screen.getByRole("button", { name: /відкликати|revoke/i }),
    ).toBeInTheDocument();
  });

  // DoD extra case guarding AC-07's stated precondition ("board ще не має
  // активного лінка" / openapi.yaml issuePublicLink 409 board.link_already_active):
  // issuing while a link already exists must surface the conflict, not silently
  // succeed or crash.
  it("surfaces a 409 conflict as a visible error when issuing while a link already exists", async () => {
    const user = userEvent.setup();
    mockIssue.mockRejectedValue(
      new ApiClientError(
        "board.link_already_active",
        "an active public link already exists — revoke it first",
        409,
      ),
    );

    render(<PublicLinkPanel />);
    await openPanel(user);

    await user.click(screen.getByRole("button", { name: /отримати лінк|get link/i }));

    await waitFor(() => {
      expect(
        screen.getByText(/an active public link already exists — revoke it first/i),
      ).toBeInTheDocument();
    });
  });

  // AC-08 (US-06) happy path — spec.md §5: "team member відкликає цей лінк" ->
  // "система припиняє показувати стан board за цим лінком". DoD: "revoking from
  // the active-link state clears the panel back to no-link".
  it("revoking from the active-link state clears the panel back to no-link", async () => {
    const user = userEvent.setup();
    mockIssue.mockResolvedValue(issuedLink);
    mockRevoke.mockResolvedValue(undefined);

    render(<PublicLinkPanel />);
    await openPanel(user);
    await user.click(screen.getByRole("button", { name: /отримати лінк|get link/i }));

    await waitFor(() => {
      expect(screen.getByText(/opaque-token-abc123/)).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /відкликати|revoke/i }));

    await waitFor(() => {
      expect(mockRevoke).toHaveBeenCalled();
    });

    // Observable outcome: back to no-link state — URL gone, issue action back,
    // revoke action gone.
    await waitFor(() => {
      expect(screen.queryByText(/opaque-token-abc123/)).not.toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: /отримати лінк|get link/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /відкликати|revoke/i }),
    ).not.toBeInTheDocument();
  });
});
