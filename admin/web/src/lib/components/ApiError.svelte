<script lang="ts">
  import * as Alert from "$lib/components/ui/alert";
  import { ExclamationTriangle } from "radix-icons-svelte";

  export let error: string | Record<string, any> = "";
</script>

<Alert.Root>
  <ExclamationTriangle class="h-4 w-4" />
  <Alert.Title>Error</Alert.Title>
  <Alert.Description>
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
