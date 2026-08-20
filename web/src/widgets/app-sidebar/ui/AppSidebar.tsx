import { NavLink, Link, useLocation } from "react-router";
import { LayoutDashboard, PanelLeftClose, PanelLeftOpen, User } from "lucide-react";
import { useSidebarState } from "../lib/useSidebarState";
import { cn } from "@/shared/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip";
import { Separator } from "@/shared/ui/separator";

interface AppSidebarProps {
  className?: string;
}

interface NavItem {
  to: string;
  label: string;
  icon: typeof LayoutDashboard;
  end?: boolean;
}

const NAV_ITEMS: NavItem[] = [
  { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { to: "/profile", label: "Profile", icon: User },
];

function isNavActive(pathname: string, to: string, end?: boolean): boolean {
  return end ? pathname === to : pathname.startsWith(to);
}

export function AppSidebar({ className }: AppSidebarProps) {
  const { collapsed, toggle } = useSidebarState();
  const { pathname } = useLocation();

  const expanded = !collapsed;

  return (
    <aside
      className={cn(
        "flex flex-col border-r bg-background transition-all duration-200",
        expanded ? "w-[220px]" : "w-[60px]",
        className,
      )}
    >
      {/* Logo + pin */}
      <div className="flex h-12 items-center justify-between px-3">
        {expanded ? (
          <>
            <span className="text-base font-semibold tracking-tight">Task Tracker</span>
            <button
              onClick={toggle}
              aria-label="Collapse sidebar"
              aria-expanded={true}
              className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            >
              <PanelLeftClose className="size-4" />
            </button>
          </>
        ) : (
          <button
            onClick={toggle}
            aria-label="Expand sidebar"
            aria-expanded={false}
            className="mx-auto rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          >
            <PanelLeftOpen className="size-4" />
          </button>
        )}
      </div>

      <Separator className="my-2" />

      {/* Navigation */}
      <nav className="flex-1 space-y-0.5 px-2">
        {NAV_ITEMS.map(({ to, label, icon: Icon, end }) => {
          const active = isNavActive(pathname, to, end);

          if (!expanded) {
            return (
              <Tooltip key={to}>
                <TooltipTrigger asChild>
                  <Link
                    to={to}
                    aria-label={label}
                    aria-current={active ? "page" : undefined}
                    className={cn(
                      "flex items-center justify-center rounded-md w-full py-2 text-sm transition-colors",
                      active
                        ? "bg-accent font-medium text-foreground"
                        : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
                    )}
                  >
                    <Icon className="size-4 shrink-0" />
                  </Link>
                </TooltipTrigger>
                <TooltipContent side="right">{label}</TooltipContent>
              </Tooltip>
            );
          }

          return (
            <NavLink
              key={to}
              to={to}
              end={end}
              aria-current={active ? "page" : undefined}
              className={cn(
                "flex items-center gap-3 rounded-md px-2 py-1.5 text-sm transition-colors",
                active
                  ? "bg-accent font-medium text-foreground"
                  : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
              )}
            >
              <Icon className="size-4 shrink-0" />
              <span>{label}</span>
            </NavLink>
          );
        })}
      </nav>
    </aside>
  );
}
