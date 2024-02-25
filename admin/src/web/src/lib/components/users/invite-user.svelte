<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as AlertDialog from "$lib/components/ui/alert-dialog";
  import { Reload } from "radix-icons-svelte";
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";

  import * as Select from "$lib/components/ui/select";
  import { toast } from "svelte-sonner";
  import { currentUser, users } from "$lib/store";
  import { getContext } from "svelte";
  import ApiError from "../ApiError.svelte";

  import { Checkbox } from "$lib/components/ui/checkbox";
  let set_superuser = false;

  const roles = [
    { value: "member", label: "Member" },
    { value: "admin", label: "Admin" },
  ];

  let email: string = "",
    role = roles[0];

  let error = "";

  const setSuperuser = () => {
    console.log("test");
    if (set_superuser) {
      role = roles[1];
    }
  };

  $: set_superuser, setSuperuser();

  export let open: boolean = false;

  let isLoading = false;

  let team = getContext("team") as string;

  const add_member = async () => {
    error = "";
    isLoading = true;
    try {
      const res = await fetch(`/api/v1/team/add`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team,
        },
        body: JSON.stringify({
          email: email,
          role: role.value,
          set_superuser: set_superuser,
        }),
      });

      if (res.ok) {
        const data = await res.json();
        if (data !== null) {
          users.update((users) => {
            return [...users, { ...data, role: role.value }];
          });
          toast.success(`${email} added to team`);
        }
        role = roles[0];
        email = "";
        set_superuser = false;
        open = false;
      } else {
        error = (await res.json()).message;
      }
    } catch (err) {
      throw err;
    } finally {
      isLoading = false;
    }
  };
</script>

<AlertDialog.Root bind:open>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>Add member</AlertDialog.Title>
      <AlertDialog.Description>
        <div class="mt-4 space-y-4">
          {#if error}
            <ApiError {error} />
          {/if}
          <div class="sm:col-span-3 space-y-2">
            <Label for="email">Email</Label>
            <Input
              type="text"
              id="email"
              class="text-black"
              placeholder="John"
              bind:value={email}
            />
          </div>

          <div class="sm:col-span-3">
            <Select.Root bind:selected={role}>
              <Select.Trigger class="w-[180px]" disabled={set_superuser}>
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
              <Checkbox id="set_superuser" bind:checked={set_superuser} />
              <Label
                for="set_superuser"
                class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                Make superuser
              </Label>
            </div>
          {/if}
        </div>
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer class="mt-8">
      <AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
      <Button on:click={add_member} disabled={isLoading}>
        {#if isLoading}
          <Reload class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Add
      </Button>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
