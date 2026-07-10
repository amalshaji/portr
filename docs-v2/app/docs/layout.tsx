import { DocsLayout, type DocsLayoutProps } from "fumadocs-ui/layouts/notebook";
import type { ReactNode } from "react";
import { baseOptions } from "@/app/layout.config";
import { GitHubStarsLink } from "@/components/github-stars-link";
import { source } from "@/lib/source";

const docsOptions: DocsLayoutProps = {
  ...baseOptions,
  tree: source.pageTree,
  sidebar: {
    defaultOpenLevel: 10,
    collapsible: true,
  },
  links: [
    {
      type: "custom",
      children: (
        <GitHubStarsLink className="rounded-md px-2 py-1.5 text-sm text-fd-muted-foreground transition-colors hover:bg-fd-accent hover:text-fd-foreground lg:-mx-2" />
      ),
    },
  ],
  nav: {
    ...baseOptions.nav,
    transparentMode: "top",
  },
};

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="portr-docs-shell">
      <DocsLayout {...docsOptions}>{children}</DocsLayout>
    </div>
  );
}
