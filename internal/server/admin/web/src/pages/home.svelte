<script lang="ts">
  import { onMount } from "svelte";

  import {
    ExclamationTriangle,
    LockClosed,
    GithubLogo,
  } from "radix-icons-svelte";

  import * as Alert from "$lib/components/ui/alert";
  import { Button } from "$lib/components/ui/button";
  import { settingsForSignup } from "$lib/store";

  let isSuperUserSignup = false;

  const getErrorMessage = (errCode: string) => {
    const errorCodes: Map<string, string> = {
      // @ts-expect-error
      "github-oauth-error": "There was an error authenticating with GitHub.",
      "invalid-invite": "The invite is not valid.",
      "requires-invite": "Signup requires an invite.",
      "domain-not-allowed": `You need to use one of the following domains to sign up: ${$settingsForSignup?.RandomUserSignupAllowedDomains.split(
        ","
      ).join(", ")}`,
    };
    return (
      // @ts-expect-error
      errorCodes[errCode] ?? ""
    );
  };

  const urlParams = new URLSearchParams(window.location.search);
  const errorCode = urlParams.get("errorCode") as string;
  let error: string = "";

  if (errorCode) {
    error = getErrorMessage(errorCode);
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
          {:else if $settingsForSignup?.SignupRequiresInvite}
            You need an invite to sign up.
          {:else if $settingsForSignup?.RandomUserSignupAllowedDomains?.length ?? 0 > 0}
            Signup is restricted to the following domains:
            {$settingsForSignup?.RandomUserSignupAllowedDomains.split(",").join(
              ", "
            )}
          {/if}
        </Alert.Description>
      </Alert.Root>
    </div>

    {#if error}
      <div class="my-4 bg-white">
        <Alert.Root>
          <ExclamationTriangle class="h-4 w-4" />
          <Alert.Title>Error</Alert.Title>
          <Alert.Description>
            {error}
          </Alert.Description>
        </Alert.Root>
      </div>
    {/if}
  </div>
</div>
