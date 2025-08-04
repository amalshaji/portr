<script lang="ts">
  import * as AlertDialog from "$lib/components/ui/alert-dialog/index.js";
  import { Button } from "$lib/components/ui/button/index.js";
  import { Input } from "$lib/components/ui/input/index.js";
  import { currentUser } from "$lib/store";
  import type { TeamUser } from "$lib/types";
  import { LoaderCircle, RotateCcw } from "lucide-svelte";
  import { getContext } from "svelte";
  import { toast } from "svelte-sonner";
  import CopyToClipboard from "../copyToClipboard.svelte";

  export let user: TeamUser;

  let resetOpen = false;
  let passwordModalOpen = false;
  let generatedPassword = "";

  const team = getContext("team") as string;

  let isLoading = false;

  const resetPassword = async () => {
    isLoading = true;
    try {
      const response = await fetch(`/api/v1/team/users/${user.id}/reset-password`, {
        method: "POST",
        headers: {
          "x-team-slug": team,
        },
      });
      if (response.ok) {
        const { password } = await response.json();
        generatedPassword = password;
        resetOpen = false;
        passwordModalOpen = true;
        toast.success("Password reset successfully");
      } else {
        const errorData = await response.json();
        toast.error(errorData.error || "Failed to reset password");
      }
    } catch (err) {
      console.error(err);
      toast.error("Network error. Please try again.");
    } finally {
      isLoading = false;
    }
  };

  const canResetPassword = $currentUser?.user.is_superuser && user.id !== $currentUser?.id;
</script>

<AlertDialog.Root bind:open={resetOpen}>
  <AlertDialog.Trigger asChild let:builder>
    <Button
      builders={[builder]}
      variant="ghost"
      class="outline-none"
      disabled={!canResetPassword}
      title="Reset Password"
    >
      <RotateCcw class="h-4 w-4" />
    </Button>
  </AlertDialog.Trigger>
  <AlertDialog.Content
    class="border border-gray-300 bg-white"
    style="border-radius: 0;"
  >
    <AlertDialog.Header class="border-b border-gray-300 pb-4">
      <AlertDialog.Title class="text-black">Reset Password</AlertDialog.Title>
      <AlertDialog.Description>
        You are about to reset the password for <strong>{user.user.email}</strong>.
        A new password will be generated and shown to you.
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer class="border-gray-300 pt-4">
      <AlertDialog.Cancel
        class="border border-gray-400 bg-white text-black hover:bg-gray-50 focus:outline-none focus:ring-0"
        style="border-radius: 0;"
      >
        Cancel
      </AlertDialog.Cancel>
      <Button
        on:click={resetPassword}
        disabled={isLoading}
        class="border-2 border-black bg-black text-white hover:bg-gray-800 focus:outline-none focus:ring-0 disabled:opacity-50 disabled:cursor-not-allowed"
        style="border-radius: 0;"
      >
        {#if isLoading}
          <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Reset Password
      </Button>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>

<AlertDialog.Root bind:open={passwordModalOpen}>
  <AlertDialog.Content
    class="border border-gray-300 bg-white"
    style="border-radius: 0;"
  >
    <AlertDialog.Header>
      <AlertDialog.Title class="text-black">New Password Generated</AlertDialog.Title>
      <AlertDialog.Description>
        <div class="space-y-4">
          <p>Here's the new password for <strong>{user.user.email}</strong>:</p>
          <div class="sm:col-span-3 space-y-2">
            <div class="flex items-center space-x-1">
              <Input
                readonly
                class="text-black border border-gray-400 focus:border-black focus:outline-none focus-visible:outline-none focus-visible:ring-0"
                style="border-radius: 0;"
                bind:value={generatedPassword}
              />
              <CopyToClipboard text={generatedPassword} />
            </div>
          </div>
        </div>
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer class="mt-2">
      <AlertDialog.Cancel
        on:click={() => (generatedPassword = "")}
        class="border border-gray-400 bg-white text-black hover:bg-gray-50 focus:outline-none focus:ring-0"
        style="border-radius: 0;"
      >
        Done
      </AlertDialog.Cancel>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>