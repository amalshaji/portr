<script lang="ts">
  import { Loader, TriangleAlert, X } from "lucide-svelte";
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

<div class="grid h-screen place-items-center">
  <Card.Root class="mx-auto w-96 shadow-none">
    <div class="px-2">
      {#if message}
        <div class="mt-4" id="error-message-box">
          <Alert.Root variant="destructive">
            <TriangleAlert class="h-4 w-4" />
            <Alert.Title>
              <p class="flex justify-between">
                <span>Error</span>
                <!-- svelte-ignore a11y-no-static-element-interactions -->
                <!-- svelte-ignore a11y-click-events-have-key-events -->
                <span
                  on:click={() => {
                    const element =
                      document.getElementById("error-message-box");
                    element?.remove();
                  }}><X class="h-3 hover:cursor-pointer" /></span
                >
              </p>
            </Alert.Title>
            <Alert.Description>
              {message}
            </Alert.Description>
          </Alert.Root>
        </div>
      {/if}
    </div>
    <Card.Header>
      <Card.Title class="text-2xl">
        {#if isSuperUserSignup}
          Create superuser account
        {:else}
          Login
        {/if}
      </Card.Title>
      <Card.Description>
        {#if isSuperUserSignup}
          You are signing up as a superuser.
        {:else}
          You need to be part of a team to continue.
        {/if}
      </Card.Description>
    </Card.Header>
    <Card.Content>
      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label for="email">Email</Label>
          <Input
            id="email"
            bind:value={email}
            type="email"
            placeholder="m@example.com"
            required
            class={emailError && "border-red-500"}
          />
          {#if emailError}
            <p class="text-red-500 text-xs italic">{emailError}</p>
          {/if}
        </div>
        <div class="grid gap-2">
          <div class="flex items-center">
            <Label for="password">Password</Label>
          </div>
          <Input
            id="password"
            bind:value={password}
            type="password"
            required
            class={passwordError && "border-red-500"}
          />
          {#if passwordError}
            <p class="text-red-500 text-xs italic">{passwordError}</p>
          {/if}
        </div>
        <Button type="submit" class="w-full" on:click={login}>
          {#if loginLoading}
            <Loader class="mr-2 h-4 w-4 animate-spin" />
          {/if}
          {#if isSuperUserSignup}
            Create superuser account
          {:else}
            Login
          {/if}
        </Button>

        {#if isSuperUserSignup || !githubAuthEnabled}
          <Button variant="outline" class="w-full" disabled
            >Login with GitHub</Button
          >
        {:else}
          <Button
            variant="outline"
            class="w-full"
            href={encodeURIComponent(next) !== "null"
              ? `/api/v1/auth/github?next=${encodeURIComponent(next)}`
              : `/api/v1/auth/github`}
          >
            Login with GitHub
          </Button>
        {/if}
      </div>
    </Card.Content>
  </Card.Root>
</div>
