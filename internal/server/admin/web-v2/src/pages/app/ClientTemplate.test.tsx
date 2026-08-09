import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import { useUserStore } from "@/lib/store"
import type { CurrentTeamUser } from "@/types"
import ClientTemplate from "./ClientTemplate"

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

afterEach(() => {
  cleanup()
  useUserStore.setState({ currentUser: null })
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

const savedTemplate = "tunnels:\n  - name: web\n    port: 3000\n"

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  })
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/acme/client-template"]}>
      <Routes>
        <Route path="/:team/client-template" element={<ClientTemplate />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe("ClientTemplate", () => {
  it("saves an edited template for the current team", async () => {
    const updated = "tunnels:\n  - name: api\n    port: 8000\n"
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ template: savedTemplate }))
      .mockResolvedValueOnce(jsonResponse({ template: updated }))
    vi.stubGlobal("fetch", fetchMock)

    renderPage()

    const editor = await screen.findByLabelText("Client template")
    expect(editor).toHaveValue(savedTemplate)

    const save = screen.getByRole("button", { name: "Save template" })
    expect(save).toBeDisabled()

    fireEvent.change(editor, { target: { value: updated } })
    expect(save).toBeEnabled()
    fireEvent.click(save)

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
    const [url, init] = fetchMock.mock.calls[1]
    expect(url).toBe("/api/v1/config/template")
    expect(init.method).toBe("PUT")
    expect(init.headers["x-team-slug"]).toBe("acme")
    expect(JSON.parse(init.body)).toEqual({ template: updated })

    await waitFor(() => expect(save).toBeDisabled())
  })

  it("shows the server validation error next to the editor", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ template: savedTemplate }))
      .mockResolvedValueOnce(
        jsonResponse(
          { error: '"server_url" is not allowed in a team template' },
          400,
        ),
      )
    vi.stubGlobal("fetch", fetchMock)

    renderPage()

    const editor = await screen.findByLabelText("Client template")
    fireEvent.change(editor, { target: { value: "server_url: evil.example.com" } })
    fireEvent.click(screen.getByRole("button", { name: "Save template" }))

    expect(
      await screen.findByText('"server_url" is not allowed in a team template'),
    ).toBeInTheDocument()
  })

  it("is read only for team members", async () => {
    useUserStore.setState({
      currentUser: { role: "member" } as CurrentTeamUser,
    })
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(jsonResponse({ template: savedTemplate })),
    )

    renderPage()

    const editor = await screen.findByLabelText("Client template")
    expect(editor).toHaveAttribute("readonly")
    expect(
      screen.getByText("Only team admins can change the client template."),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Save template" }),
    ).toBeDisabled()
  })
})
