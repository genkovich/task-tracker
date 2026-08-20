const ALLOWED_PROTOCOLS = new Set(["http:", "https:", "mailto:"]);

export function isSafeUrl(url: string): boolean {
  try {
    const parsed = new URL(url);
    return ALLOWED_PROTOCOLS.has(parsed.protocol);
  } catch {
    return false;
  }
}
