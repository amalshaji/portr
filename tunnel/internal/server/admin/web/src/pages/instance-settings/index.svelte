<script lang="ts">
  import { onMount } from "svelte";
  import { navigate } from "svelte-routing";

  const redirectToEmailSettings = () => {
    navigate("/instance-settings/email");
  };

  onMount(() => {
    redirectToEmailSettings();
  });
</script>
