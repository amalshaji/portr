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
              label: "Server",
              items: [
                { label: "Quickstart", link: "/server/" },
                {
                  label: "Cloudflare API token",
                  link: "/server/cloudflare-api-token/",
                },
                {
                  label: "Github oauth app",
                  link: "/server/github-oauth-app/",
                },
                {
                  label: "Start the server",
                  link: "/server/start-the-tunnel-server/",
                },
              ],
            },
            {
              label: "Client",
              items: [{ label: "Installation", link: "/client/installation/" }],
            },
          ],
        },
      ],
    }),
  ],
});
