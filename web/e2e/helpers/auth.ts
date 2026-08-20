/**
 * JWT scaffolding helper for authenticated E2E specs.
 *
 * Generates a valid HS256 JWT signed with JWT_SECRET (read at runtime),
 * injects it into localStorage as `access_token`, and navigates to the
 * requested start URL.
 *
 * Token shape mirrors the production token produced by
 * the API auth module jwt service:
 *   { user_id, email, role, token_type: "access", iat, exp: iat + 3600 }
 *
 * NOTE: jwt_service.go uses RegisteredClaims (iat/exp as NumericDate) and
 * the custom fields user_id / email / role / token_type at the top level.
 */
import type { Page } from "@playwright/test";
import jwt from "jsonwebtoken";

const DEFAULT_ADMIN_USER_ID = "019d1bae-c4a8-7422-bc5d-549dad0aa678";
const DEFAULT_ADMIN_EMAIL = "genkovi4@gmail.com";
const DEFAULT_ADMIN_ROLE = "member";

export interface LoginOptions {
  userId?: string;
  email?: string;
  role?: string;
  /** Page to navigate to after injecting the token. Defaults to "/login" (establishes same-origin context). */
  landingPath?: string;
}

/**
 * Signs a JWT with the runtime JWT_SECRET and sets it in localStorage.
 *
 * Call this before navigating to any protected route.
 */
export async function loginAsAdmin(page: Page, opts: LoginOptions = {}): Promise<void> {
  const secret = process.env.JWT_SECRET;
  if (!secret) {
    throw new Error(
      "JWT_SECRET environment variable is not set. " +
        "Export it before running authenticated E2E tests.",
    );
  }

  const userId = opts.userId ?? DEFAULT_ADMIN_USER_ID;
  const email = opts.email ?? DEFAULT_ADMIN_EMAIL;
  const role = opts.role ?? DEFAULT_ADMIN_ROLE;
  const landingPath = opts.landingPath ?? "/login";

  const iat = Math.floor(Date.now() / 1000);
  const exp = iat + 3600;

  const token = jwt.sign(
    {
      user_id: userId,
      email,
      role,
      token_type: "access",
      iat,
      exp,
    },
    secret,
    { algorithm: "HS256" },
  );

  // Navigate to the landing path to establish a same-origin context so that
  // localStorage.setItem works for the target origin.
  await page.goto(landingPath);
  await page.waitForLoadState("domcontentloaded");

  await page.evaluate(
    ({ accessToken }: { accessToken: string }) => {
      localStorage.setItem("access_token", accessToken);
    },
    { accessToken: token },
  );
}
