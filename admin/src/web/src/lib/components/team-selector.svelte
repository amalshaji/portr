<script lang="ts">
  import * as Select from "$lib/components/ui/select";
  import { currentUserTeams } from "$lib/store";
  import { getContext, onDestroy, onMount } from "svelte";
  import * as Avatar from "$lib/components/ui/avatar";

  let team = getContext("team") as string;

  let selected: any, currentTeam;

  const get_my_teams = async () => {
    const response = await fetch(`/api/v1/user/me/teams`, {
      headers: {
        "Content-Type": "application/json",
      },
    });
    currentUserTeams.set(await response.json());
  };

  const subscriber = currentUserTeams.subscribe((teams) => {
    currentTeam = teams.filter((item) => item.slug === team)[0];
    selected = { label: currentTeam?.name, value: currentTeam?.slug };
  });

  const switchTeams = (item: any) => {
    location.href = `/${item.value}/overview`;
  };

  onMount(() => {
    get_my_teams();
  });

  onDestroy(() => subscriber());
</script>

<Select.Root bind:selected onSelectedChange={switchTeams}>
  <Select.Trigger class="text-[15px] tracking-tight">
    <div class="flex items-center space-x-2">
      <Avatar.Root class="w-6 h-6 rounded-full">
        <Avatar.Image
          src="https://api.dicebear.com/7.x/initials/svg?seed={selected.value}&backgroundColor=transparent&textColor=000000"
          alt={selected.label}
          class="w-6 h-6 rounded-full border mr-2"
        />
      </Avatar.Root>
      <Select.Value />
    </div>
  </Select.Trigger>
  <Select.Content>
    <Select.Group>
      <Select.Label>Your teams</Select.Label>
      {#each $currentUserTeams as team}
        <Select.Item value={team.slug} label={team.name}>
          <img
            src="https://api.dicebear.com/7.x/initials/svg?seed={team.slug}&backgroundColor=transparent&textColor=000000"
            alt={team.name}
            class="w-5 h-5 rounded-full border mr-2"
          />
          {team.name}
        </Select.Item>
      {/each}
    </Select.Group>
  </Select.Content>
  <Select.Input name="favoriteFruit" />
</Select.Root>
