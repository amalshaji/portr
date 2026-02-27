<script lang="ts">
  import Button from "./ui/button/button.svelte";
  import { Download } from "lucide-svelte";

  export let message: string;
  export let canDownload: boolean = false;
  export let downloadUrl: string = "";
  export let contentLength: string = "";
</script>

<div class="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded">
  <div class="flex items-start gap-2">
    <span class="text-red-600 dark:text-red-400 text-xl">⚠️</span>
    <div class="flex-1">
      <h4 class="font-semibold text-red-800 dark:text-red-300 mb-1">Error Loading Body</h4>
      <p class="text-red-700 dark:text-red-400 text-sm mb-2">{message}</p>
      {#if canDownload && downloadUrl}
        <div class="flex items-center justify-between gap-3 mt-3 pt-3 border-t border-red-200 dark:border-red-800/50">
          {#if contentLength}
            <span class="text-xs text-red-600/70 dark:text-red-400/70 font-mono">
              Raw body ({contentLength} bytes)
            </span>
          {/if}
          <Button
            href={downloadUrl}
            class="rounded-sm"
            variant="outline"
            size="sm"
          >
            <Download class="w-3.5 h-3.5 mr-1.5" />
            Download Raw
          </Button>
        </div>
      {/if}
    </div>
  </div>
</div>
