<script lang="ts">
  import * as AlertDialog from "$lib/components/ui/alert-dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { currentUser, users } from "$lib/store";
  import type { TeamUser } from "$lib/types";
  import { Loader, Trash2 } from "lucide-svelte";
  import { getContext } from "svelte";
  import { toast } from "svelte-sonner";

  export let user: TeamUser;

  const team = getContext("team") as string;

  let isLoading = false;

  const removeUser = async () => {
    isLoading = true;
    try {
      const response = await fetch(`/api/v1/team/users/${user.id}`, {
        method: "DELETE",
        headers: {
          "x-team-slug": team,
        },
      });
      if (response.ok) {
        users.update((users) => users.filter((u) => u.id !== user.id));
        isLoading = false;
        toast.success("User removed from team");
      }
    } catch (err) {
      console.error(err);
      isLoading = false;
    }
  };

  const canDelete = !(
    $currentUser?.role === "member" ||
    user.id === $currentUser?.id ||
    (user.user.is_superuser && !$currentUser?.user.is_superuser)
  );
</script>

<AlertDialog.Root>
  <AlertDialog.Trigger asChild let:builder>
    <Button
      builders={[builder]}
      variant="ghost"
      class="outline-none"
      disabled={!canDelete}
    >
      <Trash2 class="h-4 w-4" />
    </Button>
  </AlertDialog.Trigger>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>Are you absolutely sure?</AlertDialog.Title>
      <AlertDialog.Description>
        You are about to remove <strong>{user.user.email}</strong> from the team.
        This action cannot be undone.
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
      <Button on:click={removeUser} disabled={isLoading}>
        {#if isLoading}
          <Loader class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Remove
      </Button>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
