import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import Overview from "./Overview"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/acme/overview"]}>
      <Routes>
        <Route path="/:team/overview" element={<Overview />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe("Overview", () => {
  it("lets operators choose their install method", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ message: "portr auth set test-token" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )

    renderPage()

    expect(
      screen.getByRole("heading", { name: "Bring your first tunnel online" }),
    ).toBeInTheDocument()
    expect(screen.getByText("curl -sSf https://install.portr.dev | sh")).toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "Homebrew" }))

    expect(screen.getByText("brew install amalshaji/taps/portr")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Homebrew" })).toHaveAttribute(
      "aria-pressed",
      "true",
    )
  })

  it("offers recovery when the setup command cannot be loaded", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(null, { status: 503 })),
    )

    renderPage()

    expect(
      await screen.findByText("Setup command unavailable"),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Retry setup command" }),
    ).toBeInTheDocument()
  })
})
