interface NamedUser {
  readonly first_name: string | null;
  readonly last_name: string | null;
  readonly email: string;
}

export function getDisplayName(user: NamedUser): string {
  const full = [user.first_name, user.last_name].filter(Boolean).join(" ");
  return full || user.email;
}

export function getInitials(user: NamedUser): string {
  if (user.first_name && user.last_name) {
    return `${user.first_name[0]}${user.last_name[0]}`.toUpperCase();
  }
  if (user.first_name) return user.first_name[0]!.toUpperCase();
  return user.email[0]!.toUpperCase();
}

/** Ініціали з довільного імені-рядка (виконавець task — вільний текст,
 * не NamedUser): «Sam R.» → «SR», одне слово «Alex» → «AL». */
export function getNameInitials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  if (words.length >= 2) {
    return `${words[0]![0]!}${words[1]![0]!}`.toUpperCase();
  }
  return name.trim().slice(0, 2).toUpperCase();
}
