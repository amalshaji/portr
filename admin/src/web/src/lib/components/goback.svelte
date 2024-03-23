<script lang="ts">
  import Button from "./ui/button/button.svelte";
  import { SkipBack } from "lucide-svelte";

  const goBack = () => {
    console.log(history.state);
    if (window.history.length > 2) {
      window.history.back();
    } else {
      location.href = "/";
    }
  };
</script>

<Button on:click={goBack} class="w-full" variant="secondary">
  <SkipBack class="h-4 w-4 mr-2" />
  Go Back
</Button>
