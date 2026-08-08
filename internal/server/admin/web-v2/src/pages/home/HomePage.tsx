import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Github, LoaderCircle, Eye, EyeOff, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import RouteLine from '@/components/RouteLine'
import ThemeToggle from '@/components/ThemeToggle'
import type { AuthConfig } from '@/types'
import { getResponseMessage } from './auth-errors'

export default function HomePage() {
  const [isSuperUserSignup, setIsSuperUserSignup] = useState(false)
  const [githubAuthEnabled, setGithubAuthEnabled] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [emailError, setEmailError] = useState('')
  const [passwordError, setPasswordError] = useState('')
  const [loginLoading, setLoginLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [rememberMe, setRememberMe] = useState(false)
  const [message, setMessage] = useState('')

  const navigate = useNavigate()

  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search)
    const code = urlParams.get('code')
    if (code) {
      setMessage(getResponseMessage(code))
    }

    const getAuthConfig = async () => {
      try {
        const resp = await fetch('/api/v1/auth/auth-config')
        const data: AuthConfig = await resp.json()
        setIsSuperUserSignup(data.is_first_signup)
        setGithubAuthEnabled(data.github_auth_enabled)
      } catch (err) {
        console.error('Failed to get auth config:', err)
      }
    }

    getAuthConfig()
  }, [])

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setEmailError('')
    setPasswordError('')

    if (email === '') {
      setEmailError('Email is required')
    }
    if (password === '') {
      setPasswordError('Password is required')
    }
    if (!email || !password) return

    setLoginLoading(true)

    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          email,
          password,
          remember_me: rememberMe,
        }),
      })

      if (res.ok) {
        const { redirect_to } = await res.json()
        navigate(redirect_to)
      } else {
        const data = await res.json()
        setEmailError(data.email ?? '')
        setPasswordError(data.password ?? '')
      }
    } catch (err) {
      console.error(err)
      setPasswordError('Could not reach the server. Check that it is running.')
    } finally {
      setLoginLoading(false)
    }
  }

  const nextParam = new URLSearchParams(window.location.search).get('next')
  const githubHref = `/api/v1/auth/github${
    nextParam ? `?next=${encodeURIComponent(nextParam)}` : ''
  }`

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[1.05fr_1fr]">
      {/* Always night: this side is the public internet, the form side is the
          console. Scoping `dark` here resolves every token inside to the night
          palette regardless of the active theme. */}
      <aside className="dark relative flex flex-col justify-between overflow-hidden border-b border-border bg-[var(--portr-night-deep)] px-6 py-6 text-foreground lg:border-r lg:border-b-0 lg:px-12 lg:py-10">
        <a
          href="https://portr.dev"
          target="_blank"
          rel="noopener noreferrer"
          className="flex w-fit items-center gap-3 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <img
            src={`${import.meta.env.BASE_URL}portr-mark.svg`}
            alt=""
            className="size-8 lg:size-9"
          />
          <span className="leading-tight">
            <span className="block text-sm font-semibold tracking-tight">
              Portr
            </span>
            <span className="block text-xs text-muted-foreground">
              Admin console
            </span>
          </span>
          <span className="sr-only">Portr home</span>
        </a>

        <div className="hidden max-w-lg lg:block">
          <h1 className="text-4xl leading-[1.08] font-semibold">
            One name, pointed at one port.
          </h1>
          <p className="mt-4 text-sm leading-6 text-muted-foreground">
            Portr binds a public address to a service running on your machine.
            This console is where you decide which names exist and who gets to
            use them.
          </p>

          <div className="mt-10 rounded-lg border border-border bg-card p-5">
            <RouteLine
              name="api-dev.portr.dev"
              port={3000}
              protocol="http"
              state="live"
              size="lg"
              animate
            />
          </div>
        </div>

        <p className="hidden text-xs text-muted-foreground lg:block">
          Self-hosted · {new Date().getFullYear()}
        </p>
      </aside>

      <main className="flex items-center justify-center px-4 py-10 sm:px-6 lg:px-10">
        <div className="w-full max-w-sm">
          <div className="mb-8 flex items-start justify-between gap-4">
            <div>
              <p className="eyebrow">
                {isSuperUserSignup ? 'First run' : 'Sign in'}
              </p>
              <h2 className="mt-1.5 text-2xl font-semibold">
                {isSuperUserSignup ? 'Create the admin account' : 'Welcome back'}
              </h2>
              <p className="mt-2 text-sm text-muted-foreground">
                {isSuperUserSignup
                  ? 'This server has no admin yet. The account you create here becomes the superuser.'
                  : 'Sign in to manage tunnels, names, and team access.'}
              </p>
            </div>
            <ThemeToggle className="mt-0.5 shrink-0" />
          </div>

          {message && (
            <div
              id="error-message-box"
              role="alert"
              className="mb-6 flex items-start gap-3 rounded-md border border-destructive/30 bg-destructive/10 p-3"
            >
              <p className="flex-1 text-sm text-foreground">{message}</p>
              <button
                type="button"
                aria-label="Dismiss"
                className="text-muted-foreground transition-colors hover:text-foreground"
                onClick={() => setMessage('')}
              >
                <X className="size-4" />
              </button>
            </div>
          )}

          <form className="space-y-5" onSubmit={handleLogin} noValidate>
            <div className="space-y-2">
              <Label htmlFor="email">Email address</Label>
              <Input
                id="email"
                type="email"
                name="email"
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="name@company.com"
                aria-invalid={emailError ? true : undefined}
                aria-describedby={emailError ? 'email-error' : undefined}
                required
              />
              {emailError && (
                <p id="email-error" role="alert" className="text-sm text-destructive">
                  {emailError}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <div className="relative">
                <Input
                  id="password"
                  name="password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete={
                    isSuperUserSignup ? 'new-password' : 'current-password'
                  }
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="pr-10"
                  aria-invalid={passwordError ? true : undefined}
                  aria-describedby={passwordError ? 'password-error' : undefined}
                  required
                />
                <button
                  type="button"
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                  aria-pressed={showPassword}
                  className="absolute top-1/2 right-3 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                  onClick={() => setShowPassword(!showPassword)}
                >
                  {showPassword ? (
                    <EyeOff className="size-4" />
                  ) : (
                    <Eye className="size-4" />
                  )}
                </button>
              </div>
              {passwordError && (
                <p
                  id="password-error"
                  role="alert"
                  className="text-sm text-destructive"
                >
                  {passwordError}
                </p>
              )}
            </div>

            {!isSuperUserSignup && (
              <div className="flex items-center gap-2">
                <Checkbox
                  id="remember-me"
                  checked={rememberMe}
                  onCheckedChange={(checked) => setRememberMe(checked === true)}
                />
                <Label htmlFor="remember-me" className="font-normal">
                  Keep me signed in
                </Label>
              </div>
            )}

            <Button type="submit" disabled={loginLoading} className="w-full">
              {loginLoading && <LoaderCircle className="size-4 animate-spin" />}
              {loginLoading
                ? isSuperUserSignup
                  ? 'Creating account'
                  : 'Signing in'
                : isSuperUserSignup
                  ? 'Create account'
                  : 'Sign in'}
            </Button>
          </form>

          {!isSuperUserSignup && (
            <>
              <div className="my-6 flex items-center gap-3">
                <span className="h-px flex-1 bg-border" />
                <span className="eyebrow">or</span>
                <span className="h-px flex-1 bg-border" />
              </div>

              {githubAuthEnabled ? (
                <Button variant="outline" asChild className="w-full">
                  <a href={githubHref}>
                    <Github className="size-4" />
                    Continue with GitHub
                  </a>
                </Button>
              ) : (
                <>
                  <Button
                    variant="outline"
                    disabled
                    className="w-full"
                    aria-describedby="github-disabled-hint"
                  >
                    <Github className="size-4" />
                    Continue with GitHub
                  </Button>
                  <p
                    id="github-disabled-hint"
                    className="mt-2 text-center text-xs text-muted-foreground"
                  >
                    GitHub sign-in is turned off for this server.
                  </p>
                </>
              )}

              {/* There is no self-serve reset: the only reset endpoint is
                  POST /team/users/:id/reset-password, behind RequireAdmin. */}
              <p className="mt-6 text-center text-xs text-muted-foreground">
                Lost your password? A team admin can reset it for you.
              </p>
            </>
          )}
        </div>
      </main>
    </div>
  )
}
