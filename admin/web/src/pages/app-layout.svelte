<script lang="ts">
  import IssueLink from "$lib/components/issue-link.svelte";
</script>

<div
  class="grid min-h-screen w-full md:grid-cols-[220px_1fr] lg:grid-cols-[260px_1fr]"
>
  <div class="hidden border-r bg-muted/40 md:block">
    <slot name="sidebar" />
  </div>
  <div class="flex flex-col">
    <header
      class="flex h-14 items-center gap-4 border-b bg-muted/40 px-4 lg:h-[60px] lg:px-6"
    >
      <div class="w-full flex-1"></div>
      <IssueLink />
    </header>
    <main class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
      <slot name="body" />
    </main>
  </div>
</div>
