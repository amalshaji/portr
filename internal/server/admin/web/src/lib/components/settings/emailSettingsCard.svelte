<script lang="ts">
  import { Label } from "$lib/components/ui/label";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { Textarea } from "$lib/components/ui/textarea";
  import { settings } from "$lib/store";
  import { onDestroy } from "svelte";
  import { toast } from "svelte-sonner";
  import { Reload } from "radix-icons-svelte";

  let userInviteEmailTemplate: string,
    isUpdating = false;

  let settingsUnSubscriber = settings.subscribe((settings) => {
    userInviteEmailTemplate = settings?.UserInviteEmailTemplate || "";
  });

  const updateEmailSettings = async () => {
    isUpdating = true;
    try {
      const res = await fetch("/api/setting/email/update", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          UserInviteEmailTemplate: userInviteEmailTemplate,
        }),
      });
      if (res.ok) {
        // @ts-ignore
        settings.update((settings) => {
          return {
            ...settings,
            UserInviteEmailTemplate: userInviteEmailTemplate,
          };
        });
        toast.success("Email settings updated successfully");
      }
    } catch (err) {
      console.error(err);
    } finally {
      isUpdating = false;
    }
  };

  onDestroy(() => {
    settingsUnSubscriber();
  });
</script>

<Card.Root>
  <Card.Header class="space-y-3">
    <Card.Title>Email templates</Card.Title>
    <Card.Description>Configure email template contents</Card.Description>
  </Card.Header>
  <Card.Content class="space-y-2">
    <Label for="invite_email_template">Invite email</Label>
    <Textarea
      rows={5}
      bind:value={userInviteEmailTemplate}
      id="invite_email_template"
    />
  </Card.Content>
  <Card.Footer>
    <Button on:click={updateEmailSettings} disabled={isUpdating}>
      {#if isUpdating}
        <Reload class="mr-2 h-4 w-4 animate-spin" />
      {/if}
      Save changes
    </Button>
  </Card.Footer>
</Card.Root>
