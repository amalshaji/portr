import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Github, LoaderCircle, Eye, EyeOff, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { AuthConfig } from '@/types'

export default function HomePage() {
  const [isSuperUserSignup, setIsSuperUserSignup] = useState(false)
  const [githubAuthEnabled, setGithubAuthEnabled] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [emailError, setEmailError] = useState('')
  const [passwordError, setPasswordError] = useState('')
  const [loginLoading, setLoginLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [rememberMe] = useState(false)
  const [message, setMessage] = useState('')
  
  const navigate = useNavigate()

  const getResponseMessage = (code: string) => {
    const codes: Record<string, string> = {
      'github-oauth-error': 'There was an error authenticating with GitHub.',
      'user-not-found': 'You are not a member of any team.',
      'private-email': 'Failed to verify github email. Please try again later.',
      'invalid-state': 'Broken oauth flow, please try again later.',
    }
    return codes[code] ?? ''
  }

  // const getMessageType = (code: string) => {
  //   return ['invite-accepted'].includes(code) ? 'success' : 'error'
  // }

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
    } finally {
      setLoginLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-white py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8">
        {/* Logo/Brand */}
        <div className="text-center">
          <img src="/static/logo.svg" alt="Portr Logo" className="mx-auto h-16 w-16 mb-6" />
          <h1 className="text-2xl font-bold text-black">
            {isSuperUserSignup ? 'Create Account' : 'Welcome Back'}
          </h1>
          <p className="mt-2 text-sm text-gray-600">
            {isSuperUserSignup
              ? 'Set up your admin account to get started'
              : 'Sign in to access your admin dashboard'}
          </p>
        </div>

        {message && (
          <div className="border border-red-600 bg-red-50 p-4" id="error-message-box">
            <div className="flex">
              <div className="flex-1">
                <h3 className="text-sm font-medium text-red-800">Error</h3>
                <p className="text-sm mt-1 text-red-700">{message}</p>
              </div>
              <button
                className="text-red-400 hover:text-red-600"
                onClick={() => setMessage('')}
              >
                <X className="h-5 w-5" />
              </button>
            </div>
          </div>
        )}

        <Card className="border border-gray-200 bg-white">
          <div className="p-6">
            <form className="space-y-6" onSubmit={handleLogin}>
              <div>
                <Label htmlFor="email" className="block text-sm font-medium text-black mb-1">
                  Email address
                </Label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="name@company.com"
                  required
                  className={emailError ? 'border-destructive' : ''}
                />
                {emailError && (
                  <p className="mt-1 text-sm text-red-600">{emailError}</p>
                )}
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <Label htmlFor="password" className="block text-sm font-medium text-black">
                    Password
                  </Label>
                  {!isSuperUserSignup && (
                    <button type="button" className="text-sm text-gray-600 hover:text-black">
                      Forgot password?
                    </button>
                  )}
                </div>
                <div className="relative">
                  <Input
                    id="password"
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    className={passwordError ? 'border-destructive pr-10' : 'pr-10'}
                  />
                  <button
                    type="button"
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-600 hover:text-black"
                    onClick={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </button>
                </div>
                {passwordError && (
                  <p className="mt-1 text-sm text-red-600">{passwordError}</p>
                )}
              </div>

              <div>
                <Button
                  type="submit"
                  disabled={loginLoading}
                  className="w-full"
                >
                  {loginLoading ? (
                    <>
                      <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />
                      {isSuperUserSignup ? 'Creating...' : 'Signing In...'}
                    </>
                  ) : (
                    isSuperUserSignup ? 'Create Account' : 'Sign In'
                  )}
                </Button>
              </div>
            </form>

            {githubAuthEnabled && !isSuperUserSignup ? (
              <div className="mt-6">
                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <div className="w-full border-t border-gray-300"></div>
                  </div>
                  <div className="relative flex justify-center text-sm">
                    <span className="px-2 bg-white text-gray-600">Or</span>
                  </div>
                </div>

                <div className="mt-4">
                  <Button
                    variant="outline"
                    asChild
                    className="w-full"
                  >
                    <a
                      href={`/api/v1/auth/github${new URLSearchParams(window.location.search).get('next') ? `?next=${encodeURIComponent(new URLSearchParams(window.location.search).get('next')!)}` : ''}`}
                    >
                      <Github className="mr-2 h-4 w-4" />
                      GitHub
                    </a>
                  </Button>
                </div>
              </div>
            ) : (
              <div className="mt-6">
                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <div className="w-full border-t border-gray-300"></div>
                  </div>
                  <div className="relative flex justify-center text-sm">
                    <span className="px-2 bg-white text-gray-600">Or</span>
                  </div>
                </div>

                <div className="mt-4">
                  <Button
                    variant="outline"
                    disabled
                    className="w-full"
                  >
                    <Github className="mr-2 h-4 w-4" />
                    GitHub
                  </Button>
                </div>
              </div>
            )}
          </div>
        </Card>

        <div className="text-center mt-6">
          <p className="text-xs text-gray-500">
            &copy; {new Date().getFullYear()} Portr. All rights reserved.
          </p>
        </div>
      </div>
    </div>
  )
}
