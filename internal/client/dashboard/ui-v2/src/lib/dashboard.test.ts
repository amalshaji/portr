import { describe, expect, it } from "vitest"

import { currentDashboardHost } from "@/lib/dashboard"

describe("currentDashboardHost", () => {
  it("prefers the full host when a custom port is present", () => {
    expect(
      currentDashboardHost({
        host: "localhost:8888",
        hostname: "localhost",
      })
    ).toBe("localhost:8888")
  })

  it("falls back to localhost when no location is available", () => {
    expect(currentDashboardHost(null)).toBe("localhost")
  })
})
