<script lang="ts">
  import { onMount } from "svelte";
  import { Github } from "lucide-svelte";
  import Error from "../lib/components/error.svelte";

  let isSuperUserSignup = false;
  let signupRequiresInvite = "false";
  let allowedDomains = "";

  const urlParams = new URLSearchParams(window.location.search);
  const githubAuthError = urlParams.get("github-oauth-error");

  const checkIfSuperuserSignup = async () => {
    const resp = await fetch("/auth/github/is-superuser-signup");
    const data = await resp.json();
    isSuperUserSignup = data.isSuperUserSignup;
  };

  const getSettingsForSignup = async () => {
    const resp = await fetch("/app/settings");
    const data = await resp.json();
    signupRequiresInvite = data.signup_requires_invite;
    allowedDomains = data.random_user_signup_allowed_domains;
  };

  onMount(() => {
    checkIfSuperuserSignup();
    getSettingsForSignup();
  });
</script>

<div class="grid h-screen place-items-center">
  <div
    class="w-full max-w-sm p-6 m-auto mx-auto bg-gray-50 shadow rounded-md border dark:bg-gray-800 py-8"
  >
    <div class="flex justify-center mx-auto py-8">
      <img class="w-auto h-7 sm:h-8" src="/static/logo.svg" alt="" />
    </div>

    <div class="flex items-center justify-between mt-4">
      <span class="border-b dark:border-gray-600 w-full"></span>
    </div>

    <div class="flex items-center mt-6 -mx-2">
      <a
        href="/auth/github"
        type="button"
        class="flex items-center justify-center w-full px-6 py-2 mx-2 text-sm font-medium text-white transition-colors duration-300 transform bg-gray-900 rounded-md hover:bg-gray-800 focus:outline-none"
      >
        <Github></Github>

        <span class="hidden mx-2 sm:inline">Continue with GitHub</span>
      </a>
    </div>

    <!-- banner -->
    <div
      class="flex my-8 w-full max-w-sm overflow-hidden bg-white rounded-lg border dark:bg-gray-800"
    >
      <div class="flex items-center justify-center w-12 bg-slate-700">
        <svg
          class="w-6 h-6 text-white fill-current"
          viewBox="0 0 40 40"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            d="M20 3.33331C10.8 3.33331 3.33337 10.8 3.33337 20C3.33337 29.2 10.8 36.6666 20 36.6666C29.2 36.6666 36.6667 29.2 36.6667 20C36.6667 10.8 29.2 3.33331 20 3.33331ZM21.6667 28.3333H18.3334V25H21.6667V28.3333ZM21.6667 21.6666H18.3334V11.6666H21.6667V21.6666Z"
          />
        </svg>
      </div>

      <div class="px-4 py-2 -mx-3">
        <div class="mx-3">
          <p class="text-sm text-gray-600 dark:text-gray-200">
            {#if isSuperUserSignup}
              You are signing up as a superuser.
            {:else if signupRequiresInvite == "true"}
              You need an invite to sign up.
            {:else if allowedDomains.length > 0}
              Signup is restricted to the following domains:
              {allowedDomains.split(",").join(", ")}
            {/if}
          </p>
        </div>
      </div>
    </div>

    {#if githubAuthError}
      <Error error={githubAuthError} />
    {/if}
  </div>
</div>
