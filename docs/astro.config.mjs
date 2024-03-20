import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

// https://astro.build/config
export default defineConfig({
  integrations: [
    starlight({
      title: "Portr",
      customCss: ["./src/styles/custom.css", "./src/fonts/font-face.css"],
      social: {
        github: "https://github.com/amalshaji/portr",
      },
      sidebar: [
        {
          label: "Guides",
          items: [
            // Each item here is one entry in the navigation menu.
            { label: "Overview", link: "/getting-started/" },
            {
              label: "Server Setup",
              items: [
                { label: "Quickstart", link: "/server-setup/" },
                {
                  label: "Cloudflare API token",
                  link: "/server-setup/cloudflare-api-token/",
                },
                {
                  label: "Github oauth app",
                  link: "/server-setup/github-oauth-app/",
                },
              ],
            },
            { label: "Client Setup", link: "/client-setup/" },
          ],
        },
      ],
    }),
  ],
});
