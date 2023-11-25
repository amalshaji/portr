<script lang="ts">
  import { Label } from "$lib/components/ui/label";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import * as RadioGroup from "$lib/components/ui/radio-group";
  import { Textarea } from "$lib/components/ui/textarea";

  let radioValue: string = "signup_requires_invite",
    onUpdate: any,
    random_user_signup_allowed_domains: string;

  let show_allowed_domains_textarea = false;

  $: radioValue === "allow_random_user_signup"
    ? (show_allowed_domains_textarea = true)
    : (show_allowed_domains_textarea = false);
</script>

<Card.Root>
  <Card.Header class="space-y-3">
    <Card.Title>Signup</Card.Title>
    <Card.Description>Configure who can signup for an account</Card.Description>
  </Card.Header>
  <Card.Content class="space-y-2">
    <RadioGroup.Root bind:value={radioValue} class="space-y-2">
      <div class="flex items-center space-x-2">
        <RadioGroup.Item value="signup_requires_invite" id="r2" />
        <Label for="r2">Requires invite</Label>
      </div>
      <div class="flex items-center space-x-2">
        <RadioGroup.Item value="allow_random_user_signup" id="r3" />
        <Label for="r3">Allow anyone to signup</Label>
      </div>
      <RadioGroup.Input name="spacing" />

      {#if show_allowed_domains_textarea}
        <div class="px-10">
          <Label for="allowed_domains">Allowed domains</Label>
          <Textarea
            bind:value={random_user_signup_allowed_domains}
            id="allowed_domains"
            placeholder="localport.app,example.com"
          />
        </div>
      {/if}
    </RadioGroup.Root></Card.Content
  >
  <Card.Footer>
    <Button on:click={onUpdate}>Save changes</Button>
  </Card.Footer>
</Card.Root>
