<script lang="ts">
  import { teamSettings } from "$lib/store";
  import { onMount } from "svelte";
  import { getContext } from "svelte";
  import { Label } from "$lib/components/ui/label";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";

  import { onDestroy } from "svelte";
  import { toast } from "svelte-sonner";
  import { Reload } from "radix-icons-svelte";
  import { Switch } from "$lib/components/ui/switch";
  import { Input } from "$lib/components/ui/input";

  let githubOrgWebhookSecret: string,
    githubOrgPat: string,
    isUpdating = false,
    autoInviteOrgMembers: boolean;

  let settingsUnSubscriber = teamSettings.subscribe((settings) => {
    // @ts-ignore
    autoInviteOrgMembers = settings?.auto_invite_github_org_members;
  });

  let team = getContext("team") as string;

  const updateTeamSettings = async () => {
    isUpdating = true;
    const requestBody = {
      auto_invite_github_org_members: autoInviteOrgMembers,
    };

    if (githubOrgWebhookSecret) {
      // @ts-ignore
      requestBody.github_org_webhook_secret = githubOrgWebhookSecret;
    }
    if (githubOrgPat) {
      // @ts-ignore
      requestBody.github_org_pat = githubOrgPat;
    }

    try {
      const res = await fetch("/api/v1/team/settings", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team,
        },
        body: JSON.stringify(requestBody),
      });
      if (res.ok) {
        teamSettings.set(await res.json());
        toast.success("Team settings updated");
      } else {
        toast.error("Something went wrong");
      }
    } catch (err) {
      console.error(err);
    } finally {
      isUpdating = false;
    }
  };

  onDestroy(() => {
    settingsUnSubscriber();
  });

  const getSettings = async () => {
    const res = await fetch("/api/v1/team/settings", {
      headers: {
        "x-team-slug": team,
      },
    });
    teamSettings.set(await res.json());
  };

  onMount(() => {
    getSettings();
  });
</script>

<Card.Root class="rounded-sm border-none shadow-none w-1/2">
  <Card.Header class="space-y-3">
    <Card.Title class="text-lg">Team Settings</Card.Title>
    <Card.Description>Configure team settings</Card.Description>
  </Card.Header>
  <Card.Content class="space-y-2">
    <div class="space-y-4">
      <div class="flex items-center gap-4">
        <Label for="auto_invite_org_members">Auto invite org members</Label>
        <Switch bind:checked={autoInviteOrgMembers} />
      </div>
      <div>
        <Label for="github_org_webhook_secret"
          >GitHub organization webhook secret</Label
        >
        <Input
          bind:value={githubOrgWebhookSecret}
          placeholder="••••••••"
          id="github_org_webhook_secret"
        />
      </div>
      <div>
        <Label for="github_org_pat"
          >GitHub organization personal access token</Label
        >
        <Input
          bind:value={githubOrgPat}
          placeholder="••••••••"
          id="github_org_pat"
        />
      </div>
    </div>
  </Card.Content>
  <Card.Footer>
    <Button on:click={updateTeamSettings} disabled={isUpdating}>
      {#if isUpdating}
        <Reload class="mr-2 h-4 w-4 animate-spin" />
      {/if}
      Save changes
    </Button>
  </Card.Footer>
</Card.Root>
