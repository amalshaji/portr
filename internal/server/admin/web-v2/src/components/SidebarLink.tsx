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
        "flex h-9 items-center gap-2.5 rounded-md px-2.5! text-sm text-sidebar-foreground/75 transition-colors duration-(--portr-duration-micro) ease-portr hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring",
        isActive &&
          "bg-sidebar-accent font-medium text-sidebar-accent-foreground",
        className
      )}
    >
      {children}
    </Link>
  );
}
