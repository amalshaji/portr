<script lang="ts">
  import { onMount } from "svelte";

  import {
    ExclamationTriangle,
    LockClosed,
    GithubLogo,
  } from "radix-icons-svelte";

  import * as Alert from "$lib/components/ui/alert";
  import { Button } from "$lib/components/ui/button";

  let isSuperUserSignup = false;

  const getResponseMessage = (code: string) => {
    const codes: Map<string, string> = {
      // @ts-expect-error
      "github-oauth-error": "There was an error authenticating with GitHub.",
      "user-not-found": "You are not a member of any team.",
      "private-email": "Failed to verify github email. Please try again later.",
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
  let message: string = "";
  let messageType: string = "success";

  if (code) {
    message = getResponseMessage(code);
    messageType = getMessageType(code);
  }

  const checkIfSuperuserSignup = async () => {
    const resp = await fetch("/auth/github/is-superuser-signup");
    const data = await resp.json();
    isSuperUserSignup = data.isSuperUserSignup;
  };

  onMount(() => {
    checkIfSuperuserSignup();
  });
</script>

<div class="grid h-screen place-items-center">
  <div
    class="w-full max-w-sm p-6 m-auto mx-auto rounded-md dark:bg-gray-800 py-8 border"
  >
    <div class="flex justify-center mx-auto py-8 items-center">
      <img class="w-auto h-12" src="/static/logo.svg" alt="" />
    </div>

    <Button variant="secondary" class="w-full" href="/auth/github">
      <GithubLogo class="mr-2 h-4 w-4" />
      Continue with GitHub
    </Button>

    <div class="my-4">
      <Alert.Root>
        <LockClosed class="h-4 w-4" />
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
      <div class="mt-4">
        <Alert.Root>
          <ExclamationTriangle class="h-4 w-4" />
          <Alert.Title>Error</Alert.Title>
          <Alert.Description>
            {message}
          </Alert.Description>
        </Alert.Root>
      </div>
    {/if}
  </div>
</div>
