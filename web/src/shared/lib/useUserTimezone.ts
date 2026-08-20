import { useAuth } from "@/app/providers/auth";

export function useUserTimezone(): string {
  const { user } = useAuth();
  return user?.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone;
}