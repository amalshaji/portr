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
      className={cn(
        "flex items-center gap-2 px-2 py-1.5 text-sm",
        isActive && "bg-accent text-accent-foreground",
        className
      )}
    >
      {children}
    </Link>
  );
}
