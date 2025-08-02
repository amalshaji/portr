<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { Input } from "$lib/components/ui/input";
  import { LoaderCircle } from "lucide-svelte";
  import { toast } from "svelte-sonner";
  import ErrorText from "../../lib/components/ErrorText.svelte";

  let teamName: string = "",
    teamNameError = "";

  let isUpdating = false;

  const createTeam = async () => {
    if (!teamName) {
      teamNameError = "Team name is required";
      return;
    }
    isUpdating = true;
    try {
      const res = await fetch("/api/v1/team/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          name: teamName,
        }),
      });
      if (res.ok) {
        const data = await res.json();
        location.href = `/${data.slug}/overview`;
      } else {
        const errorData = await res.json();
        teamNameError =
          errorData.error || errorData.message || "Failed to create team";
        toast.error("Something went wrong");
      }
    } catch (err) {
      throw err;
    } finally {
      isUpdating = false;
    }
  };
</script>

<Card.Root class="border-none shadow-none w-1/2">
  <Card.Header class="space-y-3">
    <Card.Title>Create new team</Card.Title>
  </Card.Header>
  <Card.Content class="space-y-2">
    <div class="sm:col-span-3">
      <Input
        type="text"
        id="team_name"
        placeholder="portr"
        bind:value={teamName}
        class={teamNameError ? "border-red-500" : ""}
      />
      {#if teamNameError}
        <ErrorText error={teamNameError} />
      {/if}
    </div>
  </Card.Content>
  <Card.Footer>
    <Button on:click={createTeam} disabled={isUpdating}>
      {#if isUpdating}
        <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
      {/if}
      Create team
    </Button>
  </Card.Footer>
</Card.Root>
