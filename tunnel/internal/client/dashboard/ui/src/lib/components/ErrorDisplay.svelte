<script lang="ts">
  export let message: string;
</script>

<div class="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded">
  <div class="flex items-start gap-2">
    <span class="text-red-600 dark:text-red-400 text-xl">⚠️</span>
    <div class="flex-1">
      <h4 class="font-semibold text-red-800 dark:text-red-300 mb-1">Error Loading Body</h4>
      <p class="text-red-700 dark:text-red-400 text-sm">{message}</p>
    </div>
  </div>
</div>
