import { fireEvent, render, screen } from "@testing-library/react"
import { beforeAll, describe, expect, it } from "vitest"

import { ReplayDialog } from "@/components/replay-dialog"
import type { RequestRecord } from "@/types"

function encodeBody(value: string) {
  return btoa(value)
}

function makeRequest(overrides: Partial<RequestRecord> = {}): RequestRecord {
  return {
    ID: "req-1",
    Subdomain: "demo",
    Host: "demo.portr.dev",
    Localport: 8010,
    Url: "/original-path",
    Method: "POST",
    Headers: {
      "Content-Type": ["application/json"],
      Accept: ["application/json"],
    },
    Body: encodeBody('{"message":"original body"}'),
    ResponseStatusCode: 200,
    ResponseHeaders: {
      "Content-Type": ["application/json"],
    },
    ResponseBody: encodeBody('{"ok":true}'),
    IsReplayed: false,
    ParentID: "",
    LoggedAt: "2026-03-29T10:00:00Z",
    ...overrides,
  }
}

beforeAll(() => {
  class ResizeObserverMock {
    observe() {}
    unobserve() {}
    disconnect() {}
  }

  globalThis.ResizeObserver = ResizeObserverMock as typeof ResizeObserver
})

describe("ReplayDialog", () => {
  it("preserves local edits when the same request refreshes during polling", () => {
    const request = makeRequest()
    const { rerender } = render(
      <ReplayDialog
        onOpenChange={() => {}}
        onReplayed={() => {}}
        open
        request={request}
      />
    )

    const pathInput = screen.getByDisplayValue("/original-path")
    const bodyTextarea = screen.getByDisplayValue('{"message":"original body"}')

    fireEvent.change(pathInput, { target: { value: "/edited-path" } })
    fireEvent.change(bodyTextarea, {
      target: { value: '{"message":"edited locally"}' },
    })

    rerender(
      <ReplayDialog
        onOpenChange={() => {}}
        onReplayed={() => {}}
        open
        request={makeRequest({
          Url: "/server-refresh",
          Body: encodeBody('{"message":"server refresh"}'),
        })}
      />
    )

    expect(screen.getByDisplayValue("/edited-path")).toBeTruthy()
    expect(
      screen.getByDisplayValue('{"message":"edited locally"}')
    ).toBeTruthy()
  })

  it("reinitializes when a different request is selected", () => {
    const { rerender } = render(
      <ReplayDialog
        onOpenChange={() => {}}
        onReplayed={() => {}}
        open
        request={makeRequest()}
      />
    )

    fireEvent.change(screen.getByDisplayValue("/original-path"), {
      target: { value: "/edited-path" },
    })

    rerender(
      <ReplayDialog
        onOpenChange={() => {}}
        onReplayed={() => {}}
        open
        request={makeRequest({
          ID: "req-2",
          Url: "/new-request",
          Body: encodeBody('{"message":"new request"}'),
        })}
      />
    )

    expect(screen.getByDisplayValue("/new-request")).toBeTruthy()
    expect(screen.getByDisplayValue('{"message":"new request"}')).toBeTruthy()
  })
})
