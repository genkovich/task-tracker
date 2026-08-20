export interface CurrentUser {
  id: string;
  email: string;
  first_name: string | null;
  last_name: string | null;
  avatar_url: string | null;
  role: "admin" | "member";
  position: string | null;
  department: string | null;
  bio: string | null;
  timezone: string | null;
}

export interface User extends CurrentUser {
  created_at: string;
  updated_at: string;
}
