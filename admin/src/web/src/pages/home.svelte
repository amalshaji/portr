<script lang="ts">
  import { Github, Lock, TriangleAlert, X, TreePalm } from "lucide-svelte";
  import { onMount } from "svelte";

  import * as Alert from "$lib/components/ui/alert";
  import { Button } from "$lib/components/ui/button";

  let isSuperUserSignup = false;

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
  console.log(next);
  let message: string = "";
  let messageType: string = "success";

  if (code) {
    message = getResponseMessage(code);
    messageType = getMessageType(code);
  }

  const checkIfSuperuserSignup = async () => {
    const resp = await fetch("/api/v1/auth/is-first-signup");
    const data = await resp.json();
    isSuperUserSignup = data.is_first_signup;
  };

  onMount(() => {
    checkIfSuperuserSignup();
  });
</script>

<div class="grid h-screen place-items-center">
  <div
    class="w-full max-w-sm p-6 m-auto mx-auto rounded-md dark:bg-gray-800 py-8"
  >
    <div class="flex space-x-2 items-center justify-center">
      <TreePalm />
      <span
        class="items-center text-center my-6 text-2xl font-semibold tracking-wide"
      >
        Welcome back!
      </span>
    </div>
    <Button
      variant="outline"
      class="w-full"
      href={encodeURIComponent(next) !== "null"
        ? `/api/v1/auth/github?next=${encodeURIComponent(next)}`
        : `/api/v1/auth/github`}
    >
      <Github class="mr-2 h-4 w-4" />
      Login with GitHub
    </Button>

    <div class="my-4">
      <Alert.Root>
        <Lock class="h-4 w-4" />
        <Alert.Title>Heads up!</Alert.Title>
        <Alert.Description>
          {#if isSuperUserSignup}
            You are signing up as a superuser.
          {:else}
            You need to be part of a team to continue.
          {/if}
        </Alert.Description>
      </Alert.Root>
    </div>

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
                  const element = document.getElementById("error-message-box");
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
</div>
