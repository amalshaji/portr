<script lang="ts">
  import { currentRequest } from "$lib/store";
  import { Button } from "$lib/components/ui/button";
  import { toast } from "svelte-sonner";
  import { RotateCw } from "lucide-svelte";
  import { Play } from "lucide-svelte";
  import RenderContent from "$lib/components/RenderContent.svelte";

  const convertJsonToSingleValue = (data: any) => {
    const jsonKeyValue: any = {};
    for (const key in data) {
      jsonKeyValue[key] = data[key][0];
    }
    return jsonKeyValue;
  };

  let replaying = false;

  const replayRequest = async () => {
    replaying = true;

    try {
      const response = await fetch(
        `/api/tunnels/replay/${$currentRequest?.ID}`
      );
      if (!response.ok) {
        const { message } = await response.json();
        toast.error(message);
        return;
      }
      toast.success("Request replayed successfully");
    } catch (error) {
      toast.error("Failed to replay request");
    } finally {
      replaying = false;
    }
  };
</script>

<div class="flex-1 p-6 overflow-y-auto bg-white dark:bg-gray-800">
  <h2
    class="text-xl font-semibold mb-4 text-gray-800 dark:text-gray-200 flex justify-between"
  >
    <div class="flex space-x-4 items-center">
      <div class="font-normal">{$currentRequest?.Method}</div>
      <div><code class="font-light text-lg">{$currentRequest?.Url}</code></div>
    </div>
    <div>
      <Button variant="outline" disabled={replaying} on:click={replayRequest}>
        {#if replaying}
          <RotateCw class="mr-2 w-4 h-4 animate-spin" />
        {:else}
          <Play class="mr-2 w-4 h-4" />
        {/if}
        Replay
      </Button>
    </div>
  </h2>
  <div class="space-y-6">
    <div>
      <h3 class="text-lg font-semibold text-gray-800 dark:text-gray-200">
        Request Headers
      </h3>
      <div class="rounded-md bg-[#F4F4F5] dark:bg-gray-700 p-4 mt-2">
        <pre
          class="text-sm font-mono text-gray-800 dark:text-gray-200 text-wrap break-words">{JSON.stringify(
            convertJsonToSingleValue($currentRequest?.Headers),
            null,
            2
          )}</pre>
      </div>
    </div>
    <div>
      <h3 class="text-lg font-semibold text-gray-800 dark:text-gray-200">
        Request Body
      </h3>
      <div class="rounded-md bg-[#F4F4F5] dark:bg-gray-700 p-4 mt-2">
        <RenderContent type="request" />
      </div>
    </div>
    <div>
      <h3 class="text-lg font-semibold text-gray-800 dark:text-gray-200">
        Response Headers
      </h3>
      <div class="rounded-md bg-[#F4F4F5] dark:bg-gray-700 p-4 mt-2">
        <pre
          class="text-sm font-mono text-gray-800 dark:text-gray-200 text-wrap break-words">{JSON.stringify(
            convertJsonToSingleValue($currentRequest?.ResponseHeaders),
            null,
            2
          )}</pre>
      </div>
    </div>
    <div>
      <h3 class="text-lg font-semibold text-gray-800 dark:text-gray-200">
        Response Body
      </h3>
      <div class="rounded-md bg-[#F4F4F5] p-4 mt-2">
        <RenderContent type="response" />
      </div>
    </div>
  </div>
</div>
