<script lang="ts">
  import * as Select from "$lib/components/ui/select";
  import { currentUserTeams } from "$lib/store";
  import { getContext, onDestroy, onMount } from "svelte";
  import * as Avatar from "$lib/components/ui/avatar";
  import { createAvatar } from "@dicebear/core";
  import { initials } from "@dicebear/collection";

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

  const avatar = (text: string) => {
    const svg = createAvatar(initials, {
      seed: text,
    });
    // @ts-ignore
    return `data:image/svg+xml,${encodeURIComponent(svg)}`;
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
  <Select.Trigger class="text-[14px] border-black focus:ring-0">
    <div class="flex items-center space-x-2">
      <Avatar.Root class="w-5 h-5 rounded-full">
        <Avatar.Image
          src={avatar(selected.label)}
          alt={selected.label}
          class="w-5 h-5 rounded-full border mr-2"
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
            src={avatar(team.name)}
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
