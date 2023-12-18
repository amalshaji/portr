<script lang="ts">
  import { Input } from "$lib/components/ui/input";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { toast } from "svelte-sonner";
  import { Reload } from "radix-icons-svelte";

  let teamName: string = "";

  let isUpdating = false;

  const createTeam = async () => {
    isUpdating = true;
    try {
      const res = await fetch("/api/team", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          Name: teamName,
        }),
      });
      if (res.ok) {
        const data = await res.json();
        location.href = `/${data.Slug}/overview`;
      } else {
        toast.error("Something went wrong");
      }
    } catch (err) {
      throw err;
    } finally {
      isUpdating = false;
    }
  };
</script>

<div class="container mt-4">
  <div class="space-y-4">
    <Card.Root class="rounded-sm">
      <Card.Header class="space-y-3">
        <Card.Title>Enter team name</Card.Title>
      </Card.Header>
      <Card.Content class="space-y-2">
        <div class="sm:col-span-3">
          <Input
            type="text"
            id="team_name"
            placeholder="localport"
            bind:value={teamName}
          />
        </div>
      </Card.Content>
      <Card.Footer>
        <Button on:click={createTeam} disabled={isUpdating}>
          {#if isUpdating}
            <Reload class="mr-2 h-4 w-4 animate-spin" />
          {/if}
          Create team
        </Button>
      </Card.Footer>
    </Card.Root>
  </div>
</div>
