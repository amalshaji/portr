const FALLBACK = 'Sign in failed. Please try again.'

// Every code here is emitted by internal/server/admin/api/auth/handlers.go.
// An unmapped code used to render an empty error box, so the user saw a blank
// panel instead of a reason — anything unrecognised now falls back.
export const getResponseMessage = (code: string) => {
  const codes: Record<string, string> = {
    'github-oauth-error': 'There was an error authenticating with GitHub.',
    'github-disabled': 'GitHub sign-in is turned off for this server.',
    'user-not-found': 'You are not a member of any team.',
    'auto-signup-disabled': 'You are not a member of any team.',
    'auto-signup-domain-denied': 'Your GitHub email is not allowed for automatic signup.',
    'auto-signup-team-missing': 'Automatic signup is not fully configured.',
    'private-email': 'Your GitHub account does not have a verified email address.',
    'email-verification-failed': 'GitHub email verification is temporarily unavailable. Please try again.',
    'invalid-state': 'Broken oauth flow, please try again later.',
    'invalid-session': 'Your sign-in session expired. Start again.',
    'no-code': 'GitHub did not return an authorization code. Start again.',
    'token-exchange-failed': 'GitHub rejected the sign-in request. Try again.',
    'user-fetch-failed': 'Portr could not read your GitHub profile. Try again.',
    'database-error': 'The server could not complete sign-in. Check the server logs.',
    'session-creation-failed': 'The server could not start your session. Try again.',
  }
  return codes[code] ?? FALLBACK
}
