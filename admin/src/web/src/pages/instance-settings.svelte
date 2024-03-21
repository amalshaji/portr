<script lang="ts">
  import EmailSettingsCard from "$lib/components/settings/emailSettingsCard.svelte";
  import { instanceSettings } from "$lib/store";
  import { onMount } from "svelte";

  const getSettings = async () => {
    const res = await fetch("/api/v1/instance-settings/");
    instanceSettings.set(await res.json());
  };

  const goBack = () => {
    console.log(history.state);
    if (window.history.length > 2) {
      window.history.back();
    } else {
      location.href = "/";
    }
  };

  onMount(() => {
    getSettings();
  });
</script>

<div class="grid place-items-center p-16 mx-0">
  <button on:click={goBack} class="text-sm underline">go back</button>
  <EmailSettingsCard />
</div>
