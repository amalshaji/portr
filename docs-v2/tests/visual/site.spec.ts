import { expect, test, type Page } from "@playwright/test";

async function settle(page: Page) {
  await page.waitForLoadState("networkidle");
  await page.evaluate(() => document.fonts.ready);
}

test("landing page stays aligned at desktop and mobile breakpoints", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/");
  await settle(page);

  await page.locator(".portr-map__route").evaluate((route) => {
    const svg = route as SVGSVGElement;
    svg.pauseAnimations();
    svg.setCurrentTime(2.2);
  });

  const cta = page.locator(".portr-nav__cta");
  await expect(cta).toHaveCount(1);
  await cta.hover();
  await expect(cta).not.toHaveCSS("color", "rgba(0, 0, 0, 0)");
  await expect(page).toHaveScreenshot("landing-desktop-dark.png", { fullPage: true });

  await page.setViewportSize({ width: 390, height: 844 });
  await page.reload();
  await settle(page);
  const requestLinesStayTogether = await page.locator(".portr-map__request p").evaluateAll((lines) =>
    lines.every((line) => getComputedStyle(line).whiteSpace === "nowrap"),
  );
  expect(requestLinesStayTogether).toBe(true);
  await expect(page).toHaveScreenshot("landing-mobile-dark.png", { fullPage: true });
});

test("documentation theme covers navigation, code, cards, and both color schemes", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/docs/getting-started.html");
  await settle(page);

  await expect(page.locator("#nd-sidebar .portr-docs-brand")).toBeVisible();
  await expect(page.locator(".portr-code-block")).toBeVisible();
  await expect(page.locator(".portr-code-viewport")).toHaveCount(1);
  await expect(page.locator(".portr-code-actions")).toHaveCount(1);
  await expect(page).toHaveScreenshot("docs-desktop-dark.png", { fullPage: true });

  await page.emulateMedia({ colorScheme: "light" });
  await page.reload();
  await settle(page);
  await expect(page).toHaveScreenshot("docs-desktop-light.png", { fullPage: true });

  await page.goto("/docs.html");
  await settle(page);
  await expect(page.locator(".portr-callout")).toHaveCount(1);
  await expect(page.locator(".portr-cards")).toHaveCount(3);
  await expect(page.locator(".portr-card")).toHaveCount(14);
});
