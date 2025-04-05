<script lang="ts">
  import { LoaderCircle, TriangleAlert, X, Github } from "lucide-svelte";
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

<div class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
  <div class="max-w-md w-full space-y-8">
    <!-- Logo/Brand -->
    <div class="text-center">
      <div class="mx-auto h-16 w-16 rounded-full bg-primary/10 flex items-center justify-center mb-2">
        <span class="text-2xl font-bold text-primary">P</span>
      </div>
      <h2 class="text-3xl font-extrabold text-gray-900">
        {isSuperUserSignup ? "Create Account" : "Welcome Back"}
      </h2>
      <p class="mt-2 text-sm text-gray-600">
        {isSuperUserSignup
          ? "Set up your admin account to get started"
          : "Sign in to access your admin dashboard"}
      </p>
    </div>

    {#if message}
      <div class="rounded-md" id="error-message-box">
        <Alert.Root variant="destructive" class="shadow-sm">
          <div class="flex gap-2">
            <TriangleAlert class="h-5 w-5" />
            <div class="flex-1">
              <Alert.Title class="flex justify-between items-center">
                <span>Error</span>
                <X
                  class="h-4 w-4 cursor-pointer opacity-70 hover:opacity-100"
                  on:click={() => {
                    const element = document.getElementById("error-message-box");
                    element?.remove();
                  }}
                />
              </Alert.Title>
              <Alert.Description class="text-sm mt-1">
                {message}
              </Alert.Description>
            </div>
          </div>
        </Alert.Root>
      </div>
    {/if}

    <Card.Root class="overflow-hidden rounded-lg shadow-md border border-gray-200">
      <Card.Content class="p-6">
        <form class="space-y-6" on:submit|preventDefault={login}>
          <div>
            <Label for="email" class="block text-sm font-medium text-gray-700">Email address</Label>
            <div class="mt-1">
              <Input
                id="email"
                bind:value={email}
                type="email"
                placeholder="name@company.com"
                required
                class={`block w-full rounded-md ${emailError ? "border-red-500 ring-1 ring-red-500" : "border-gray-300"}`}
              />
              {#if emailError}
                <p class="mt-1 text-sm text-red-600">{emailError}</p>
              {/if}
            </div>
          </div>

          <div>
            <div class="flex items-center justify-between">
              <Label for="password" class="block text-sm font-medium text-gray-700">Password</Label>
            </div>
            <div class="mt-1">
              <Input
                id="password"
                bind:value={password}
                type="password"
                required
                class={`block w-full rounded-md ${passwordError ? "border-red-500 ring-1 ring-red-500" : "border-gray-300"}`}
              />
              {#if passwordError}
                <p class="mt-1 text-sm text-red-600">{passwordError}</p>
              {/if}
            </div>
          </div>

          <div>
            <Button
              type="submit"
              class="w-full justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
            >
              {#if loginLoading}
                <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
              {/if}
              {isSuperUserSignup ? "Create Account" : "Sign In"}
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
                <span class="px-2 bg-white text-gray-500">Or continue with</span>
              </div>
            </div>

            <div class="mt-6">
              <Button
                variant="outline"
                class="w-full flex justify-center py-2 px-4 border border-gray-300 rounded-md shadow-sm bg-white text-sm font-medium text-gray-700 hover:bg-gray-50"
                href={encodeURIComponent(next) !== "null"
                  ? `/api/v1/auth/github?next=${encodeURIComponent(next)}`
                  : `/api/v1/auth/github`}
              >
                <Github class="mr-2 h-5 w-5" />
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
                <span class="px-2 bg-white text-gray-500">Social login</span>
              </div>
            </div>

            <div class="mt-6">
              <Button
                variant="outline"
                class="w-full flex justify-center py-2 px-4 border border-gray-300 rounded-md shadow-sm bg-white text-sm font-medium text-gray-500 opacity-60 cursor-not-allowed"
                disabled
              >
                <Github class="mr-2 h-5 w-5" />
                GitHub
              </Button>
            </div>
          </div>
        {/if}
      </Card.Content>
    </Card.Root>
  </div>
</div>
