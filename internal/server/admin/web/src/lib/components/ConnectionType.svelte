<script lang="ts">
  import { Badge } from "$lib/components/ui/badge";
  import type { ConnectionType } from "$lib/types";

  export let Type: ConnectionType;
</script>

{#if Type === "http"}
  <Badge class="bg-blue-700 hover:bg-blue-600">HTTP</Badge>
{:else}
  <Badge class="bg-green-700 hover:bg-green-600">TCP</Badge>
{/if}
