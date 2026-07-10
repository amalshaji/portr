import { describe, expect, it } from "vitest"
import { normalizeSubdomain, validateSubdomain } from "./subdomain"

describe("subdomain validation", () => {
  it("normalizes case and whitespace", () => {
    expect(normalizeSubdomain("  My-App  ")).toBe("my-app")
  })

  it.each(["a", "api", "preview-42"])("accepts %s", (value) => {
    expect(validateSubdomain(value)).toBeNull()
  })

  it.each(["", "-api", "api-", "my_api", "two.words"])(
    "rejects %s",
    (value) => {
      expect(validateSubdomain(value)).not.toBeNull()
    },
  )
})
