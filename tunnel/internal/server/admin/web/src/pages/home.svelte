<script lang="ts">
  import { Github, LoaderCircle, Lock, Mail, TriangleAlert, X, Eye, EyeOff } from "lucide-svelte";
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

<div class="min-h-screen flex items-center justify-center bg-white py-12 px-4 sm:px-6 lg:px-8">
  <div class="max-w-md w-full space-y-8">
    <!-- Logo/Brand -->
    <div class="text-center">
      <div class="mx-auto h-16 w-16 border-2 border-black bg-black flex items-center justify-center mb-6">
        <span class="text-2xl font-bold text-white">P</span>
      </div>
      <h1 class="text-2xl font-bold text-black">
        {isSuperUserSignup ? "Create Account" : "Welcome Back"}
      </h1>
      <p class="mt-2 text-sm text-gray-600">
        {isSuperUserSignup
          ? "Set up your admin account to get started"
          : "Sign in to access your admin dashboard"}
      </p>
    </div>

    {#if message}
      <div class="border border-red-600 bg-red-50 p-4" id="error-message-box">
        <div class="flex">
          <div class="flex-1">
            <h3 class="text-sm font-medium text-red-800">Error</h3>
            <p class="text-sm mt-1 text-red-700">{message}</p>
          </div>
          <button
            class="text-red-400 hover:text-red-600"
            on:click={() => {
              const element = document.getElementById("error-message-box");
              element?.remove();
            }}
          >
            <X class="h-5 w-5" />
          </button>
        </div>
      </div>
    {/if}

    <div class="border-2 border-black bg-white">
      <div class="p-6">
        <form class="space-y-6" on:submit|preventDefault={login}>
          <div>
            <Label for="email" class="block text-sm font-medium text-black mb-1">Email address</Label>
            <Input
              id="email"
              bind:value={email}
              type="email"
              placeholder="name@company.com"
              required
              class={`block w-full border px-3 py-2 ${
                emailError ? "border-red-600" : "border-gray-400"
              } focus:border-black focus:outline-none focus-visible:outline-none focus-visible:ring-0 bg-white`}
              style="border-radius: 0;"
            />
            {#if emailError}
              <p class="mt-1 text-sm text-red-600">{emailError}</p>
            {/if}
          </div>

          <div>
            <div class="flex items-center justify-between mb-1">
              <Label for="password" class="block text-sm font-medium text-black">Password</Label>
              {#if !isSuperUserSignup}
                <button type="button" class="text-sm text-gray-600 hover:text-black">
                  Forgot password?
                </button>
              {/if}
            </div>
            <div class="relative">
              <Input
                id="password"
                bind:value={password}
                type={showPassword ? "text" : "password"}
                required
                class={`block w-full border px-3 py-2 pr-10 ${
                  passwordError ? "border-red-600" : "border-gray-400"
                } focus:border-black focus:outline-none focus-visible:outline-none focus-visible:ring-0 bg-white`}
                style="border-radius: 0;"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-600 hover:text-black"
                on:click={() => showPassword = !showPassword}
              >
                {#if showPassword}
                  <EyeOff class="h-4 w-4" />
                {:else}
                  <Eye class="h-4 w-4" />
                {/if}
              </button>
            </div>
            {#if passwordError}
              <p class="mt-1 text-sm text-red-600">{passwordError}</p>
            {/if}
          </div>


          <div>
            <Button
              type="submit"
              disabled={loginLoading}
              class="w-full py-2 px-4 border-2 border-black bg-black text-white hover:bg-gray-800 focus:outline-none focus:ring-0 disabled:opacity-50 disabled:cursor-not-allowed"
              style="border-radius: 0;"
            >
              {#if loginLoading}
                <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
                {isSuperUserSignup ? "Creating..." : "Signing In..."}
              {:else}
                {isSuperUserSignup ? "Create Account" : "Sign In"}
              {/if}
            </Button>
          </div>
        </form>

        {#if githubAuthEnabled && !isSuperUserSignup}
          <div class="mt-6">
            <div class="relative">
              <div class="absolute inset-0 flex items-center">
                <div class="w-full border-t border-gray-300"></div>
              </div>
              <div class="relative flex justify-center text-sm">
                <span class="px-2 bg-white text-gray-600">Or</span>
              </div>
            </div>

            <div class="mt-4">
              <Button
                variant="outline"
                class="w-full py-2 px-4 border border-gray-400 bg-white text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-0 focus:border-black"
                style="border-radius: 0;"
                href={encodeURIComponent(next) !== "null"
                  ? `/api/v1/auth/github?next=${encodeURIComponent(next)}`
                  : `/api/v1/auth/github`}
              >
                <Github class="mr-2 h-4 w-4" />
                GitHub
              </Button>
            </div>
          </div>
        {:else if isSuperUserSignup || !githubAuthEnabled}
          <div class="mt-6">
            <div class="relative">
              <div class="absolute inset-0 flex items-center">
                <div class="w-full border-t border-gray-300"></div>
              </div>
              <div class="relative flex justify-center text-sm">
                <span class="px-2 bg-white text-gray-600">Or</span>
              </div>
            </div>

            <div class="mt-4">
              <Button
                variant="outline"
                class="w-full py-2 px-4 border border-gray-300 bg-gray-100 text-gray-400 cursor-not-allowed focus:outline-none focus:ring-0"
                style="border-radius: 0;"
                disabled
              >
                <Github class="mr-2 h-4 w-4" />
                GitHub
              </Button>
            </div>
          </div>
        {/if}
      </div>
    </div>

    <div class="text-center mt-6">
      <p class="text-xs text-gray-500">
        &copy; {new Date().getFullYear()} Portr. All rights reserved.
      </p>
    </div>
  </div>
</div>
