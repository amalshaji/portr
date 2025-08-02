<script lang="ts">
  import * as AlertDialog from "$lib/components/ui/alert-dialog";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";
  import { LoaderCircle } from "lucide-svelte";
  import { navigate } from "svelte-routing";
  import { toast } from "svelte-sonner";

  export let isOpen = false;

  let teamName = "";
  let teamSlug = "";
  let submitting = false;
  let error = "";

  const handleTeamNameChange = (e: Event) => {
    teamName = (e.target as HTMLInputElement).value;
    // Generate the slug automatically
    teamSlug = teamName
      .toLowerCase()
      .replace(/\s+/g, "-")
      .replace(/[^a-z0-9-]/g, "");
  };

  const createTeam = async () => {
    error = "";

    if (!teamName) {
      error = "Team name is required";
      return;
    }

    if (!teamSlug) {
      error = "Team slug is required";
      return;
    }

    submitting = true;

    try {
      const response = await fetch("/api/v1/team", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          name: teamName,
          slug: teamSlug,
        }),
      });

      const data = await response.json();

      if (response.ok) {
        toast.success("Team created successfully!");
        isOpen = false;
        // Navigate to the new team
        navigate(`/${teamSlug}/overview`);
      } else {
        error = data.error || data.message || "Failed to create team";
      }
    } catch (err) {
      console.error(err);
      error = "Something went wrong";
    } finally {
      submitting = false;
    }
  };
</script>

<AlertDialog.Root bind:open={isOpen}>
  <AlertDialog.Content class="sm:max-w-md">
    <AlertDialog.Header>
      <AlertDialog.Title>Create New Team</AlertDialog.Title>
      <AlertDialog.Description>
        Create a new team to manage connections and users
      </AlertDialog.Description>
    </AlertDialog.Header>

    <div class="space-y-4 py-4">
      <div class="space-y-2">
        <Label for="team-name">Team Name</Label>
        <Input
          id="team-name"
          bind:value={teamName}
          on:input={handleTeamNameChange}
          placeholder="My Awesome Team"
        />
      </div>

      <div class="space-y-2">
        <Label for="team-slug">Team Slug</Label>
        <Input
          id="team-slug"
          bind:value={teamSlug}
          placeholder="my-awesome-team"
          class="text-sm font-mono bg-gray-50"
          readonly
        />
        <p class="text-xs text-gray-500">
          The slug will be used in URLs and is automatically generated from the
          team name
        </p>
      </div>

      {#if error}
        <p class="text-sm text-red-600">{error}</p>
      {/if}
    </div>

    <AlertDialog.Footer>
      <AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
      <Button
        on:click={createTeam}
        disabled={submitting || !teamName || !teamSlug}
      >
        {#if submitting}
          <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Create Team
      </Button>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
