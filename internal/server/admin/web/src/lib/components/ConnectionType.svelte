<script lang="ts">
  import { Badge } from "$lib/components/ui/badge";
  import type { ConnectionType } from "$lib/types";

  export let Type: ConnectionType;
</script>

{#if Type === "http"}
  <Badge variant="outline" class="border-blue-300">HTTP</Badge>
{:else}
  <Badge variant="outline" class="border-green-300">TCP</Badge>
{/if}
