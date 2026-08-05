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
    ].find(([path]) => location.pathname.endsWith(path))?.[1] ?? "Admin";

  return (
    <SidebarProvider
      style={
        {
          "--sidebar-width": "17.5rem",
          "--sidebar-width-icon": "4rem",
        } as React.CSSProperties
      }
      className="bg-background"
    >
      <Sidebar
        variant="inset"
        collapsible="icon"
        className="[&_[data-sidebar=sidebar]]:border [&_[data-sidebar=sidebar]]:border-sidebar-border/80 [&_[data-sidebar=sidebar]]:bg-sidebar/90 [&_[data-sidebar=sidebar]]:shadow-[0_12px_35px_rgba(23,33,30,0.06)]"
      >
        {sidebar}
        <SidebarRail />
      </Sidebar>
      <SidebarInset className="min-w-0 overflow-hidden border border-border/70 bg-background shadow-[0_12px_40px_rgba(23,33,30,0.05)] md:my-2 md:mr-2">
        <header className="app-chrome sticky top-0 z-30 flex h-16 shrink-0 items-center gap-3 border-b border-border/70 px-4 lg:px-6">
          <SidebarTrigger className="-ml-1 size-9 rounded-full" />
          <div className="min-w-0 flex-1">
            <p className="truncate text-[0.68rem] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
              {team ?? "Workspace"}
            </p>
            <p className="truncate text-sm font-semibold text-foreground">
              {pageName}
            </p>
          </div>
          <IssueLink />
        </header>
        <main className="min-h-0 flex-1 overflow-y-auto">
          <div className="mx-auto flex w-full max-w-[90rem] flex-col gap-5 p-4 sm:p-6 lg:gap-7 lg:p-8 xl:p-10">
            {children}
          </div>
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
