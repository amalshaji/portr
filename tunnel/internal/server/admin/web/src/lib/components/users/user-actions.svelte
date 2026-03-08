<script lang="ts">
  import type { TeamUser } from "$lib/types";
  import Delete from "./delete.svelte";
  import ResetPassword from "./reset-password.svelte";

  export let user: TeamUser;
</script>

<div class="flex items-center gap-1">
  <ResetPassword {user} />
  <Delete {user} />
</div>