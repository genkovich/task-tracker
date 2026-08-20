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
