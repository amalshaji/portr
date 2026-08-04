import { describe, expect, it } from 'vitest'
import { getResponseMessage } from './auth-errors'

describe('getResponseMessage', () => {
  it('distinguishes a missing verified email from a verification API failure', () => {
    expect(getResponseMessage('private-email')).toBe(
      'Your GitHub account does not have a verified email address.',
    )
    expect(getResponseMessage('email-verification-failed')).toBe(
      'GitHub email verification is temporarily unavailable. Please try again.',
    )
  })
})
