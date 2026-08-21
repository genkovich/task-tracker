const MONTHS_SHORT_UK = [
  "січ",
  "лют",
  "бер",
  "квіт",
  "трав",
  "черв",
  "лип",
  "серп",
  "вер",
  "жовт",
  "лист",
  "груд",
];

/** Today as `YYYY-MM-DD` in the viewer's own timezone — a deadline is a
 * calendar day, so "overdue" has to be decided against the viewer's calendar,
 * not against UTC midnight. */
export function todayISO(now: Date = new Date()): string {
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

/** TSK-06: a deadline strictly before today. Compared as strings on purpose —
 * `YYYY-MM-DD` sorts chronologically, so this needs no Date parsing and can't
 * drift by a timezone offset. */
export function isOverdue(dueDate: string, now: Date = new Date()): boolean {
  return dueDate < todayISO(now);
}

/** `2026-09-01` → `1 вер`, with the year appended when it isn't the current
 * one (a deadline in another year read as this year would be misleading). */
export function formatDueDate(dueDate: string, now: Date = new Date()): string {
  const [year, month, day] = dueDate.split("-").map(Number);
  if (!year || !month || !day) return dueDate;

  const label = `${day} ${MONTHS_SHORT_UK[month - 1] ?? ""}`.trim();
  return year === now.getFullYear() ? label : `${label} ${year}`;
}
