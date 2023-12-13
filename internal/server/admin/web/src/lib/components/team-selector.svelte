<script lang="ts">
  import * as Select from "$lib/components/ui/select";
  import { currentUser } from "$lib/store";
  import type { Team } from "$lib/types";
  import { getContext, onDestroy } from "svelte";

  let team = getContext("team");

  let teams: Team[] = [];
  let selected: any, currentTeam;

  const subscriber = currentUser.subscribe((user) => {
    teams = user?.Teams as Team[];
    currentTeam = $currentUser?.Teams.filter((item) => item.Slug === team)[0];
    selected = { label: currentTeam?.Name, value: currentTeam?.Slug };
  });

  const switchTeams = (item: any) => {
    location.href = `/${item.value}/overview`;
  };

  onDestroy(() => subscriber());
</script>

<Select.Root bind:selected onSelectedChange={switchTeams}>
  <Select.Trigger class="w-[180px]">
    <Select.Value />
  </Select.Trigger>
  <Select.Content>
    <Select.Group on:change={(e) => console.log(e)}>
      <Select.Label>Switch team</Select.Label>
      {#each teams as team}
        <Select.Item value={team.Slug} label={team.Name}
          >{team.Name}</Select.Item
        >
      {/each}
    </Select.Group>
  </Select.Content>
  <Select.Input name="favoriteFruit" />
</Select.Root>
