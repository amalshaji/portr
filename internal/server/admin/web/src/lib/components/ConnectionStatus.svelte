<script lang="ts">
  import { Badge } from "$lib/components/ui/badge";
  import type { ConnectionStatus } from "$lib/types";

  export let Status: ConnectionStatus;
</script>

{#if Status === "closed"}
  <Badge variant="destructive">Closed</Badge>
{:else}
  <Badge variant="secondary">Active</Badge>
{/if}
