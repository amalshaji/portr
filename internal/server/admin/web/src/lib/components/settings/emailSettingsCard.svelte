<script lang="ts">
  import { Label } from "$lib/components/ui/label";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { Textarea } from "$lib/components/ui/textarea";

  let onUpdate: any,
    invite_email_template: string = `
Hi {{email}}, you have been invited to join {{org_name}} on LocalPort. Click the link below to get started.

<a href="{{inviteUrl}}">Click here to create your account</a>
`.trim();
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
      bind:value={invite_email_template}
      id="invite_email_template"
    />
  </Card.Content>
  <Card.Footer>
    <Button on:click={onUpdate}>Save changes</Button>
  </Card.Footer>
</Card.Root>
