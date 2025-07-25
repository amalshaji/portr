<script lang="ts">
  import Members from "$lib/components/users/members.svelte";
  import * as Card from "$lib/components/ui/card";
</script>

<div class="space-y-6">
  <h1 class="text-2xl font-bold tracking-tight">Team Members</h1>

  <Card.Root class="shadow-sm">
    <Card.Header>
      <Card.Title class="text-xl">Team Members</Card.Title>
      <Card.Description>Manage your team members and their access levels</Card.Description>
    </Card.Header>
    <Card.Content>
      <Members />
    </Card.Content>
  </Card.Root>
</div>
