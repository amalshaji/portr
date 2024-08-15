<script lang="ts">
  import { Badge } from "$lib/components/ui/badge";
  import type { ConnectionType } from "$lib/types";

  export let type: ConnectionType;
</script>

{#if type === "http"}
  <Badge variant="outline" class="border-blue-300 font-normal">HTTP</Badge>
{:else}
  <Badge variant="outline" class="border-green-300 font-normal">TCP</Badge>
{/if}
