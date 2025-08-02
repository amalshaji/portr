<script lang="ts">
  import * as AlertDialog from "$lib/components/ui/alert-dialog";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";
  import { LoaderCircle } from "lucide-svelte";

  import * as Select from "$lib/components/ui/select";
  import { currentUser, users } from "$lib/store";
  import { getContext } from "svelte";
  import { toast } from "svelte-sonner";
  import ErrorAlert from "$lib/components/ui/error-alert.svelte";

  import { Checkbox } from "$lib/components/ui/checkbox";
  import CopyToClipboard from "../copyToClipboard.svelte";
  let set_superuser = false;

  const roles = [
    { value: "member", label: "Member" },
    { value: "admin", label: "Admin" },
  ];

  let email: string = "",
    role = roles[0];

  let error = "",
    generatedPassword = "";

  // Email validation function
  const isValidEmail = (email: string): boolean => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email.trim());
  };

  // Reactive statement to check if form is valid
  $: isFormValid = email.trim() !== "" && isValidEmail(email.trim());

  const setSuperuser = () => {
    console.log("test");
    if (set_superuser) {
      role = roles[1];
    }
  };

  $: set_superuser, setSuperuser();

  export let open: boolean = false,
    displayPassword: boolean = false;

  let isLoading = false;

  let team = getContext("team") as string;

  const add_member = async () => {
    error = "";

    // Frontend validation
    if (!email.trim()) {
      error = "Email is required";
      return;
    }

    if (!isValidEmail(email.trim())) {
      error = "Please enter a valid email address";
      return;
    }

    isLoading = true;
    try {
      const res = await fetch(`/api/v1/team/add`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team,
        },
        body: JSON.stringify({
          email: email.trim(),
          role: role.value,
          set_superuser: set_superuser,
        }),
      });

      if (res.ok) {
        const { team_user, password } = await res.json();
        if (team_user !== null) {
          users.update((users) => {
            return [...users, { ...team_user, role: role.value }];
          });
          toast.success(`${email} added to team`);
        }
        role = roles[0];
        email = "";
        set_superuser = false;
        open = false;
        if (password) {
          generatedPassword = password;
          displayPassword = true;
        }
      } else {
        const errorData = await res.json();
        error = errorData.message || errorData.error || "Failed to add member";
      }
    } catch (err) {
      console.error("Add member error:", err);
      error = "Network error. Please try again.";
    } finally {
      isLoading = false;
    }
  };
</script>

<AlertDialog.Root bind:open>
  <AlertDialog.Content
    class="border border-gray-300 bg-white"
    style="border-radius: 0;"
  >
    <AlertDialog.Header class="border-b border-gray-300 pb-4">
      <AlertDialog.Title class="text-black">Add member</AlertDialog.Title>
      <AlertDialog.Description>
        <div class="mt-4 space-y-4">
          {#if error}
            <ErrorAlert message={error} />
          {/if}
          <div class="sm:col-span-3 space-y-2">
            <Label for="email" class="text-black">Email</Label>
            <Input
              type="email"
              id="email"
              class={`text-black border focus:outline-none focus-visible:outline-none focus-visible:ring-0 ${
                email.trim() !== "" && !isValidEmail(email.trim())
                  ? "border-red-600"
                  : "border-gray-400 focus:border-black"
              }`}
              style="border-radius: 0;"
              placeholder="john@example.com"
              bind:value={email}
            />
            {#if email.trim() !== "" && !isValidEmail(email.trim())}
              <p class="text-red-600 text-xs mt-1">
                Please enter a valid email address
              </p>
            {/if}
          </div>

          <div class="sm:col-span-3">
            <Select.Root bind:selected={role}>
              <Select.Trigger
                class="w-[180px] border border-gray-400 focus:border-black focus:outline-none focus:ring-0"
                style="border-radius: 0;"
                disabled={set_superuser}
              >
                <Select.Value placeholder="Select a role" />
              </Select.Trigger>
              <Select.Content>
                <Select.Group>
                  <Select.Label>Role</Select.Label>
                  {#each roles as role}
                    <Select.Item value={role.value} label={role.label}
                      >{role.label}</Select.Item
                    >
                  {/each}
                </Select.Group>
              </Select.Content>
              <Select.Input />
            </Select.Root>
          </div>
          {#if $currentUser?.user.is_superuser}
            <div class="sm:col-span-3 items-center">
              <Checkbox
                id="set_superuser"
                bind:checked={set_superuser}
                class="border border-gray-400 focus:border-black"
                style="border-radius: 0;"
              />
              <Label
                for="set_superuser"
                class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 text-black ml-2"
              >
                Make superuser
              </Label>
            </div>
          {/if}
        </div>
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer class="border-gray-300 pt-4">
      <AlertDialog.Cancel
        class="border border-gray-400 bg-white text-black hover:bg-gray-50 focus:outline-none focus:ring-0"
        style="border-radius: 0;">Cancel</AlertDialog.Cancel
      >
      <Button
        on:click={add_member}
        disabled={isLoading || !isFormValid}
        class="border-2 border-black bg-black text-white hover:bg-gray-800 focus:outline-none focus:ring-0 disabled:opacity-50 disabled:cursor-not-allowed"
        style="border-radius: 0;"
      >
        {#if isLoading}
          <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Add member
      </Button>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>

<AlertDialog.Root bind:open={displayPassword}>
  <AlertDialog.Content
    class="border border-gray-300 bg-white"
    style="border-radius: 0;"
  >
    <AlertDialog.Header class="border-b border-gray-300 pb-4">
      <AlertDialog.Title class="text-black"
        >Here's your password</AlertDialog.Title
      >
      <AlertDialog.Description>
        <div class="mt-4 space-y-4">
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
    <AlertDialog.Footer class="mt-8 border-t border-gray-300 pt-4">
      <AlertDialog.Cancel
        on:click={() => (generatedPassword = "")}
        class="border border-gray-400 bg-white text-black hover:bg-gray-50 focus:outline-none focus:ring-0"
        style="border-radius: 0;">Done</AlertDialog.Cancel
      >
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
