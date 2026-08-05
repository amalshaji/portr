import type { ReactNode } from "react";
import { Link, useLocation } from "react-router-dom";
import { cn } from "@/lib/utils";

interface SidebarLinkProps {
  to: string;
  children: ReactNode;
  className?: string;
}

export default function SidebarLink({
  to,
  children,
  className,
}: SidebarLinkProps) {
  const location = useLocation();
  const isActive = location.pathname === to;

  return (
    <Link
      to={to}
      aria-current={isActive ? "page" : undefined}
      className={cn(
        "flex h-10 items-center gap-2 rounded-xl px-3! text-[0.875rem] font-medium text-sidebar-foreground/75 transition-[background-color,color,transform] hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring",
        isActive &&
          "bg-sidebar-primary text-sidebar-primary-foreground shadow-[0_1px_2px_rgba(23,33,30,0.18)] hover:bg-sidebar-primary hover:text-sidebar-primary-foreground",
        className
      )}
    >
      {children}
    </Link>
  );
}
