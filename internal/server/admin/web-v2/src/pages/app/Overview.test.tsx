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

/** Overview calls three endpoints; route by URL so each test can describe the
 *  server state it cares about. */
function stubFetch({
  setupScript = "portr auth set test-token",
  setupStatus = 200,
  connections = [] as unknown[],
  count = 0,
}) {
  vi.stubGlobal(
    "fetch",
    vi.fn((input: RequestInfo | URL) => {
      const url = String(input)

      if (url.includes("/config/setup-script")) {
        return Promise.resolve(
          setupStatus === 200
            ? new Response(JSON.stringify({ message: setupScript }), {
                status: 200,
                headers: { "Content-Type": "application/json" },
              })
            : new Response(null, { status: setupStatus }),
        )
      }

      if (url.includes("/connections/")) {
        return Promise.resolve(
          new Response(JSON.stringify({ data: connections, count }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        )
      }

      return Promise.resolve(
        new Response(
          JSON.stringify({
            team_stats: { active_connections: 1, team_members: 4 },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      )
    }),
  )
}

const connection = {
  id: 1,
  type: "http",
  port: 8000,
  subdomain: "api-dev",
  created_at: new Date().toISOString(),
  started_at: new Date().toISOString(),
  closed_at: null,
  status: "active",
  created_by: { user: { email: "ada@example.com" } },
}

describe("Overview", () => {
  it("lets operators choose their install method on first run", async () => {
    stubFetch({ count: 0 })

    renderPage()

    expect(
      await screen.findByRole("heading", { name: "Bring this workspace online" }),
    ).toBeInTheDocument()
    expect(
      screen.getByText("curl -sSf https://install.portr.dev | sh"),
    ).toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "Homebrew" }))

    expect(
      screen.getByText("brew install amalshaji/taps/portr"),
    ).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Homebrew" })).toHaveAttribute(
      "aria-pressed",
      "true",
    )
  })

  it("offers recovery when the setup command cannot be loaded", async () => {
    stubFetch({ setupStatus: 503, count: 0 })

    renderPage()

    expect(
      await screen.findByText("Setup command unavailable"),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Retry setup command" }),
    ).toBeInTheDocument()
  })

  it("shows recent routes once the team has connected", async () => {
    stubFetch({ connections: [connection], count: 12 })

    renderPage()

    expect(
      await screen.findByRole("heading", { name: "Recent routes" }),
    ).toBeInTheDocument()
    expect(screen.getByText("api-dev")).toBeInTheDocument()
    expect(screen.getByText(":8000")).toBeInTheDocument()

    // The setup path stays reachable, just collapsed.
    expect(
      screen.queryByText("curl -sSf https://install.portr.dev | sh"),
    ).not.toBeInTheDocument()
    fireEvent.click(
      screen.getByRole("button", { name: /Connect another machine/ }),
    )
    expect(
      screen.getByText("curl -sSf https://install.portr.dev | sh"),
    ).toBeInTheDocument()
  })
})
