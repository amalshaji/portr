<script lang="ts">
  import EmailSettingsCard from "$lib/components/settings/emailSettingsCard.svelte";
  import SignupSettingsCard from "$lib/components/settings/signupSettingsCard.svelte";
  import * as Tabs from "$lib/components/ui/tabs";
  import { settings } from "$lib/store";
  import { onMount } from "svelte";

  const getSettings = async () => {
    const res = await fetch("/api/settings/all");
    settings.set(await res.json());
  };

  onMount(() => {
    getSettings();
  });
</script>

<div class="container mx-auto py-16 w-3/4">
  <p class="text-2xl py-4">Settings</p>

  <Tabs.Root value="signup" class="mx-auto">
    <Tabs.List class="grid w-full grid-cols-2">
      <Tabs.Trigger value="signup">Signup</Tabs.Trigger>
      <Tabs.Trigger value="email">Email</Tabs.Trigger>
    </Tabs.List>
    <Tabs.Content value="signup">
      <SignupSettingsCard />
    </Tabs.Content>
    <Tabs.Content value="email">
      <EmailSettingsCard />
    </Tabs.Content>
  </Tabs.Root>
</div>
