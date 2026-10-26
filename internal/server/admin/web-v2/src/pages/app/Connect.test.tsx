import { afterEach, describe, expect, it, vi } from "vitest"
import { cleanup, render, screen } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import Connect from "./Connect"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe("Connect", () => {
  it("shows the install and setup commands for the team", async () => {
    const fetchMock = vi.fn(() =>
      Promise.resolve(
        new Response(JSON.stringify({ message: "portr auth set test-token" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    )
    vi.stubGlobal("fetch", fetchMock)

    render(
      <MemoryRouter initialEntries={["/acme/connect"]}>
        <Routes>
          <Route path="/:team/connect" element={<Connect />} />
        </Routes>
      </MemoryRouter>,
    )

    expect(
      await screen.findByText("portr auth set test-token"),
    ).toBeInTheDocument()
    expect(
      screen.getByText("curl -sSf https://install.portr.dev | sh"),
    ).toBeInTheDocument()
    // The commands are already numbered, so the step rail is left out here.
    expect(
      screen.queryByRole("heading", { name: "Install the client" }),
    ).not.toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/config/setup-script",
      expect.objectContaining({ headers: { "x-team-slug": "acme" } }),
    )
  })
})
