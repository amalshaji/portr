import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import AutoSignupSettings from "./AutoSignupSettings"

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe("AutoSignupSettings", () => {
  it("does not expose editable defaults when settings fail to load", async () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ error: "database unavailable" }), {
            status: 500,
            headers: { "Content-Type": "application/json" },
          }),
        )
        .mockResolvedValueOnce(
          new Response(JSON.stringify([{ id: 1, name: "Engineering", slug: "engineering" }]), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        ),
    )

    render(<AutoSignupSettings />)

    expect(
      await screen.findByText("Auto signup settings could not be loaded."),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: "Save Settings" }),
    ).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument()
  })
})
