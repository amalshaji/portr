<script lang="ts">
  import EmailSettingsCard from "$lib/components/settings/emailSettingsCard.svelte";
  import { settings } from "$lib/store";
  import { onMount } from "svelte";

  const getSettings = async () => {
    const res = await fetch("/api/v1/settings/");
    settings.set(await res.json());
    console.log($settings);
  };

  onMount(() => {
    getSettings();
  });
</script>

<EmailSettingsCard />
