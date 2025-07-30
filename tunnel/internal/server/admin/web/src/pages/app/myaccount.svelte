<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";
  import { currentUser } from "$lib/store";
  import { LoaderCircle, User, KeySquare, KeyRound, Copy } from "lucide-svelte";
  import { getContext } from "svelte";
  import { toast } from "svelte-sonner";
  import { copyCodeToClipboard } from "$lib/utils";

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
    isChangingPassword = false,
    isRotatingSecretKey = false;

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
      const res = await fetch("/api/v1/user/me/change-password", {
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

  const rotateSecretKey = async () => {
    isRotatingSecretKey = true;
    try {
      const res = await fetch(`/api/v1/user/me/rotate-secret-key`, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team,
        },
      });
      if (res.ok) {
        const secret_key = (await res.json()).secret_key;
        // @ts-ignore
        $currentUser = {
          ...$currentUser,
          secret_key: secret_key,
        };
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

  const copySecretKey = () => {
    if ($currentUser?.secret_key) {
      copyCodeToClipboard(String($currentUser.secret_key));
      toast.success("Secret key copied to clipboard");
    }
  };
</script>

<div class="space-y-6">
  <div class="flex justify-between items-center">
    <h1 class="text-2xl font-bold tracking-tight">Account & Settings</h1>
  </div>

  <Card.Root class="shadow-sm">
    <Card.Header>
      <Card.Title class="text-xl">Profile Information</Card.Title>
      <Card.Description>Update your personal details</Card.Description>
    </Card.Header>
    <Card.Content>
      <div class="bg-gray-50 rounded-lg p-6 border border-gray-100 space-y-4">
        <div class="flex items-center gap-3">
          <User class="h-5 w-5 text-primary" />
          <div>
            <h3 class="text-sm font-medium">Personal Details</h3>
            <p class="text-xs text-gray-500">Your name as it appears across the platform</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
          <div>
            <Label for="first_name">First Name</Label>
            <Input
              type="text"
              id="first_name"
              placeholder="John"
              bind:value={firstName}
              class="bg-white"
            />
          </div>

          <div>
            <Label for="last_name">Last Name</Label>
            <Input
              type="text"
              id="last_name"
              placeholder="Doe"
              bind:value={lastName}
              class="bg-white"
            />
          </div>
        </div>

        <Button
          on:click={updateProfile}
          disabled={isUpdating}
          class="mt-2"
        >
          {#if isUpdating}
            <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
          {/if}
          Save Profile
        </Button>
      </div>
    </Card.Content>
  </Card.Root>

  <Card.Root class="shadow-sm">
    <Card.Header>
      <Card.Title class="text-xl">Security</Card.Title>
      <Card.Description>Manage your security settings and credentials</Card.Description>
    </Card.Header>
    <Card.Content class="space-y-6">
      <div class="bg-gray-50 rounded-lg p-6 border border-gray-100 space-y-4">
        <div class="flex items-center gap-3">
          <KeySquare class="h-5 w-5 text-primary" />
          <div>
            <h3 class="text-sm font-medium">Change Password</h3>
            <p class="text-xs text-gray-500">Update your login credentials</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
          <div>
            <Label for="password">New Password</Label>
            <Input
              type="password"
              id="password"
              bind:value={password}
              class="bg-white"
            />
          </div>

          <div>
            <Label for="confirm_password">Confirm Password</Label>
            <Input
              type="password"
              id="confirm_password"
              bind:value={confirmPassword}
              class={`bg-white ${passwordError && "border-red-500"}`}
            />
            {#if passwordError}
              <p class="text-red-500 text-xs mt-1">{passwordError}</p>
            {/if}
          </div>
        </div>

        <Button
          on:click={changePassword}
          disabled={isChangingPassword || password === ""}
          class="mt-2"
        >
          {#if isChangingPassword}
            <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
          {/if}
          Update Password
        </Button>
      </div>

      <div class="bg-gray-50 rounded-lg p-6 border border-gray-100 space-y-4">
        <div class="flex items-center gap-3">
          <KeyRound class="h-5 w-5 text-primary" />
          <div>
            <h3 class="text-sm font-medium">API Secret Key</h3>
            <p class="text-xs text-gray-500">Used to authenticate client connections for team: <span class="font-medium">{team}</span></p>
          </div>
        </div>

        <div class="relative">
          <Input
            type="text"
            readonly
            value={$currentUser?.secret_key}
            class="pr-10 font-mono text-sm bg-white"
          />
          <Button
            variant="ghost"
            size="sm"
            class="absolute right-1 top-1/2 -translate-y-1/2 h-8 w-8 p-0"
            on:click={copySecretKey}
          >
            <Copy class="h-4 w-4" />
          </Button>
        </div>

        <div>
          <Button
            variant="outline"
            on:click={rotateSecretKey}
            disabled={isRotatingSecretKey}
            class="mt-2"
          >
            {#if isRotatingSecretKey}
              <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
            {/if}
            Rotate Key
          </Button>
          <p class="text-xs text-gray-500 mt-2">
            Rotating your key will invalidate your previous key immediately
          </p>
        </div>
      </div>
    </Card.Content>
  </Card.Root>
</div>
