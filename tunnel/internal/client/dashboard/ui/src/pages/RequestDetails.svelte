<script lang="ts">
  import HttpBadge from "$lib/components/HttpBadge.svelte";
  import RenderContent from "$lib/components/RenderContent.svelte";
  import { Button } from "$lib/components/ui/button";
  import { currentRequest } from "$lib/store";
  import { convertDateToHumanReadable } from "$lib/utils";
  import { Loader, Play } from "lucide-svelte";
  import Highlight from "svelte-highlight";
  import json from "svelte-highlight/languages/json";
  import atomonelight from "svelte-highlight/styles/atom-one-light";
  import { toast } from "svelte-sonner";

  const convertJsonToSingleValue = (data: any) => {
    const jsonKeyValue: any = {};
    for (const key in data) {
      jsonKeyValue[key] = data[key][0];
    }
    return jsonKeyValue;
  };

  export let viewParent;

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

<svelte:head>
  {@html atomonelight}
</svelte:head>

<div class="flex-1 p-6 overflow-y-auto bg-white dark:bg-gray-800">
  <h2
    class="text-xl font-medium text-gray-800 dark:text-gray-200 flex justify-between"
  >
    <div class="flex space-x-4 items-center">
      <HttpBadge method={$currentRequest?.Method} />
      <div>
        <p class="font-light text-lg">{$currentRequest?.Url}</p>
      </div>
    </div>
    <div class="flex items-center gap-2">
      {#if $currentRequest?.ParentID !== ""}
        <Button on:click={viewParent}>
          <p class="text-sm">View parent</p>
        </Button>
      {/if}
      <Button
        variant="outline"
        class="shadow-md"
        disabled={replaying}
        on:click={replayRequest}
      >
        {#if replaying}
          <Loader class="mr-2 w-4 h-4 animate-spin" />
        {:else}
          <Play class="mr-2 w-4 h-4" />
        {/if}
        Replay
      </Button>
    </div>
  </h2>
  <div>
    {#if $currentRequest?.LoggedAt}
      <p class="font-light mb-4">
        {convertDateToHumanReadable($currentRequest.LoggedAt)}
      </p>
    {/if}
  </div>
  <div class="space-y-6">
    <div>
      <h3 class="text-lg font-medium text-gray-800 dark:text-gray-200">
        Request Headers
      </h3>
      <div class="rounded-md bg-[#FAFAFA] border dark:bg-gray-700 p-4 mt-2">
        <Highlight
          language={json}
          code={JSON.stringify(
            convertJsonToSingleValue($currentRequest?.Headers),
            null,
            2
          )}
        />
      </div>
    </div>
    <div>
      <h3 class="text-lg font-medium text-gray-800 dark:text-gray-200">
        Request Body
      </h3>
      <div class="rounded-md bg-[#FAFAFA] border dark:bg-gray-700 p-4 mt-2">
        <RenderContent type="request" />
      </div>
    </div>
    <div>
      <h3 class="text-lg font-medium text-gray-800 dark:text-gray-200">
        Response Headers
      </h3>
      <div class="rounded-md bg-[#FAFAFA] border dark:bg-gray-700 p-4 mt-2">
        <Highlight
          language={json}
          code={JSON.stringify(
            convertJsonToSingleValue($currentRequest?.ResponseHeaders),
            null,
            2
          )}
        />
      </div>
    </div>
    <div>
      <h3 class="text-lg font-medium text-gray-800 dark:text-gray-200">
        Response Body
      </h3>
      <div class="rounded-md bg-[#FAFAFA] border p-4 mt-2">
        <RenderContent type="response" />
      </div>
    </div>
  </div>
</div>

<style>
  :global(.hljs) {
    font-size: 14px !important;
    padding: 16px !important;
  }
</style>
