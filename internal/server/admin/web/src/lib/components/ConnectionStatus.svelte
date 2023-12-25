<script lang="ts">
  import { Badge } from "$lib/components/ui/badge";

  export let ClosedAt: string | null;
</script>

{#if ClosedAt}
  <Badge variant="destructive">Closed</Badge>
{:else}
  <Badge variant="secondary">Active</Badge>
{/if}
