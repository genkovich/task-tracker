import { formatInTimeZone } from "date-fns-tz";

export function tzShortLabel(tz: string): string {
  const parts = tz.split("/");
  return parts[parts.length - 1]!.replace(/_/g, " ");
}

export function getBrowserTimezone(): string | null {
  if (typeof Intl === "undefined") return null;
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || null;
  } catch {
    return null;
  }
}

interface FormatOptions {
  readonly browserTz?: string | null;
}

/**
 * Formats `date` in the user's preferred timezone (or browser local when
 * the user hasn't set one). When the user's timezone differs from the
 * browser local one we append a short label like " (Kyiv)" so the time
 * is unambiguous to readers from other zones.
 */
export function formatTimeForUser(
  date: Date,
  formatStr: string,
  userTz: string | null | undefined,
  options?: FormatOptions,
): string {
  const browserTz = options?.browserTz === undefined ? getBrowserTimezone() : options.browserTz;
  const effective = userTz || browserTz;
  if (!effective) {
    return date.toString();
  }
  const formatted = formatInTimeZone(date, effective, formatStr);
  if (userTz && browserTz && userTz !== browserTz) {
    return `${formatted} (${tzShortLabel(userTz)})`;
  }
  return formatted;
}
