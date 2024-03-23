<script lang="ts">
  import { currentRequest } from "$lib/store";
  import Button from "./ui/button/button.svelte";

  export let type: "request" | "response";

  let headers, body, contentType: string, contentLength: string;

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

  if (type === "request") {
    headers = convertJsonToSingleValue($currentRequest?.Headers);
    body = $currentRequest?.Body;
  } else {
    headers = convertJsonToSingleValue($currentRequest?.ResponseHeaders);
    body = $currentRequest?.ResponseBody;
  }

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
  {#if contentType.startsWith("application/json")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}`) then response}
      <pre class="text-sm">{JSON.stringify(JSON.parse(response), null, 2)}</pre>
    {/await}
  {:else if contentType.startsWith("image/")}
    <img src={`/api/tunnels/render/${$currentRequest?.ID}`} alt="portr" />
  {:else if contentType.startsWith("video/")}
    <!-- svelte-ignore a11y-media-has-caption -->
    <video controls>
      <source
        src={`/api/tunnels/render/${$currentRequest?.ID}`}
        type={contentType}
      />
    </video>
  {:else if contentType.startsWith("audio/")}
    <!-- svelte-ignore a11y-media-has-caption -->
    <audio controls>
      <source
        src={`/api/tunnels/render/${$currentRequest?.ID}`}
        type={contentType}
      />
    </audio>
  {:else if contentType.startsWith("text/html")}
    <!-- svelte-ignore a11y-missing-attribute -->
    <iframe
      src={`/api/tunnels/render/${$currentRequest?.ID}`}
      width="100%"
      height="400px"
    ></iframe>
  {:else if contentType.startsWith("text/")}
    {#await loadResponse(`/api/tunnels/render/${$currentRequest?.ID}`) then response}
      <pre>{response}</pre>
    {/await}
  {:else}
    <Button
      href={`/api/tunnels/render/${$currentRequest?.ID}`}
      class="rounded-sm"
      variant="outline">Load response</Button
    >
  {/if}
{/if}
