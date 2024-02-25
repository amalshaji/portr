<script lang="ts">
  import Members from "$lib/components/users/members.svelte";
  import { Button } from "$lib/components/ui/button";
  import InviteUser from "$lib/components/users/invite-user.svelte";
  import { currentUser } from "$lib/store";

  let addMemberModalOpen = false;
</script>

<InviteUser bind:open={addMemberModalOpen} />

<div class="py-2 flex justify-between">
  <div></div>
  <Button
    on:click={() => (addMemberModalOpen = !addMemberModalOpen)}
    disabled={$currentUser?.role === "member"}>Add member</Button
  >
</div>

<Members />
