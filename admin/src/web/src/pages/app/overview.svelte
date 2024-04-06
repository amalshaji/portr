<script lang="ts">
  import { setupScript } from "$lib/store";
  import { copyCodeToClipboard } from "$lib/utils";
  import { getContext, onMount } from "svelte";
  import Highlight from "svelte-highlight";
  import bash from "svelte-highlight/languages/bash";
  import "svelte-highlight/styles/atom-one-light.css";

  const helpCommand = "portr -h";

  let team = getContext("team") as string;

  const getSetupScript = async () => {
    const res = await fetch("/api/v1/config/setup-script", {
      headers: {
        "x-team-slug": team,
      },
    });
    setupScript.set((await res.json())["message"]);
  };

  const installCommand = `
  brew install amalshaji/taps/portr
  `.trim();

  onMount(() => {
    getSetupScript();
  });
</script>

<div class="p-6">
  <p class="text-lg py-2 font-semibold leading-none tracking-tight">
    Client setup
  </p>

  <div class="px-6 mt-2">
    <ul class="list-decimal space-y-6">
      <li class="space-y-2">
        <span>Install the portr client using homebrew</span>
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <span on:click={() => copyCodeToClipboard(installCommand)}>
          <Highlight
            language={bash}
            code={installCommand}
            class="border w-3/4 text-sm my-2"
          />
        </span>
        <p class="mt-4 text-sm">
          Or download the binary from the <a
            href="https://github.com/amalshaji/portr/releases"
            target="_blank"
            class="underline">github releases</a
          >
        </p>
      </li>
      <li class="space-y-2">
        <span>Run the following command to setup portr client auth</span>
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <span on:click={() => copyCodeToClipboard($setupScript)}>
          <Highlight
            language={bash}
            code={$setupScript}
            class="border w-3/4 text-sm my-2"
          />
        </span>
        <p class="mt-4 text-sm">
          Note: use <code>./portr</code> instead of <code>portr</code> if the
          binary is in the same folder and not set in <code>$PATH</code>
        </p>
      </li>
      <li>
        You're ready to use the tunnel, run <code
          class="border px-1 py-0.5 rounded-lg">{helpCommand}</code
        >
        or checkout the
        <a
          href="https://portr.dev/client/installation/"
          target="_blank"
          class="underline">client docs</a
        > for more info.
      </li>
    </ul>
  </div>
</div>
