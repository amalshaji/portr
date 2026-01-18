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

<div class="border border-gray-300 bg-white w-1/2">
  <div class="p-6 border-b border-gray-300">
    <h2 class="text-xl font-semibold text-black">Create new team</h2>
  </div>
  <div class="p-6 space-y-4">
    <div class="sm:col-span-3">
      <Input
        type="text"
        id="team_name"
        placeholder="portr"
        bind:value={teamName}
        class={`border focus:outline-none focus-visible:outline-none focus-visible:ring-0 ${teamNameError ? "border-red-600" : "border-gray-400 focus:border-black"}`}
        style="border-radius: 0;"
      />
      {#if teamNameError}
        <ErrorText error={teamNameError} />
      {/if}
    </div>
  </div>
  <div class="p-6 border-t border-gray-300">
    <Button 
      on:click={createTeam} 
      disabled={isUpdating}
      class="border-2 border-black bg-black text-white hover:bg-gray-800 focus:outline-none focus:ring-0 disabled:opacity-50"
      style="border-radius: 0;"
    >
      {#if isUpdating}
        <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
      {/if}
      Create team
    </Button>
  </div>
</div>
