import { DocsLayout, type DocsLayoutProps } from "fumadocs-ui/layouts/notebook";
import type { ReactNode } from "react";
import { baseOptions } from "@/app/layout.config";
import { source } from "@/lib/source";

const docsOptions: DocsLayoutProps = {
  ...baseOptions,
  tree: source.pageTree,
  sidebar: {
    defaultOpenLevel: 10,
    collapsible: true,
  },
  nav: {
    ...baseOptions.nav,
    transparentMode: "top",
  },
};

export default function Layout({ children }: { children: ReactNode }) {
  return <DocsLayout {...docsOptions}>{children}</DocsLayout>;
}
