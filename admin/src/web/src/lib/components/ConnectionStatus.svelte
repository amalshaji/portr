<script lang="ts">
  import { Badge } from "$lib/components/ui/badge";
  import type { ConnectionStatus } from "$lib/types";

  export let status: ConnectionStatus;
</script>

{#if status === "closed"}
  <Badge variant="outline" class="border-red-300 font-normal">closed</Badge>
{:else}
  <Badge variant="outline" class="border-fuchsia-300 font-normal">active</Badge>
{/if}
