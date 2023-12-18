<script lang="ts">
  import EmailSettingsCard from "$lib/components/settings/emailSettingsCard.svelte";
  import { settings } from "$lib/store";
  import { onMount } from "svelte";

  const getSettings = async () => {
    const res = await fetch("/api/setting/all");
    settings.set(await res.json());
  };

  onMount(() => {
    getSettings();
  });
</script>

<EmailSettingsCard />
