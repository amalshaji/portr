import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/visual",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? "github" : "list",
  expect: {
    toHaveScreenshot: {
      maxDiffPixelRatio: 0.01,
    },
  },
  use: {
    baseURL: "http://127.0.0.1:3015",
    channel: "chrome",
    locale: "en-US",
    timezoneId: "UTC",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "bun run build && python3 -m http.server 3015 --directory out",
    url: "http://127.0.0.1:3015",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
