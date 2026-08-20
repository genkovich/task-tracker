import { NavLink } from "react-router";
import { LayoutDashboard, User } from "lucide-react";
import { cn } from "@/shared/lib/utils";

interface BottomTabsProps {
  className?: string;
}

const TABS = [
  { to: "/dashboard", label: "Home", icon: LayoutDashboard },
  { to: "/profile", label: "Profile", icon: User },
];

export function BottomTabs({ className }: BottomTabsProps) {
  return (
    <div
      className={cn(
        "fixed inset-x-0 bottom-0 z-50 border-t bg-background safe-area-bottom",
        className,
      )}
    >
      <nav className="flex h-14 items-center justify-around px-2">
        {TABS.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              cn(
                "flex flex-col items-center gap-0.5 px-2 py-1 text-[10px] transition-colors",
                isActive ? "text-primary" : "text-muted-foreground",
              )
            }
          >
            <Icon className="size-5" />
            <span>{label}</span>
          </NavLink>
        ))}
      </nav>
    </div>
  );
}
