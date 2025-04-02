<script lang="ts">
  import HttpBadge from "$lib/components/HttpBadge.svelte";
  import RenderContent from "$lib/components/RenderContent.svelte";
  import { Button } from "$lib/components/ui/button";
  import { currentRequest } from "$lib/store";
  import { convertDateToHumanReadable } from "$lib/utils";
  import { ArrowLeft, ArrowUpRight, Clock, Copy, Loader, Play, RefreshCw } from "lucide-svelte";
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
  let activeTab = "request";

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

  const generateCurlCommand = () => {
    if (!$currentRequest) return '';

    // Construct full tunnel URL
    const tunnelUrl = `https://${$currentRequest.Host}${$currentRequest.Url}`;
    let curl = `curl -X ${$currentRequest.Method} '${tunnelUrl}'`;

    // Add headers
    const contentType = $currentRequest.Headers?.['Content-Type']?.[0] || '';
    const isMultipartForm = contentType.startsWith('multipart/form-data');

    if ($currentRequest.Headers) {
      Object.entries($currentRequest.Headers).forEach(([key, value]) => {
        if (Array.isArray(value) && value.length > 0 && key !== 'Content-Type' && key !== 'Content-Length') {
          curl += ` \\\n  -H '${key}: ${value[0]}'`;
        }
      });
    }

    // Add body if present
    if ($currentRequest.Body) {
      try {
        // First decode from base64
        const decodedBytes = atob($currentRequest.Body);

        if (isMultipartForm) {
          // For multipart form data, we'll use -F instead of -d
          // Extract boundary from content type
          const boundaryMatch = contentType.match(/boundary=([^;]+)/);
          if (boundaryMatch) {
            const boundary = boundaryMatch[1];
            const parts = decodedBytes.split('--' + boundary);

            // Process each part
            parts.forEach(part => {
              if (part.trim() && !part.includes('--\r\n')) {
                const contentDispositionMatch = part.match(/Content-Disposition: form-data; name="([^"]+)"(?:; filename="([^"]+)")?/);
                if (contentDispositionMatch) {
                  const name = contentDispositionMatch[1];
                  const filename = contentDispositionMatch[2];

                  if (filename) {
                    // For file uploads, use a placeholder
                    curl += ` \\\n  -F '${name}=@path/to/${filename}'`;
                  } else {
                    // For regular form fields, extract the value
                    const value = part.split('\r\n\r\n')[1]?.trim();
                    if (value) {
                      curl += ` \\\n  -F '${name}=${value}'`;
                    }
                  }
                }
              }
            });
          }
        } else {
          // For non-multipart data, use -d as before
          try {
            const decodedBody = decodeURIComponent(decodedBytes);
            curl += ` \\\n  -d '${decodedBody}'`;
          } catch {
            curl += ` \\\n  -d '${decodedBytes}'`;
          }
        }
      } catch (e) {
        curl += ` \\\n  -d '${$currentRequest.Body}'`;
      }
    }

    return curl;
  };

  const copyCurlCommand = async () => {
    const curl = generateCurlCommand();
    try {
      await navigator.clipboard.writeText(curl);
      toast.success('Curl command copied to clipboard');
    } catch (error) {
      toast.error('Failed to copy curl command');
    }
  };
</script>

<svelte:head>
  {@html atomonelight}
</svelte:head>

<div class="flex-1 overflow-y-auto bg-gray-50 dark:bg-gray-900">
  {#if !$currentRequest}
    <div class="flex flex-col items-center justify-center h-full p-6 text-center">
      <div class="rounded-full bg-gray-100 p-3 mb-4">
        <ArrowLeft class="w-6 h-6 text-gray-400" />
      </div>
      <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-1">Select a request</h3>
      <p class="text-sm text-gray-500 dark:text-gray-400">
        Choose a request from the sidebar to view details
      </p>
    </div>
  {:else}
    <div class="p-6">
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border dark:border-gray-700 mb-6">
        <div class="p-6 border-b dark:border-gray-700">
          <div class="flex justify-between items-start">
            <div class="space-y-1">
              <div class="flex items-center gap-3">
                <HttpBadge method={$currentRequest.Method} />
                {#if $currentRequest.IsReplayed}
                  <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                    <RefreshCw class="mr-1 w-3 h-3" />
                    Replayed
                  </span>
                {/if}
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {$currentRequest.ResponseStatusCode >= 400 ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200' : $currentRequest.ResponseStatusCode >= 300 ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200' : 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'}">
                  {$currentRequest.ResponseStatusCode}
                </span>
              </div>
              <h2 class="text-xl font-medium text-gray-900 dark:text-white break-all">
                {$currentRequest.Url}
              </h2>
              <div class="flex items-center text-sm text-gray-500 dark:text-gray-400">
                <Clock class="w-4 h-4 mr-1" />
                <span>{convertDateToHumanReadable($currentRequest.LoggedAt)}</span>
              </div>
            </div>
            <div class="flex items-center gap-2">
              {#if $currentRequest.ParentID !== ""}
                <Button variant="outline" size="sm" on:click={viewParent} class="flex items-center gap-2">
                  <ArrowUpRight class="w-4 h-4" />
                  View parent
                </Button>
              {/if}
              <Button
                variant="default"
                size="sm"
                disabled={replaying}
                on:click={replayRequest}
                class="flex items-center gap-2"
              >
                {#if replaying}
                  <Loader class="w-4 h-4 animate-spin" />
                  Replaying...
                {:else}
                  <Play class="w-4 h-4" />
                  Replay
                {/if}
              </Button>
              <Button
                variant="outline"
                size="sm"
                on:click={copyCurlCommand}
                class="flex items-center gap-2"
              >
                <Copy class="w-4 h-4" />
                Copy as cURL
              </Button>
            </div>
          </div>
        </div>

        <div class="w-full">
          <div class="border-b dark:border-gray-700 px-6">
            <div class="flex">
              <button
                class="px-4 py-3 relative font-medium text-sm transition-all focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 {activeTab === 'request' ? 'text-primary border-b-2 border-primary' : ''}"
                on:click={() => activeTab = "request"}
              >
                Request
              </button>
              <button
                class="px-4 py-3 relative font-medium text-sm transition-all focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 {activeTab === 'response' ? 'text-primary border-b-2 border-primary' : ''}"
                on:click={() => activeTab = "response"}
              >
                Response
              </button>
            </div>
          </div>

          {#if activeTab === "request"}
            <div class="p-6 space-y-6">
              <div>
                <h3 class="text-lg font-medium text-gray-800 dark:text-gray-200 mb-3">
                  Headers
                </h3>
                <div class="rounded-md bg-[#FAFAFA] border dark:bg-gray-700 dark:border-gray-600 overflow-hidden">
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
                <h3 class="text-lg font-medium text-gray-800 dark:text-gray-200 mb-3">
                  Body
                </h3>
                <div class="rounded-md bg-[#FAFAFA] border dark:bg-gray-700 dark:border-gray-600 overflow-hidden">
                  <RenderContent type="request" />
                </div>
              </div>
            </div>
          {:else if activeTab === "response"}
            <div class="p-6 space-y-6">
              <div>
                <h3 class="text-lg font-medium text-gray-800 dark:text-gray-200 mb-3">
                  Headers
                </h3>
                <div class="rounded-md bg-[#FAFAFA] border dark:bg-gray-700 dark:border-gray-600 overflow-hidden">
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
                <h3 class="text-lg font-medium text-gray-800 dark:text-gray-200 mb-3">
                  Body
                </h3>
                <div class="rounded-md bg-[#FAFAFA] border dark:bg-gray-700 dark:border-gray-600 overflow-hidden">
                  <RenderContent type="response" />
                </div>
              </div>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  :global(.hljs) {
    font-size: 14px !important;
    padding: 16px !important;
    background: transparent !important;
  }
</style>
