<script lang="ts">
  import * as Select from "$lib/components/ui/select";
  import { currentUser } from "$lib/store";
  import type { Team } from "$lib/types";
  import { BadgePlus } from "lucide-svelte";
  import { getContext, onDestroy } from "svelte";
  import { Link } from "svelte-routing";

  let team = getContext("team");

  let teams: Team[] = [];
  let selected: any, currentTeam;

  const subscriber = currentUser.subscribe((user) => {
    teams = user?.Teams as Team[];
    currentTeam = $currentUser?.Teams.filter((item) => item.Slug === team)[0];
    selected = { label: currentTeam?.Name, value: currentTeam?.Slug };
  });

  const switchTeams = (item: any) => {
    if (item.value === "new_team") {
      return;
    }
    location.href = `/${item.value}/overview`;
  };

  onDestroy(() => subscriber());
</script>

<Select.Root bind:selected onSelectedChange={switchTeams}>
  <Select.Trigger>
    <div class="flex items-center space-x-2">
      <img
        src="https://api.dicebear.com/7.x/initials/svg?seed={selected.value}"
        alt={selected.Label}
        class="w-5 h-5 rounded-full"
      />
      <Select.Value />
    </div>
  </Select.Trigger>
  <Select.Content>
    <Select.Group>
      <Select.Label>Your teams</Select.Label>
      {#each teams as team}
        <Select.Item value={team.Slug} label={team.Name}>
          <img
            src="https://api.dicebear.com/7.x/initials/svg?seed={team.Slug}"
            alt={team.Name}
            class="w-5 h-5 rounded-full mr-2"
          />
          {team.Name}
        </Select.Item>
      {/each}
    </Select.Group>
  </Select.Content>
  <Select.Input name="favoriteFruit" />
</Select.Root>
