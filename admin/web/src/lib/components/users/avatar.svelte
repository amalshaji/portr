<script lang="ts">
  import * as Avatar from "$lib/components/ui/avatar";

  export let url: string | null | undefined, fallback: string;
</script>

<Avatar.Root>
  <Avatar.Image
    src={url ? url : `https://api.dicebear.com/7.x/rings/svg?seed=${fallback}`}
    alt={fallback}
  />
</Avatar.Root>
