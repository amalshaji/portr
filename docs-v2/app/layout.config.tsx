import type { BaseLayoutProps } from "fumadocs-ui/layouts/shared";
import { Logo } from "../components/ui/logo";

/**
 * Shared layout configurations
 *
 * you can customise layouts individually from:
 * Home Layout: app/(home)/layout.tsx
 * Docs Layout: app/docs/layout.tsx
 */
export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <>
        <Logo width={24} height={24} />
        Portr
      </>
    ),
  },
  links: [
    {
      text: "GitHub",
      url: "https://github.com/amalshaji/portr",
      active: "nested-url",
    },
  ],
};
