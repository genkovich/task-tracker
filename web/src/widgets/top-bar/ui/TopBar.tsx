import { useLocation } from "react-router";
import { NavLink } from "react-router";
import { LogOut, Menu, Monitor, Moon, Sun, User } from "lucide-react";
import { useAuth } from "@/app/providers/auth";
import { useTheme, type Theme } from "@/app/providers/theme";
import { cn } from "@/shared/lib/utils";
import { UserAvatar } from "@/shared/ui/UserAvatar";
import { Button } from "@/shared/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";

const THEME_META: Record<Theme, { next: Theme; label: string; Icon: typeof Sun }> = {
  light: { next: "dark", label: "Switch to dark theme", Icon: Sun },
  dark: { next: "system", label: "Switch to system theme", Icon: Moon },
  system: { next: "light", label: "Switch to light theme", Icon: Monitor },
};

interface TopBarProps {
  onMenuClick?: () => void;
  className?: string;
}

const PAGE_NAMES: Record<string, string> = {
  "/dashboard": "Dashboard",
  "/profile": "Profile",
};

function getPageName(pathname: string): string {
  if (PAGE_NAMES[pathname]) return PAGE_NAMES[pathname];
  return "Task Tracker";
}

export function TopBar({ onMenuClick, className }: TopBarProps) {
  const location = useLocation();
  const { user, logout } = useAuth();
  const { theme, setTheme } = useTheme();
  const pageName = getPageName(location.pathname);

  const displayName = user?.first_name || user?.email.split("@")[0];
  const initials = user?.first_name
    ? user.first_name.charAt(0).toUpperCase()
    : user?.email.charAt(0).toUpperCase();

  const themeMeta = THEME_META[theme];
  const ThemeIcon = themeMeta.Icon;

  return (
    <div className={cn("flex h-12 items-center gap-3 border-b px-4", className)}>
      {onMenuClick && (
        <Button
          variant="ghost"
          size="icon"
          className="size-8 md:hidden"
          onClick={onMenuClick}
          aria-label="Open navigation menu"
        >
          <Menu className="size-4" />
        </Button>
      )}
      <h2 className="flex-1 text-sm font-medium text-muted-foreground">{pageName}</h2>

      <Button
        variant="ghost"
        size="icon"
        className="size-8"
        aria-label={themeMeta.label}
        onClick={() => setTheme(themeMeta.next)}
      >
        <ThemeIcon
          key={theme}
          className="size-4 animate-in fade-in zoom-in-50 spin-in-12 duration-200"
        />
      </Button>

      {user && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              aria-label="Open user menu"
              className="mr-1 rounded-full transition-colors hover:ring-2 hover:ring-accent"
            >
              <UserAvatar
                src={user.avatar_url}
                seed={user.email}
                initials={initials ?? "U"}
                alt={displayName ?? ""}
                className="size-8"
                fallbackClassName="text-xs"
              />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuItem asChild>
              <NavLink to="/profile">
                <User className="mr-2 size-4" />
                Profile
              </NavLink>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={logout}>
              <LogOut className="mr-2 size-4" />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}
