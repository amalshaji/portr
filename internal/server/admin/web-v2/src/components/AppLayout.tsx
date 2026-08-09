import type { ReactNode } from "react";
import { useLocation, useParams } from "react-router-dom";
import {
  SidebarProvider,
  Sidebar,
  SidebarInset,
  SidebarRail,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import IssueLink from "./IssueLink";

interface AppLayoutProps {
  sidebar: ReactNode;
  children: ReactNode;
}

export default function AppLayout({ sidebar, children }: AppLayoutProps) {
  const location = useLocation();
  const { team } = useParams<{ team: string }>();
  const pageName =
    [
      ["/overview", "Overview"],
      ["/metrics", "Metrics"],
      ["/connections", "Connections"],
      ["/reserved-domains", "Reserved domains"],
      ["/users", "Users"],
      ["/my-account", "Account & settings"],
      ["/auto-signup", "GitHub auto signup"],
    ].find(([path]) => location.pathname.endsWith(path))?.[1] ?? "Admin";

  return (
    <SidebarProvider
      style={
        {
          "--sidebar-width": "17rem",
          "--sidebar-width-icon": "3.75rem",
        } as React.CSSProperties
      }
      className="bg-background"
    >
      <Sidebar
        variant="inset"
        collapsible="icon"
        className="[&_[data-sidebar=sidebar]]:border [&_[data-sidebar=sidebar]]:border-sidebar-border [&_[data-sidebar=sidebar]]:bg-sidebar"
      >
        {sidebar}
        <SidebarRail />
      </Sidebar>
      <SidebarInset className="min-w-0 overflow-hidden border border-border bg-background md:my-2 md:mr-2">
        <header className="app-chrome sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b border-border px-4 lg:px-6">
          <SidebarTrigger className="-ml-1 size-8" />
          <div className="flex min-w-0 flex-1 items-baseline gap-2">
            <span className="data truncate text-xs text-muted-foreground">
              {team ?? "workspace"}
            </span>
            <span aria-hidden="true" className="text-muted-foreground">
              /
            </span>
            <h1 className="truncate text-sm font-semibold">{pageName}</h1>
          </div>
          <IssueLink />
        </header>
        <main className="min-h-0 flex-1 overflow-y-auto">
          <div className="mx-auto flex w-full max-w-[88rem] flex-col gap-6 p-4 sm:p-6 lg:p-8">
            {children}
          </div>
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
