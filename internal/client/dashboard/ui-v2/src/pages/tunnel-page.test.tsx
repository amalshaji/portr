import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { TunnelPage } from "@/pages/tunnel-page"
import { getRequests, getWebSocketSessions } from "@/lib/api"

vi.mock("@/components/theme-toggle", () => ({
  ThemeToggle: () => <div data-testid="theme-toggle" />,
}))

vi.mock("@/lib/api", () => ({
  getRequests: vi.fn(),
  getWebSocketSession: vi.fn(),
  getWebSocketSessions: vi.fn(),
  replayRequest: vi.fn(),
}))

describe("TunnelPage", () => {
  beforeEach(() => {
    vi.mocked(getRequests).mockReset()
    vi.mocked(getWebSocketSessions).mockReset()
  })

  it("shows a persistent server banner until polling recovers", async () => {
    vi.mocked(getRequests)
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockResolvedValue({ requests: [] })
    vi.mocked(getWebSocketSessions)
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockResolvedValue({ sessions: [] })

    render(
      <MemoryRouter initialEntries={["/demo-8010"]}>
        <Routes>
          <Route element={<TunnelPage />} path="/:id" />
        </Routes>
      </MemoryRouter>
    )

    expect(
      await screen.findByText("Unable to reach dashboard server, is it running?")
    ).toBeTruthy()

    fireEvent.click(screen.getByRole("button", { name: /refresh/i }))

    await waitFor(() => {
      expect(vi.mocked(getRequests)).toHaveBeenCalledTimes(2)
      expect(vi.mocked(getWebSocketSessions)).toHaveBeenCalledTimes(2)
    })

    await waitFor(() => {
      expect(
        screen.queryByText("Unable to reach dashboard server, is it running?")
      ).toBeNull()
    })

    expect(screen.getByText("No request traces")).toBeTruthy()
  })
})
