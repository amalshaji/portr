<script lang="ts">
  import { Badge } from "$lib/components/ui/badge";

  export let email: string, is_superuser: boolean;
</script>

<p class="flex space-x-2">
  <span>{email}</span>
  <span
    >{#if is_superuser}
      <Badge class="rounded-full">superuser</Badge>
    {/if}
  </span>
</p>
