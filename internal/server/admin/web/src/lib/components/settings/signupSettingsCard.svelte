<script lang="ts">
  import { Label } from "$lib/components/ui/label";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import * as RadioGroup from "$lib/components/ui/radio-group";
  import { Textarea } from "$lib/components/ui/textarea";
  import { settings } from "$lib/store";
  import { onDestroy } from "svelte";
  import { toast } from "svelte-sonner";
  import { Reload } from "radix-icons-svelte";

  let isUpdating = false;

  let signupRequiresInvite: boolean,
    allowRandomUserSignup: boolean,
    randomUserSignupAllowedDomains: string;

  let randomUserSignupAllowedDomainsValid = true;

  let radioValue: string;

  let settingsUnSubscriber = settings.subscribe((settings) => {
    signupRequiresInvite = settings?.SignupRequiresInvite || true;
    allowRandomUserSignup = settings?.AllowRandomUserSignup || false;
    randomUserSignupAllowedDomains =
      settings?.RandomUserSignupAllowedDomains || "";
    radioValue = settings?.AllowRandomUserSignup
      ? "allow_random_user_signup"
      : "signup_requires_invite";
  });

  let show_allowed_domains_textarea = true;

  $: if (radioValue === "allow_random_user_signup") {
    signupRequiresInvite = false;
    allowRandomUserSignup = true;
    show_allowed_domains_textarea = true;
  } else {
    signupRequiresInvite = true;
    allowRandomUserSignup = false;
    show_allowed_domains_textarea = false;
  }

  const updateSignupSettings = async () => {
    if (allowRandomUserSignup) {
      const domainsValid = validateDomains();
      if (!domainsValid) {
        randomUserSignupAllowedDomainsValid = false;
        return;
      } else {
        randomUserSignupAllowedDomainsValid = true;
      }
    }
    isUpdating = true;
    try {
      const res = await fetch("/api/setting/signup/update", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          SignupRequiresInvite: signupRequiresInvite,
          AllowRandomUserSignup: allowRandomUserSignup,
          RandomUserSignupAllowedDomains: randomUserSignupAllowedDomains,
        }),
      });
      if (res.ok) {
        // @ts-ignore
        settings.update((settings) => {
          return {
            ...settings,
            SignupRequiresInvite: signupRequiresInvite,
            AllowRandomUserSignup: allowRandomUserSignup,
            RandomUserSignupAllowedDomains: randomUserSignupAllowedDomains,
          };
        });
        toast.success("Signup settings updated");
      } else {
        toast.error("Failed to update signup settings");
      }
    } catch (err) {
      console.error(err);
    } finally {
      isUpdating = false;
    }
  };

  const validateDomains = () => {
    const domains = randomUserSignupAllowedDomains.split(",");
    for (let i = 0; i < domains.length; i++) {
      const domain = domains[i].trim();
      if (domain.split(".").length < 2) {
        return false;
      }
    }
    return true;
  };

  onDestroy(() => {
    settingsUnSubscriber();
  });
</script>

<Card.Root class="rounded-sm">
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
            bind:value={randomUserSignupAllowedDomains}
            id="allowed_domains"
            class={randomUserSignupAllowedDomainsValid ? "" : "border-red-300"}
            placeholder="localport.app,example.com"
          />
        </div>
      {/if}
    </RadioGroup.Root></Card.Content
  >
  <Card.Footer>
    <Button on:click={updateSignupSettings} disabled={isUpdating}>
      {#if isUpdating}
        <Reload class="mr-2 h-4 w-4 animate-spin" />
      {/if}
      Save changes
    </Button>
  </Card.Footer>
</Card.Root>
