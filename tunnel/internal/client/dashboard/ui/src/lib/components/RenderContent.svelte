<script lang="ts">
  import { currentRequest } from "$lib/store";
  import RenderFormUrlEncoded from "./RenderFormUrlEncoded.svelte";
  import RenderMultipartFormData from "./RenderMultipartFormData.svelte";
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
    <div class="text-gray-500 dark:text-gray-400">No content</div>
  {:else if contentType.startsWith("application/json")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`) then response}
      <pre class="text-sm">{JSON.stringify(JSON.parse(response), null, 2)}</pre>
    {/await}
  {:else if contentType.startsWith("application/x-www-form-urlencoded")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`) then response}
      <RenderFormUrlEncoded data={response} />
    {/await}
  {:else if contentType.startsWith("multipart/form-data")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`) then response}
      <!-- <pre>{response}</pre> -->
      <RenderMultipartFormData />
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
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`) then response}
      <pre>{response}</pre>
    {/await}
  {:else}
    <Button
      href={`/api/tunnels/render/${$currentRequest?.ID}?type=${type}`}
      class="rounded-sm"
      variant="outline">Load response</Button
    >
  {/if}
{/if}
