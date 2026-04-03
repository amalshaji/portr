import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { PayloadViewer } from "@/components/payload-viewer"
import type { RequestRecord } from "@/types"

vi.mock("@/components/theme-provider", () => ({
  useTheme: () => ({
    theme: "light",
  }),
}))

function encodeBody(value: string) {
  return btoa(value)
}

function makeRequest(overrides: Partial<RequestRecord> = {}): RequestRecord {
  return {
    ID: "req-html",
    Subdomain: "demo",
    Host: "demo.portr.dev",
    Localport: 8010,
    Url: "/page",
    Method: "GET",
    Headers: {},
    Body: "",
    ResponseStatusCode: 200,
    ResponseHeaders: {
      "Content-Type": ["text/html; charset=utf-8"],
      "Content-Length": ["21"],
    },
    ResponseBody: encodeBody("<html></html>"),
    IsReplayed: false,
    ParentID: "",
    LoggedAt: "2026-04-04T00:00:00Z",
    ...overrides,
  }
}

describe("PayloadViewer", () => {
  it("renders HTML previews in a scrollable iframe viewport", () => {
    render(<PayloadViewer request={makeRequest()} type="response" />)

    const iframe = screen.getByTitle("Response body")
    expect(iframe.getAttribute("scrolling")).toBe("yes")
    expect(iframe.parentElement?.className.includes("h-full")).toBe(true)
  })
})
