<script lang="ts">
  import EmailSettingsCard from "$lib/components/settings/emailSettingsCard.svelte";
  import { instanceSettings } from "$lib/store";
  import { onMount } from "svelte";

  import Goback from "$lib/components/goback.svelte";
  import AppLayout from "./app-layout.svelte";

  const getSettings = async () => {
    const res = await fetch("/api/v1/instance-settings/");
    instanceSettings.set(await res.json());
  };

  onMount(() => {
    getSettings();
  });
</script>

<AppLayout>
  <div slot="sidebar">
    <div class="flex flex-col justify-between flex-1 mt-6 mx-4">
      <nav class="flex-1 -mx-3 space-y-3">
        <Goback />
      </nav>
    </div>
  </div>
  <div slot="body">
    <EmailSettingsCard />
  </div>
</AppLayout>
