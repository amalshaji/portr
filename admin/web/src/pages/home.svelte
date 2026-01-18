<script lang="ts">
  import {
    Github,
    LoaderCircle,
    Lock,
    Mail,
    TriangleAlert,
    X,
    Eye,
    EyeOff,
  } from "lucide-svelte";
  import { onMount } from "svelte";

  import * as Alert from "$lib/components/ui/alert";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card/index.js";
  import { Input } from "$lib/components/ui/input/index.js";
  import { Label } from "$lib/components/ui/label/index.js";
  import { navigate } from "svelte-routing";

  let isSuperUserSignup = false,
    githubAuthEnabled = false;

  const getResponseMessage = (code: string) => {
    const codes: Map<string, string> = {
      // @ts-expect-error
      "github-oauth-error": "There was an error authenticating with GitHub.",
      "user-not-found": "You are not a member of any team.",
      "private-email": "Failed to verify github email. Please try again later.",
      "invalid-state": "Broken oauth flow, please try again later.",
    };
    return (
      // @ts-expect-error
      codes[code] ?? ""
    );
  };

  const getMessageType = (code: string) => {
    return ["invite-accepted"].includes(code) ? "success" : "error";
  };

  const urlParams = new URLSearchParams(window.location.search);
  const code = urlParams.get("code") as string;
  const next = urlParams.get("next") as string;

  let message: string = "";
  let messageType: string = "success";

  let email = "",
    emailError = "",
    password = "",
    passwordError = "";

  let loginLoading = false;
  let showPassword = false;
  let rememberMe = false;

  if (code) {
    message = getResponseMessage(code);
    messageType = getMessageType(code);
  }

  const getAuthConfig = async () => {
    const resp = await fetch("/api/v1/auth/auth-config");
    const data = await resp.json();
    isSuperUserSignup = data.is_first_signup;
    githubAuthEnabled = data.github_auth_enabled;
  };

  const login = async () => {
    emailError = "";
    passwordError = "";

    if (email === "") {
      emailError = "Email is required";
    }

    if (password === "") {
      passwordError = "Password is required";
    }

    if (emailError || passwordError) return;

    loginLoading = true;

    try {
      const res = await fetch("/api/v1/auth/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email,
          password,
          remember_me: rememberMe,
        }),
      });
      if (res.ok) {
        const { redirect_to } = await res.json();
        navigate(redirect_to);
      } else {
        const data = await res.json();
        emailError = data.email ?? "";
        passwordError = data.password ?? "";
      }
    } catch (err) {
      console.error(err);
    } finally {
      loginLoading = false;
    }
  };

  onMount(() => {
    getAuthConfig();
  });
</script>

<div
  class="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 via-blue-50/30 to-indigo-50/20 py-12 px-4 sm:px-6 lg:px-8 transition-all duration-700"
>
  <div
    class="max-w-md w-full space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700"
  >
    <!-- Logo/Brand -->
    <div class="text-center">
      <div
        class="mx-auto h-24 w-24 rounded-2xl bg-gradient-to-br from-primary/90 via-primary to-primary/80 flex items-center justify-center mb-6 shadow-xl shadow-primary/20 transition-transform duration-300 hover:scale-105"
      >
        <span class="text-4xl font-bold text-white tracking-tight">P</span>
      </div>
      <h1
        class="text-4xl font-bold bg-gradient-to-r from-gray-900 via-gray-800 to-gray-900 bg-clip-text text-transparent"
      >
        {isSuperUserSignup ? "Create Account" : "Welcome Back"}
      </h1>
      <p class="mt-3 text-base text-gray-600 font-medium">
        {isSuperUserSignup
          ? "Set up your admin account to get started"
          : "Sign in to access your admin dashboard"}
      </p>
    </div>

    {#if message}
      <div
        class="rounded-md animate-in fade-in slide-in-from-top-2 duration-300"
        id="error-message-box"
      >
        <Alert.Root
          variant="destructive"
          class="shadow-lg border-red-200 bg-red-50/90 backdrop-blur-sm"
        >
          <div class="flex gap-3">
            <TriangleAlert class="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
            <div class="flex-1">
              <Alert.Title
                class="flex justify-between items-center text-red-700 font-semibold"
              >
                <span>Authentication Error</span>
                <X
                  class="h-4 w-4 cursor-pointer opacity-70 hover:opacity-100 transition-opacity duration-200 rounded-sm hover:bg-red-100 p-0.5"
                  on:click={() => {
                    const element =
                      document.getElementById("error-message-box");
                    element?.remove();
                  }}
                />
              </Alert.Title>
              <Alert.Description class="text-sm mt-1.5 text-red-600">
                {message}
              </Alert.Description>
            </div>
          </div>
        </Alert.Root>
      </div>
    {/if}

    <Card.Root
      class="overflow-hidden rounded-2xl shadow-2xl border border-gray-200/50 backdrop-blur-md bg-white/90 transition-all duration-300 hover:shadow-3xl hover:shadow-primary/5"
    >
      <Card.Content class="p-8">
        <form class="space-y-7" on:submit|preventDefault={login}>
          <div class="space-y-2">
            <Label for="email" class="block text-sm font-semibold text-gray-800"
              >Email address</Label
            >
            <div class="relative group">
              <div
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 group-focus-within:text-primary transition-colors duration-200"
              >
                <Mail class="h-5 w-5" />
              </div>
              <Input
                id="email"
                bind:value={email}
                type="email"
                placeholder="name@company.com"
                required
                aria-describedby={emailError ? "email-error" : undefined}
                class={`pl-11 pr-4 py-3 block w-full rounded-xl border-2 transition-all duration-200 ${
                  emailError
                    ? "border-red-400 ring-2 ring-red-100 focus-visible:border-red-500 focus-visible:ring-red-200"
                    : "border-gray-200 focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/20"
                } focus-visible:outline-none bg-white/80 hover:bg-white/90`}
              />
            </div>
            {#if emailError}
              <p
                id="email-error"
                class="mt-2 text-sm text-red-600 flex items-center gap-1.5 animate-in fade-in slide-in-from-top-1 duration-200"
              >
                <TriangleAlert class="h-4 w-4 flex-shrink-0" />
                {emailError}
              </p>
            {/if}
          </div>

          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <Label
                for="password"
                class="block text-sm font-semibold text-gray-800"
                >Password</Label
              >
              {#if !isSuperUserSignup}
                <button
                  type="button"
                  class="text-sm text-primary hover:text-primary/80 font-medium transition-colors duration-200"
                >
                  Forgot password?
                </button>
              {/if}
            </div>
            <div class="relative group">
              <div
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 group-focus-within:text-primary transition-colors duration-200"
              >
                <Lock class="h-5 w-5" />
              </div>
              <Input
                id="password"
                bind:value={password}
                type={showPassword ? "text" : "password"}
                required
                aria-describedby={passwordError ? "password-error" : undefined}
                class={`pl-11 pr-12 py-3 block w-full rounded-xl border-2 transition-all duration-200 ${
                  passwordError
                    ? "border-red-400 ring-2 ring-red-100 focus-visible:border-red-500 focus-visible:ring-red-200"
                    : "border-gray-200 focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/20"
                } focus-visible:outline-none bg-white/80 hover:bg-white/90`}
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors duration-200 p-1 rounded-md hover:bg-gray-100"
                on:click={() => (showPassword = !showPassword)}
                aria-label={showPassword ? "Hide password" : "Show password"}
              >
                {#if showPassword}
                  <EyeOff class="h-4 w-4" />
                {:else}
                  <Eye class="h-4 w-4" />
                {/if}
              </button>
            </div>
            {#if passwordError}
              <p
                id="password-error"
                class="mt-2 text-sm text-red-600 flex items-center gap-1.5 animate-in fade-in slide-in-from-top-1 duration-200"
              >
                <TriangleAlert class="h-4 w-4 flex-shrink-0" />
                {passwordError}
              </p>
            {/if}
          </div>

          {#if !isSuperUserSignup}
            <div class="flex items-center space-x-2">
              <input
                id="remember-me"
                bind:checked={rememberMe}
                type="checkbox"
                class="h-4 w-4 text-primary focus:ring-primary border-gray-300 rounded transition-colors duration-200"
              />
              <Label
                for="remember-me"
                class="text-sm text-gray-700 cursor-pointer"
              >
                Remember me for 30 days
              </Label>
            </div>
          {/if}

          <div>
            <Button
              type="submit"
              disabled={loginLoading}
              class="w-full justify-center py-3.5 px-6 border border-transparent rounded-xl shadow-lg text-base font-semibold text-white bg-gradient-to-r from-primary to-primary/90 hover:from-primary/90 hover:to-primary/80 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary disabled:opacity-60 disabled:cursor-not-allowed transform transition-all duration-200 hover:scale-[1.02] active:scale-[0.98]"
            >
              {#if loginLoading}
                <LoaderCircle class="mr-2 h-5 w-5 animate-spin" />
                {isSuperUserSignup ? "Creating..." : "Signing In..."}
              {:else}
                {isSuperUserSignup ? "Create Account" : "Sign In"}
              {/if}
            </Button>
          </div>
        </form>

        {#if githubAuthEnabled && !isSuperUserSignup}
          <div class="mt-8">
            <div class="relative">
              <div class="absolute inset-0 flex items-center">
                <div class="w-full border-t border-gray-200"></div>
              </div>
              <div class="relative flex justify-center text-sm">
                <span class="px-4 bg-white/90 text-gray-500 font-medium"
                  >Or continue with</span
                >
              </div>
            </div>

            <div class="mt-6">
              <Button
                variant="outline"
                class="w-full flex justify-center py-3 px-4 border-2 border-gray-200 rounded-xl shadow-sm bg-white/80 text-sm font-semibold text-gray-700 hover:bg-white hover:border-gray-300 hover:shadow-md transition-all duration-200 transform hover:scale-[1.01]"
                href={encodeURIComponent(next) !== "null"
                  ? `/api/v1/auth/github?next=${encodeURIComponent(next)}`
                  : `/api/v1/auth/github`}
              >
                <Github class="mr-2.5 h-5 w-5" />
                Continue with GitHub
              </Button>
            </div>
          </div>
        {:else if isSuperUserSignup || !githubAuthEnabled}
          <div class="mt-8">
            <div class="relative">
              <div class="absolute inset-0 flex items-center">
                <div class="w-full border-t border-gray-200"></div>
              </div>
              <div class="relative flex justify-center text-sm">
                <span class="px-4 bg-white/90 text-gray-500 font-medium"
                  >Social authentication</span
                >
              </div>
            </div>

            <div class="mt-6">
              <Button
                variant="outline"
                class="w-full flex justify-center py-3 px-4 border-2 border-gray-200 rounded-xl shadow-sm bg-gray-50/80 text-sm font-semibold text-gray-400 opacity-60 cursor-not-allowed"
                disabled
              >
                <Github class="mr-2.5 h-5 w-5" />
                GitHub (Not configured)
              </Button>
            </div>
          </div>
        {/if}
      </Card.Content>
    </Card.Root>

    <div class="text-center mt-8">
      <p class="text-sm text-gray-500 font-medium">
        &copy; {new Date().getFullYear()} Portr. All rights reserved.
      </p>
    </div>
  </div>
</div>
