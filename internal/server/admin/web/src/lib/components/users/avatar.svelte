<script lang="ts">
  import * as Avatar from "$lib/components/ui/avatar";

  export let url: string, fallback: string;
</script>

<Avatar.Root>
  <Avatar.Image src={url} alt={fallback} />
  <Avatar.Fallback>{fallback}</Avatar.Fallback>
</Avatar.Root>
