<script lang="ts">
  import { currentRequest } from "$lib/store";
  import Highlight from "svelte-highlight";
  import json from "svelte-highlight/languages/json";
  import RenderFormUrlEncoded from "./RenderFormUrlEncoded.svelte";
  import RenderMultipartFormData from "./RenderMultipartFormData.svelte";
  import ErrorDisplay from "./ErrorDisplay.svelte";
  import Button from "./ui/button/button.svelte";
  import { Download } from "lucide-svelte";

  export let type: "request" | "response";

  let headers, contentType: string, contentLength: string;

  const convertJsonToSingleValue = (data: any) => {
    const jsonKeyValue: any = {};
    for (const key in data) {
      jsonKeyValue[key] = data[key][0];
    }
    return jsonKeyValue;
  };

  let hasError = false;

  const loadResponse = async (url: string) => {
    const response = await fetch(url);
    if (!response.ok) {
      let message = "Failed to load response";
      let canDownload = false;
      try {
        const errorData = await response.json();
        message = errorData.message || errorData.error || message;
        canDownload = !!errorData.canDownload;
      } catch {
        try {
          message = await response.text();
        } catch { /* noop */ }
      }
      hasError = true;
      const err: any = new Error(message);
      err.canDownload = canDownload;
      throw err;
    }
    hasError = false;
    return await response.text();
  };

  const tryParseJson = (text: string): { valid: boolean; formatted: string } => {
    try {
      const parsed = JSON.parse(text);
      return { valid: true, formatted: JSON.stringify(parsed, null, 2) };
    } catch {
      return { valid: false, formatted: "" };
    }
  };

  currentRequest.subscribe((value) => {
    hasError = false;
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
  {#if contentType?.startsWith("application/json")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`)}
      <div class="p-4 text-gray-500">Loading...</div>
    {:then response}
      {@const jsonResult = tryParseJson(response)}
      {#if jsonResult.valid}
        <div class="overflow-auto max-h-[600px]">
          <Highlight
            language={json}
            code={jsonResult.formatted}
          />
        </div>
      {:else}
        <ErrorDisplay
          message="Invalid JSON payload: unable to parse"
          canDownload={true}
          downloadUrl={`/api/tunnels/download/${$currentRequest?.ID}?type=${type}`}
          contentLength={contentLength}
        />
      {/if}
    {:catch error}
      <ErrorDisplay
        message={error.message}
        canDownload={error.canDownload}
        downloadUrl={`/api/tunnels/download/${$currentRequest?.ID}?type=${type}`}
        contentLength={contentLength}
      />
    {/await}
  {:else if contentType?.startsWith("application/x-www-form-urlencoded")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`)}
      <div class="p-4 text-gray-500">Loading...</div>
    {:then response}
      <RenderFormUrlEncoded data={response} />
    {:catch error}
      <ErrorDisplay
        message={error.message}
        canDownload={error.canDownload}
        downloadUrl={`/api/tunnels/download/${$currentRequest?.ID}?type=${type}`}
        contentLength={contentLength}
      />
    {/await}
  {:else if contentType?.startsWith("multipart/form-data")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`)}
      <div class="p-4 text-gray-500">Loading...</div>
    {:then response}
      <RenderMultipartFormData />
    {:catch error}
      <ErrorDisplay
        message={error.message}
        canDownload={error.canDownload}
        downloadUrl={`/api/tunnels/download/${$currentRequest?.ID}?type=${type}`}
        contentLength={contentLength}
      />
    {/await}
  {:else if contentType?.startsWith("image/")}
    <img
      src={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
      alt="portr"
    />
  {:else if contentType?.startsWith("video/")}
    <!-- svelte-ignore a11y-media-has-caption -->
    <video controls>
      <source
        src={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
        type={contentType}
      />
    </video>
  {:else if contentType?.startsWith("audio/")}
    <!-- svelte-ignore a11y-media-has-caption -->
    <audio controls>
      <source
        src={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
        type={contentType}
      />
    </audio>
  {:else if contentType?.startsWith("text/html")}
    <!-- svelte-ignore a11y-missing-attribute -->
    <iframe
      src={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
      width="100%"
      height="400px"
    ></iframe>
  {:else if contentType?.startsWith("text/")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`)}
      <div class="p-4 text-gray-500">Loading...</div>
    {:then response}
      <pre class="p-4 overflow-auto max-h-[600px]">{response}</pre>
    {:catch error}
      <ErrorDisplay
        message={error.message}
        canDownload={error.canDownload}
        downloadUrl={`/api/tunnels/download/${$currentRequest?.ID}?type=${type}`}
        contentLength={contentLength}
      />
    {/await}
  {:else}
    <Button
      href={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
      class="rounded-sm"
      variant="outline">Load response</Button
    >
  {/if}

  {#if contentLength !== "0" && !hasError}
    <div class="px-4 py-3 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 flex items-center justify-between gap-4">
      <div class="text-xs text-gray-500 dark:text-gray-400 font-mono">
        Raw body ({contentLength} bytes)
      </div>
      <Button
        href={`/api/tunnels/download/${$currentRequest?.ID}?type=${type}`}
        class="rounded-sm"
        variant="outline"
        size="sm"
      >
        <Download class="w-3.5 h-3.5 mr-1.5" />
        Download Raw
      </Button>
    </div>
  {/if}
{/if}
