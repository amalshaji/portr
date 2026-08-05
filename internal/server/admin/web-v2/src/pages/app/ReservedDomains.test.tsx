import { afterEach, describe, expect, it, vi } from "vitest"
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import ReservedDomains from "./ReservedDomains"

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/acme/reserved-domains"]}>
      <Routes>
        <Route path="/:team/reserved-domains" element={<ReservedDomains />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe("ReservedDomains", () => {
  it("normalizes and reserves one subdomain", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: [],
            count: 0,
            limit: 3,
            base_domain: "example.test",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            subdomain: "my-app",
            created_at: "2026-07-10T12:00:00Z",
            claim_status: "idle",
          }),
          { status: 201, headers: { "Content-Type": "application/json" } },
        ),
      )
    vi.stubGlobal("fetch", fetchMock)

    renderPage()

    expect(await screen.findByText(".example.test")).toBeInTheDocument()
    const input = screen.getByLabelText("Subdomain")
    fireEvent.change(input, { target: { value: "My-App" } })
    expect(input).toHaveValue("my-app")

    fireEvent.click(screen.getByRole("button", { name: "Reserve" }))

    expect(await screen.findByText("my-app.example.test")).toBeInTheDocument()
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
    expect(fetchMock).toHaveBeenLastCalledWith(
      "/api/v1/reserved-subdomains/",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ subdomain: "my-app" }),
      }),
    )
  })

  it("shows DNS-safe validation on blur", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: [],
            count: 0,
            limit: 3,
            base_domain: "example.test",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    )

    renderPage()
    const input = await screen.findByLabelText("Subdomain")
    fireEvent.change(input, { target: { value: "my_app" } })
    fireEvent.blur(input)

    expect(
      screen.getByText("Use 1–63 letters, numbers, or internal hyphens"),
    ).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Reserve" })).toBeDisabled()
  })

  it("rejects a malformed reservation list", async () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            data: [{ subdomain: "missing-fields" }],
            count: 1,
            limit: 3,
            base_domain: "example.test",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    )

    renderPage()

    expect(
      await screen.findByText("Reserved subdomains could not be loaded"),
    ).toBeInTheDocument()
  })

  it("explains active release behavior before deleting", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            data: [
              {
                subdomain: "live-api",
                created_at: "2026-07-10T12:00:00Z",
                claim_status: "active",
              },
            ],
            count: 1,
            limit: 3,
            base_domain: "example.test",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
    vi.stubGlobal("fetch", fetchMock)

    renderPage()
    expect(await screen.findByText("live-api.example.test")).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "Release" }))

    const dialog = screen.getByRole("alertdialog")
    expect(
      within(dialog).getByText(
        "live-api.example.test will stay unavailable until its current tunnel stops, then other users can claim it.",
      ),
    ).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole("button", { name: "Release" }))

    await waitFor(() =>
      expect(screen.queryByText("live-api.example.test")).not.toBeInTheDocument(),
    )
    expect(fetchMock).toHaveBeenLastCalledWith(
      "/api/v1/reserved-subdomains/live-api",
      expect.objectContaining({ method: "DELETE" }),
    )
  })
})
