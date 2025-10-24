<script lang="ts">
  import * as Alert from "$lib/components/ui/alert";
  import { ExclamationTriangle } from "radix-icons-svelte";

  export let error: string | Record<string, any> = "";
</script>

<Alert.Root variant="destructive" class="border border-red-600 bg-red-50 text-red-800" style="border-radius: 0;">
  <ExclamationTriangle class="h-4 w-4 text-red-600" />
  <Alert.Title class="text-red-800">Error</Alert.Title>
  <Alert.Description class="text-red-700">
    {#if typeof error === "string"}
      {error}
    {:else}
      <ul>
        {#each Object.keys(error) as key}
          {#each Object.keys(error[key]) as message}
            <li>{error[key][message]}</li>
          {/each}
        {/each}
      </ul>
    {/if}
  </Alert.Description>
</Alert.Root>
