<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as AlertDialog from "$lib/components/ui/alert-dialog";

  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";

  import * as Select from "$lib/components/ui/select";
  const roles = [
    { value: "member", label: "Member" },
    { value: "admin", label: "Admin" },
  ];

  let email: string = "",
    role = roles[0];

  export let open: boolean = false;

  const invite = async () => {
    const res = await fetch("/api/invite", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ Email: email, Role: role.value }),
    });

    if (res.ok) {
      open = false;
    }
  };
</script>

<AlertDialog.Root bind:open>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>Invite user</AlertDialog.Title>
      <AlertDialog.Description>
        <div class="mt-4 space-y-4">
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
      <AlertDialog.Action on:click={invite}>Continue</AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
