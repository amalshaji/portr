import { DocsLayout, type DocsLayoutProps } from "fumadocs-ui/layouts/notebook";
import type { ReactNode } from "react";
import { baseOptions } from "@/app/layout.config";
import { source } from "@/lib/source";
import { GithubInfo } from "fumadocs-ui/components/github-info";

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
        <GithubInfo owner="amalshaji" repo="portr" className="lg:-mx-2" />
      ),
    },
  ],
  nav: {
    ...baseOptions.nav,
    transparentMode: "top",
  },
};

export default function Layout({ children }: { children: ReactNode }) {
  return <DocsLayout {...docsOptions}>{children}</DocsLayout>;
}
