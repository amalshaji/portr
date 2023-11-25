<script lang="ts">
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { currentUser } from "$lib/store";
  import { toast } from "svelte-sonner";
  import { Reload } from "radix-icons-svelte";

  let firstName: string = "",
    lastName: string = "";

  currentUser.subscribe((user) => {
    firstName = user?.FirstName || "";
    lastName = user?.LastName || "";
  });

  let isUpdating = false;

  const updateProfile = async () => {
    isUpdating = true;
    try {
      const res = await fetch("/api/users/me/update", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          firstName,
          lastName,
        }),
      });
      $currentUser = await res.json();
      toast.success("Profile updated successfully");
    } catch (err) {
      throw err;
    } finally {
      isUpdating = false;
    }
  };
</script>

<Card.Root>
  <Card.Header class="space-y-3">
    <Card.Title>Profile</Card.Title>
    <Card.Description>Some basic information about you</Card.Description>
  </Card.Header>
  <Card.Content class="space-y-2">
    <div class="mt-10 grid grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
      <div class="sm:col-span-3">
        <Label for="first_name">First Name</Label>
        <Input
          type="text"
          id="first_name"
          placeholder="John"
          bind:value={firstName}
        />
      </div>

      <div class="sm:col-span-3">
        <Label for="first_name">Last Name</Label>
        <Input
          type="text"
          id="first_name"
          placeholder="Wick"
          bind:value={lastName}
        />
      </div>
    </div>
  </Card.Content>
  <Card.Footer>
    <Button on:click={updateProfile} disabled={isUpdating}>
      {#if isUpdating}
        <Reload class="mr-2 h-4 w-4 animate-spin" />
      {/if}
      Save changes
    </Button>
  </Card.Footer>
</Card.Root>
