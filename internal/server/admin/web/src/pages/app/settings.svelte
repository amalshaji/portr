<script lang="ts">
  import EmailSettingsCard from "$lib/components/settings/emailSettingsCard.svelte";
  import ProfileSettingsCard from "$lib/components/settings/profileSettingsCard.svelte";
  import SignupSettingsCard from "$lib/components/settings/signupSettingsCard.svelte";
  import * as Tabs from "$lib/components/ui/tabs";
</script>

<div class="container mx-auto py-16 w-3/4">
  <p class="text-2xl py-4">Settings</p>

  <Tabs.Root value="profile" class="mx-auto">
    <Tabs.List class="grid w-full grid-cols-3">
      <Tabs.Trigger value="profile">Profile</Tabs.Trigger>
      <Tabs.Trigger value="signup">Signup</Tabs.Trigger>
      <Tabs.Trigger value="email">Email</Tabs.Trigger>
    </Tabs.List>
    <Tabs.Content value="profile">
      <ProfileSettingsCard />
    </Tabs.Content>
    <Tabs.Content value="signup">
      <SignupSettingsCard />
    </Tabs.Content>
    <Tabs.Content value="email">
      <EmailSettingsCard />
    </Tabs.Content>
  </Tabs.Root>
</div>
