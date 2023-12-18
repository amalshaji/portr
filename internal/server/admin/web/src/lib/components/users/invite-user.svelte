<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as AlertDialog from "$lib/components/ui/alert-dialog";
  import { ExclamationTriangle, Reload } from "radix-icons-svelte";
  import * as Alert from "$lib/components/ui/alert";
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";

  import * as Select from "$lib/components/ui/select";
  import { toast } from "svelte-sonner";
  import { invites } from "$lib/store";
  import { getContext } from "svelte";

  const roles = [
    { value: "member", label: "Member" },
    { value: "admin", label: "Admin" },
  ];

  let email: string = "",
    role = roles[0];

  let error = "";

  export let open: boolean = false;

  let isLoading = false;

  let team = getContext("team");

  const invite = async () => {
    error = "";
    isLoading = true;
    try {
      const res = await fetch(`/api/${team}/invite`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ Email: email, Role: role.value }),
      });

      if (res.ok) {
        const data = await res.json();
        if (data !== null) {
          invites.update((invites) => {
            return [data, ...invites];
          });
          toast.success("Invitation sent");
        } else {
          toast.success("User added to team");
        }
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
      <AlertDialog.Title>Invite user</AlertDialog.Title>
      <AlertDialog.Description>
        <div class="mt-4 space-y-4">
          {#if error}
            <Alert.Root>
              <ExclamationTriangle class="h-4 w-4" />
              <Alert.Title>Error</Alert.Title>
              <Alert.Description>
                {error}
              </Alert.Description>
            </Alert.Root>
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
              <Select.Trigger class="w-[180px]">
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
        </div>
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
      <Button on:click={invite} disabled={isLoading}>
        {#if isLoading}
          <Reload class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Invite
      </Button>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
