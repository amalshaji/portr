<script lang="ts">
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { currentTeamUser, currentUser } from "$lib/store";
  import { toast } from "svelte-sonner";
  import { Reload } from "radix-icons-svelte";
  import { getContext } from "svelte";

  let team = getContext("team");

  let firstName: string = "",
    lastName: string = "";
  currentUser.subscribe((user) => {
    firstName = user?.FirstName || "";
    lastName = user?.LastName || "";
    console.log(user);
  });

  let isUpdating = false,
    isRotatingSecretKey = false;

  const updateProfile = async () => {
    isUpdating = true;
    try {
      const res = await fetch("/api/user/me/update", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          firstName,
          lastName,
        }),
      });
      if (res.ok) {
        currentUser.set(await res.json());
        toast.success("Profile updated");
      } else {
        toast.error("Something went wrong");
      }
    } catch (err) {
      throw err;
    } finally {
      isUpdating = false;
    }
  };

  const copySecretToClipboard = () => {
    navigator.clipboard.writeText($currentTeamUser?.SecretKey as string);
    toast.success("Secret key copied to clipboard");
  };

  const rotateSecretKey = async () => {
    isRotatingSecretKey = true;
    try {
      const res = await fetch(`/api/${team}/user/me/rotate-secret-key`, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
      });
      if (res.ok) {
        currentTeamUser.set(await res.json());
        toast.success("New secret key generated");
      } else {
        toast.error("Something went wrong");
      }
    } catch (err) {
      console.error(err);
    } finally {
      isRotatingSecretKey = false;
    }
  };
</script>

<div class="space-y-4">
  <Card.Root class="rounded-sm border-none shadow-none">
    <Card.Header class="space-y-3">
      <Card.Title>Profile</Card.Title>
      <Card.Description>Some basic information about you</Card.Description>
    </Card.Header>
    <Card.Content class="space-y-2">
      <div class="grid grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
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

  <Card.Root class="rounded-sm border-none shadow-none">
    <Card.Header class="space-y-3">
      <Card.Title>Secret key</Card.Title>
      <Card.Description
        >The secret key to authenticate client connection</Card.Description
      >
    </Card.Header>
    <Card.Content class="space-y-2 flex items-center">
      <Input
        type="text"
        readonly
        value={$currentTeamUser?.SecretKey}
        on:click={copySecretToClipboard}
      />
    </Card.Content>
    <Card.Footer>
      <Button on:click={rotateSecretKey} disabled={isRotatingSecretKey}>
        {#if isRotatingSecretKey}
          <Reload class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Rotate key
      </Button>
    </Card.Footer>
  </Card.Root>
</div>
