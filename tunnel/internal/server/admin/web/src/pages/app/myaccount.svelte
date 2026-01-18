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
    <h1 class="text-2xl font-bold tracking-tight text-black">Account & Settings</h1>
  </div>

  <div class="border border-gray-300 bg-white">
    <div class="p-6 border-b border-gray-300">
      <h2 class="text-xl font-semibold text-black">Profile Information</h2>
      <p class="text-gray-600 mt-1">Update your personal details</p>
    </div>
    <div class="p-6">
      <div class="bg-gray-50 border border-gray-300 p-6 space-y-4">
        <div class="flex items-center gap-3">
          <User class="h-5 w-5 text-black" />
          <div>
            <h3 class="text-sm font-medium text-black">Personal Details</h3>
            <p class="text-xs text-gray-600">Your name as it appears across the platform</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
          <div>
            <Label for="first_name" class="text-black">First Name</Label>
            <Input
              type="text"
              id="first_name"
              placeholder="John"
              bind:value={firstName}
              class="bg-white border border-gray-400 focus:border-black focus:outline-none focus-visible:outline-none focus-visible:ring-0"
              style="border-radius: 0;"
            />
          </div>

          <div>
            <Label for="last_name" class="text-black">Last Name</Label>
            <Input
              type="text"
              id="last_name"
              placeholder="Doe"
              bind:value={lastName}
              class="bg-white border border-gray-400 focus:border-black focus:outline-none focus-visible:outline-none focus-visible:ring-0"
              style="border-radius: 0;"
            />
          </div>
        </div>

        <Button
          on:click={updateProfile}
          disabled={isUpdating}
          class="mt-2 border-2 border-black bg-black text-white hover:bg-gray-800 focus:outline-none focus:ring-0 disabled:opacity-50"
          style="border-radius: 0;"
        >
          {#if isUpdating}
            <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
          {/if}
          Save Profile
        </Button>
      </div>
    </div>
  </div>

  <div class="border border-gray-300 bg-white">
    <div class="p-6 border-b border-gray-300">
      <h2 class="text-xl font-semibold text-black">Security</h2>
      <p class="text-gray-600 mt-1">Manage your security settings and credentials</p>
    </div>
    <div class="p-6 space-y-6">
      <div class="bg-gray-50 border border-gray-300 p-6 space-y-4">
        <div class="flex items-center gap-3">
          <KeySquare class="h-5 w-5 text-black" />
          <div>
            <h3 class="text-sm font-medium text-black">Change Password</h3>
            <p class="text-xs text-gray-600">Update your login credentials</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
          <div>
            <Label for="password" class="text-black">New Password</Label>
            <Input
              type="password"
              id="password"
              bind:value={password}
              class="bg-white border border-gray-400 focus:border-black focus:outline-none focus-visible:outline-none focus-visible:ring-0"
              style="border-radius: 0;"
            />
          </div>

          <div>
            <Label for="confirm_password" class="text-black">Confirm Password</Label>
            <Input
              type="password"
              id="confirm_password"
              bind:value={confirmPassword}
              class={`bg-white border focus:outline-none focus-visible:outline-none focus-visible:ring-0 ${passwordError ? "border-red-600" : "border-gray-400 focus:border-black"}`}
              style="border-radius: 0;"
            />
            {#if passwordError}
              <p class="text-red-600 text-xs mt-1">{passwordError}</p>
            {/if}
          </div>
        </div>

        <Button
          on:click={changePassword}
          disabled={isChangingPassword || password === ""}
          class="mt-2 border-2 border-black bg-black text-white hover:bg-gray-800 focus:outline-none focus:ring-0 disabled:opacity-50"
          style="border-radius: 0;"
        >
          {#if isChangingPassword}
            <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
          {/if}
          Update Password
        </Button>
      </div>

      <div class="bg-gray-50 border border-gray-300 p-6 space-y-4">
        <div class="flex items-center gap-3">
          <KeyRound class="h-5 w-5 text-black" />
          <div>
            <h3 class="text-sm font-medium text-black">API Secret Key</h3>
            <p class="text-xs text-gray-600">Used to authenticate client connections for team: <span class="font-medium">{team}</span></p>
          </div>
        </div>

        <div class="relative">
          <Input
            type="text"
            readonly
            value={$currentUser?.secret_key}
            class="pr-10 font-mono text-sm bg-white border border-gray-400 focus:border-black focus:outline-none focus-visible:outline-none focus-visible:ring-0"
            style="border-radius: 0;"
          />
          <Button
            variant="ghost"
            size="sm"
            class="absolute right-1 top-1/2 -translate-y-1/2 h-8 w-8 p-0 hover:bg-gray-100 focus:outline-none focus:ring-0"
            style="border-radius: 0;"
            on:click={copySecretKey}
          >
            <Copy class="h-4 w-4 text-black" />
          </Button>
        </div>

        <div>
          <Button
            variant="outline"
            on:click={rotateSecretKey}
            disabled={isRotatingSecretKey}
            class="mt-2 border border-gray-400 bg-white text-black hover:bg-gray-50 focus:outline-none focus:ring-0 disabled:opacity-50"
            style="border-radius: 0;"
          >
            {#if isRotatingSecretKey}
              <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
            {/if}
            Rotate Key
          </Button>
          <p class="text-xs text-gray-600 mt-2">
            Rotating your key will invalidate your previous key immediately
          </p>
        </div>
      </div>
    </div>
  </div>
</div>
