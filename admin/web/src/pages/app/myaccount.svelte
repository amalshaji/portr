<script lang="ts">
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { currentUser } from "$lib/store";
  import { toast } from "svelte-sonner";
  import { Loader } from "lucide-svelte";
  import { getContext } from "svelte";

  let team = getContext("team") as string;

  let firstName: string = "",
    lastName: string = "",
    password = "",
    confirmPassword = "",
    passwordError = "";

  currentUser.subscribe((currentTeamUser) => {
    firstName = currentTeamUser?.user.first_name || "";
    lastName = currentTeamUser?.user.last_name || "";
  });

  let isUpdating = false,
    isChangingPassword = false;

  const updateProfile = async () => {
    isUpdating = true;
    try {
      const res = await fetch("/api/v1/user/me/update", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team,
        },
        body: JSON.stringify({
          first_name: firstName,
          last_name: lastName,
        }),
      });
      if (res.ok) {
        const data = await res.json();
        // @ts-ignore
        $currentUser = {
          ...$currentUser,
          user: { ...$currentUser?.user, ...data },
        };
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

  const changePassword = async () => {
    passwordError = "";

    if (password !== confirmPassword) {
      passwordError = "Passwords do not match";
      return;
    }

    isChangingPassword = true;

    try {
      const res = await fetch("/api/v1/user/me/change_password", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team,
        },
        body: JSON.stringify({ password }),
      });
      if (res.ok) {
        const data = await res.json();
        toast.success("Password changed");
      } else {
        toast.error("Something went wrong");
      }
    } catch (err) {
      throw err;
    } finally {
      isChangingPassword = false;
    }
  };
</script>

<div class="space-y-4">
  <Card.Root class="rounded-sm border-none shadow-none">
    <Card.Header class="space-y-3">
      <Card.Title class="text-lg">Profile</Card.Title>
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
          <Loader class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Save changes
      </Button>
    </Card.Footer>
  </Card.Root>
</div>
<div class="space-y-4">
  <Card.Root class="rounded-sm border-none shadow-none">
    <Card.Header class="space-y-3">
      <Card.Title class="text-lg">Change password</Card.Title>
      <Card.Description>Set/change your login password.</Card.Description>
    </Card.Header>
    <Card.Content class="space-y-2">
      <div class="grid grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
        <div class="sm:col-span-3">
          <Label for="password">Password</Label>
          <Input type="password" id="password" bind:value={password} />
        </div>

        <div class="sm:col-span-3">
          <Label for="confirm_password">Confirm password</Label>
          <Input
            type="password"
            id="confirm_password"
            bind:value={confirmPassword}
            class={passwordError && "border-red-500"}
          />
          {#if passwordError}
            <p class="text-red-500 text-xs italic">{passwordError}</p>
          {/if}
        </div>
      </div>
    </Card.Content>
    <Card.Footer>
      <Button
        on:click={changePassword}
        disabled={isChangingPassword || password === ""}
      >
        {#if isChangingPassword}
          <Loader class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Save changes
      </Button>
    </Card.Footer>
  </Card.Root>
</div>
