<script lang="ts">
  import * as Tabs from "$lib/components/ui/tabs";
  import Invites from "$lib/components/users/invites.svelte";
  import Members from "$lib/components/users/members.svelte";
  import { Button } from "$lib/components/ui/button";
  import InviteUser from "$lib/components/users/invite-user.svelte";
  import { currentTeamUser } from "$lib/store";

  let currentTab = "members";

  let inviteUserModalOpen = false;
</script>

<InviteUser bind:open={inviteUserModalOpen} />

<div class="py-2">
  {#if currentTab === "invites"}
    <Button
      on:click={() => (inviteUserModalOpen = !inviteUserModalOpen)}
      disabled={$currentTeamUser?.Role === "member"}>Invite user</Button
    >
  {/if}
</div>

<Tabs.Root bind:value={currentTab} class="mx-auto">
  <Tabs.List class="grid w-full grid-cols-2">
    <Tabs.Trigger value="members">Members</Tabs.Trigger>
    <Tabs.Trigger value="invites">Invites</Tabs.Trigger>
  </Tabs.List>
  <Tabs.Content value="members">
    <Members />
  </Tabs.Content>
  <Tabs.Content value="invites">
    <Invites />
  </Tabs.Content>
</Tabs.Root>
