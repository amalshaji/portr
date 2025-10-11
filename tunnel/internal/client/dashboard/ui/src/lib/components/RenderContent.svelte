<script lang="ts">
  import { currentRequest } from "$lib/store";
  import Highlight from "svelte-highlight";
  import json from "svelte-highlight/languages/json";
  import RenderFormUrlEncoded from "./RenderFormUrlEncoded.svelte";
  import RenderMultipartFormData from "./RenderMultipartFormData.svelte";
  import ErrorDisplay from "./ErrorDisplay.svelte";
  import Button from "./ui/button/button.svelte";

  export let type: "request" | "response";

  let headers, contentType: string, contentLength: string;

  const convertJsonToSingleValue = (data: any) => {
    const jsonKeyValue: any = {};
    for (const key in data) {
      jsonKeyValue[key] = data[key][0];
    }
    return jsonKeyValue;
  };

  const loadResponse = async (url: string) => {
    const response = await fetch(url);
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.message || errorData.error || 'Failed to load response');
    }
    return await response.text();
  };

  currentRequest.subscribe((value) => {
    if (type === "request") {
      headers = convertJsonToSingleValue(value?.Headers);
    } else {
      headers = convertJsonToSingleValue(value?.ResponseHeaders);
    }

    contentType = headers["Content-Type"] as string;
    contentLength = headers["Content-Length"] || "0";
  });
</script>

{#if $currentRequest}
  {#if contentLength === "0"}
    <div class="text-gray-500 dark:text-gray-400 p-4">No content</div>
  {:else if contentType.startsWith("application/json")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`)}
      <div class="p-4 text-gray-500">Loading...</div>
    {:then response}
      <div class="overflow-auto max-h-[600px]">
        <Highlight
          language={json}
          code={JSON.stringify(JSON.parse(response), null, 2)}
        />
      </div>
    {:catch error}
      <ErrorDisplay message={error.message} />
    {/await}
  {:else if contentType.startsWith("application/x-www-form-urlencoded")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`)}
      <div class="p-4 text-gray-500">Loading...</div>
    {:then response}
      <RenderFormUrlEncoded data={response} />
    {:catch error}
      <ErrorDisplay message={error.message} />
    {/await}
  {:else if contentType.startsWith("multipart/form-data")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`)}
      <div class="p-4 text-gray-500">Loading...</div>
    {:then response}
      <RenderMultipartFormData />
    {:catch error}
      <ErrorDisplay message={error.message} />
    {/await}
  {:else if contentType.startsWith("image/")}
    <img
      src={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
      alt="portr"
    />
  {:else if contentType.startsWith("video/")}
    <!-- svelte-ignore a11y-media-has-caption -->
    <video controls>
      <source
        src={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
        type={contentType}
      />
    </video>
  {:else if contentType.startsWith("audio/")}
    <!-- svelte-ignore a11y-media-has-caption -->
    <audio controls>
      <source
        src={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
        type={contentType}
      />
    </audio>
  {:else if contentType.startsWith("text/html")}
    <!-- svelte-ignore a11y-missing-attribute -->
    <iframe
      src={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
      width="100%"
      height="400px"
    ></iframe>
  {:else if contentType.startsWith("text/")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`)}
      <div class="p-4 text-gray-500">Loading...</div>
    {:then response}
      <pre class="p-4 overflow-auto max-h-[600px]">{response}</pre>
    {:catch error}
      <ErrorDisplay message={error.message} />
    {/await}
  {:else}
    <Button
      href={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
      class="rounded-sm"
      variant="outline">Load response</Button
    >
  {/if}
{/if}
