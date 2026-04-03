import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { HomePage } from "@/pages/home-page"
import { getTunnels } from "@/lib/api"

vi.mock("@/components/theme-toggle", () => ({
  ThemeToggle: () => <div data-testid="theme-toggle" />,
}))

vi.mock("@/lib/api", () => ({
  deleteTunnelLogs: vi.fn(),
  getTunnels: vi.fn(),
}))

function makeTunnel() {
  return {
    Subdomain: "demo",
    Localport: 8010,
    last_request_id: "req-1",
    last_method: "GET",
    last_url: "/health",
    last_status: 200,
    last_activity_at: "2026-04-04T00:00:00Z",
    last_activity_kind: "http" as const,
    http_request_count: 3,
    websocket_session_count: 1,
    active_websocket_count: 0,
  }
}

describe("HomePage", () => {
  beforeEach(() => {
    vi.mocked(getTunnels).mockReset()
  })

  it("keeps a server banner visible until a later poll succeeds", async () => {
    vi.mocked(getTunnels)
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockResolvedValue({ tunnels: [makeTunnel()] })

    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>
    )

    expect(
      await screen.findByText("Unable to reach dashboard server, is it running?")
    ).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: /refresh/i }))

    await waitFor(() => {
      expect(vi.mocked(getTunnels)).toHaveBeenCalledTimes(2)
    })

    await waitFor(() => {
      expect(
        screen.queryByText("Unable to reach dashboard server, is it running?")
      ).toBeNull()
    })

    expect(screen.getAllByText("demo").length).toBeGreaterThan(0)
  })
})
