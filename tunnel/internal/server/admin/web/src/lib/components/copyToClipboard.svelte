<script lang="ts">
  import { Clipboard } from "lucide-svelte";
  import { ClipboardCheck } from "lucide-svelte";
  import { Button } from "$lib/components/ui/button";

  let copied = false;

  export let text: string;

  const ctc = () => {
    console.log("copying");
    navigator.clipboard.writeText(text);
    copied = true;
    setTimeout(() => {
      copied = false;
    }, 2000);
  };
</script>

{#if copied}
  <Button size="icon" variant="ghost" on:click={ctc}>
    <ClipboardCheck color="green" strokeWidth={1.5} class="h-4 w-4" />
  </Button>
{:else}
  <Button size="icon" variant="ghost" on:click={ctc}>
    <Clipboard strokeWidth={1.5} class="h-4 w-4" />
  </Button>
{/if}
