<script lang="ts">
  import { Badge } from "$lib/components/ui/badge";
  import type { ConnectionStatus } from "$lib/types";

  export let Status: ConnectionStatus;
</script>

{#if Status === "closed"}
  <Badge variant="outline" class="border-red-300">closed</Badge>
{:else}
  <Badge variant="outline" class="border-gray-300">active</Badge>
{/if}
