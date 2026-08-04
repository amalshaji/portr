export const getResponseMessage = (code: string) => {
  const codes: Record<string, string> = {
    'github-oauth-error': 'There was an error authenticating with GitHub.',
    'user-not-found': 'You are not a member of any team.',
    'auto-signup-disabled': 'You are not a member of any team.',
    'auto-signup-domain-denied': 'Your GitHub email is not allowed for automatic signup.',
    'auto-signup-team-missing': 'Automatic signup is not fully configured.',
    'private-email': 'Your GitHub account does not have a verified email address.',
    'email-verification-failed': 'GitHub email verification is temporarily unavailable. Please try again.',
    'invalid-state': 'Broken oauth flow, please try again later.',
  }
  return codes[code] ?? ''
}
