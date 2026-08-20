import type { PublicLink } from "../api/publicLinkApi";

export type PublicLinkState = { status: "none" } | { status: "active"; link: PublicLink };

export const noLink: PublicLinkState = { status: "none" };

export function activeLink(link: PublicLink): PublicLinkState {
  return { status: "active", link };
}
