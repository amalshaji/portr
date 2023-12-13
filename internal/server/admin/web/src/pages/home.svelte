<script lang="ts">
  import { onMount } from "svelte";

  import {
    ExclamationTriangle,
    CheckCircled,
    LockClosed,
    GithubLogo,
  } from "radix-icons-svelte";

  import * as Alert from "$lib/components/ui/alert";
  import { Button } from "$lib/components/ui/button";
  import { settingsForSignup } from "$lib/store";

  let isSuperUserSignup = false;

  const getResponseMessage = (code: string) => {
    const codes: Map<string, string> = {
      // @ts-expect-error
      "github-oauth-error": "There was an error authenticating with GitHub.",
      "invalid-invite": "The invite is invalid/expired.",
      "requires-invite": "Signup requires an invite.",
      "private-email": "Your email is private. Please make it public.",
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

  const getSettingsForSignup = async () => {
    const resp = await fetch("/api/setting/signup");
    settingsForSignup.set(await resp.json());
  };

  onMount(() => {
    checkIfSuperuserSignup();
    getSettingsForSignup();
  });
</script>

<div class="grid h-screen place-items-center">
  <div
    class="w-full max-w-sm p-6 m-auto mx-auto bg-slate-50 shadow rounded-md border dark:bg-gray-800 py-8"
  >
    <div class="flex justify-center mx-auto py-8">
      <img class="w-auto h-7 sm:h-8" src="/static/logo.svg" alt="" />
    </div>

    <div class="flex items-center justify-between mt-4">
      <span class="border-b dark:border-gray-600 w-full"></span>
    </div>

    <Button class="w-full" href="/auth/github">
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
            You need an invite to sign up.
          {/if}
        </Alert.Description>
      </Alert.Root>
    </div>

    {#if message}
      <div class="my-4 bg-white">
        <Alert.Root>
          {#if messageType === "success"}
            <CheckCircled class="h-4 w-4" />
            <Alert.Title>Success</Alert.Title>
          {:else}
            <ExclamationTriangle class="h-4 w-4" />
            <Alert.Title>Error</Alert.Title>
          {/if}
          <Alert.Description>
            {message}
          </Alert.Description>
        </Alert.Root>
      </div>
    {/if}
  </div>
</div>
