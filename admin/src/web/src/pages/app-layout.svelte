<script lang="ts">
  import IssueLink from "$lib/components/issue-link.svelte";
</script>

<div class="flex">
  <aside
    class="sticky top-0 flex flex-col w-64 h-screen px-2 py-8 overflow-y-auto border-r rtl:border-r-0 rtl:border-l bg-[#FBFBFB] dark:bg-gray-900"
  >
    <slot name="sidebar" />
  </aside>
  <aside class="w-full">
    <div
      class="py-2 px-8 flex justify-between border-b bg-[#FBFBFB] dark:bg-gray-900"
    >
      <div></div>
      <div class="flex items-center space-x-2">
        <IssueLink />
      </div>
    </div>
    <div class="mx-auto pb-16 pt-6 w-full px-16">
      <slot name="body" />
    </div>
  </aside>
</div>
