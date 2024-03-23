<script>
  import Goback from "$lib/components/goback.svelte";
  import Newteam from "$lib/components/newteam.svelte";
  import AppLayout from "./app-layout.svelte";
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
    <Newteam />
  </div>
</AppLayout>
