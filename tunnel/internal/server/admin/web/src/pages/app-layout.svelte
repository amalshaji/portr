<script lang="ts">
  import IssueLink from "$lib/components/issue-link.svelte";
  import { Button } from "$lib/components/ui/button";
</script>

<div class="h-screen w-full flex">
  <div class="hidden md:flex md:w-[220px] lg:w-[260px] border-r border-gray-300 bg-white">
    <slot name="sidebar" />
  </div>
  <div class="flex flex-col flex-1 min-w-0">
    <header
      class="flex h-14 items-center gap-4 border-b border-gray-300 bg-white px-4 lg:h-[60px] lg:px-6 flex-shrink-0"
    >
      <div class="w-full flex-1 flex items-center justify-end">
        <IssueLink />
      </div>
    </header>
    <main class="flex-1 overflow-y-auto bg-white">
      <div class="flex flex-col gap-4 p-4 lg:gap-6 lg:p-6">
        <slot name="body" />
      </div>
    </main>
  </div>
</div>
