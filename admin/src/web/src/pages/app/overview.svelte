<script lang="ts">
  import { toast } from "svelte-sonner";
  import "svelte-highlight/styles/stackoverflow-light.css";
  import { setupScript } from "$lib/store";
  import { getContext, onMount } from "svelte";
  import { copyCodeToClipboard } from "$lib/utils";

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

  onMount(() => {
    getSetupScript();
  });
</script>

<p class="text-lg py-2">Client setup</p>

<div class="px-6 mt-2">
  <ul class="list-decimal space-y-6">
    <li>
      Setup up the portr client from the <a
        href="https://portr.dev/client-setup/"
        target="_blank"
        class="underline">docs</a
      >
    </li>
    <li class="space-y-2">
      <span>Run the following command to setup portr client auth</span>
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div
        class="border rounded-lg w-3/4 bg-zinc-100 dark:bg-zinc-800 text-sm"
        on:click={() => copyCodeToClipboard($setupScript)}
      >
        <pre class="px-4 py-3 overflow-auto">{"$ " + $setupScript}</pre>
      </div>
      <p class="mt-4 text-sm">
        Note: use <code>portr</code> instead of <code>./portr</code> if the binary
        is set in $PATH
      </p>
    </li>
    <li>
      You're ready to use the tunnel, run <code
        class="border px-2 py-1 rounded-sm">{helpCommand}</code
      >
      or checkout the
      <a
        href="https://portr.dev/client-setup/"
        target="_blank"
        class="underline">client docs</a
      > for more info.
    </li>
  </ul>
</div>
