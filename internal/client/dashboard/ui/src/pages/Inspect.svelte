<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  // @ts-ignore
  import HttpBadge from "$lib/components/HttpBadge.svelte";
  import InspectorIcon from "$lib/components/InspectorIcon.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import { currentRequest } from "$lib/store";
  import type { Request } from "$lib/types";
  import HttpStatus from "http-status-codes";
  import { ArrowLeft, Clock, RefreshCw, Search } from "lucide-svelte";
  import { Link } from "svelte-routing";
  import RequestDetails from "./RequestDetails.svelte";

  export let id: string;

  const idLastDashIndex = id.lastIndexOf("-");
  const [subdomain, localport] = [
    id.substring(0, idLastDashIndex),
    id.substring(idLastDashIndex + 1),
  ];

  let requests: Request[] = [];
  let filteredRequests: Request[] = [];
  let filterRequestError: string | null = null;
  let search = "";
  let loading = true;

  const getRequests = async () => {
    loading = true;
    try {
      const response = await fetch(`/api/tunnels/${subdomain}/${localport}`);
      requests = (await response.json())["requests"];
      console.log(`Logging ${requests.length} requests`);

      filteredRequests = requests;
      if (search) {
        filterRequestsBasedOnUrl();
      }

      if ($currentRequest && !requests.some((request) => request.ID === $currentRequest?.ID)) {
        currentRequest.set(requests[0] || null);
      } else if (!$currentRequest && requests.length > 0) {
        currentRequest.set(requests[0]);
      }
    } catch (error) {
      console.error("Failed to fetch requests:", error);
    } finally {
      loading = false;
    }
  };

  const setCurrentRequest = (request: Request) => {
    currentRequest.set(request);
  };

  const filterRequestsBasedOnUrl = () => {
    filteredRequests = requests.filter((request) => {
      return request.Url.toLowerCase().includes(search.toLowerCase().trim());
    });
    if (filteredRequests.length === 0) {
      filteredRequests = requests;
      filterRequestError = "No results found";
    } else {
      if (!$currentRequest) {
        currentRequest.set(filteredRequests[0]);
      }
      filterRequestError = null;
    }
  };

  let interval: number | undefined;

  const viewParent = () => {
    const parentId = $currentRequest?.ParentID;
    // @ts-ignore
    currentRequest.set(requests.find((request) => request.ID === parentId));
  };

  onMount(() => {
    currentRequest.set(null);
    getRequests();
    interval = setInterval(getRequests, 2000);
  });

  onDestroy(() => {
    clearInterval(interval);
  });
</script>

<div class="flex flex-col h-screen bg-gray-50 dark:bg-gray-900">
  <header
    class="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800 bg-white dark:bg-gray-800 shadow-sm"
  >
    <div class="flex items-center gap-4">
      <Link to="/" class="flex items-center gap-2 text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white transition-colors">
        <ArrowLeft class="w-5 h-5" />
        <span class="text-sm font-medium">Back to Dashboard</span>
      </Link>
      <div class="h-6 w-px bg-gray-300 dark:bg-gray-700"></div>
      <div class="flex items-center gap-2">
        <InspectorIcon />
        <div class="flex flex-col">
          <span class="text-lg font-semibold text-gray-900 dark:text-white">Portr Inspector</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{subdomain}:{localport}</span>
        </div>
      </div>
    </div>
    <div class="flex items-center gap-4">
      {#if filterRequestError}
        <div class="text-red-500 text-sm">{filterRequestError}</div>
      {/if}
      <div class="relative">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
        <Input
          class="h-8 pl-10 w-56 text-xs"
          placeholder="Filter URL"
          bind:value={search}
          on:input={(e) => filterRequestsBasedOnUrl()}
        />
      </div>
      <Button variant="outline" size="sm" on:click={getRequests} class="flex items-center gap-2">
        <RefreshCw class="w-4 h-4" />
        Refresh
      </Button>
    </div>
  </header>
  <main class="flex flex-1 overflow-hidden">
    <div
      class="w-96 border-r overflow-y-auto dark:border-gray-800 bg-white dark:bg-gray-800 shadow-sm"
    >
      {#if loading && requests.length === 0}
        <div class="flex justify-center items-center h-full">
          <RefreshCw class="w-8 h-8 animate-spin text-gray-400" />
        </div>
      {:else if filteredRequests.length === 0}
        <div class="flex flex-col items-center justify-center h-full p-6 text-center">
          <div class="rounded-full bg-gray-100 p-3 mb-4">
            <Search class="w-6 h-6 text-gray-400" />
          </div>
          <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-1">No requests found</h3>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {search ? 'Try a different search term' : 'Waiting for requests to arrive'}
          </p>
        </div>
      {:else}
        <div class="divide-y dark:divide-gray-700">
          {#each filteredRequests as request, i (request.ID)}
            <button
              class="w-full px-4 py-3 transition-colors hover:bg-gray-50 dark:hover:bg-gray-700 text-left relative {$currentRequest?.ID === request.ID ? 'bg-gray-100 dark:bg-gray-700' : ''}"
              on:click={() => setCurrentRequest(request)}
            >
              {#if $currentRequest?.ID === request.ID}
                <div class="absolute left-0 top-0 bottom-0 w-1 bg-primary"></div>
              {/if}
              <div class="flex items-center justify-between mb-1.5">
                <div class="flex items-center gap-2">
                  <HttpBadge method={request.Method} />
                  <span class="text-xs px-2 py-0.5 rounded-full {request.ResponseStatusCode >= 400 ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200' : request.ResponseStatusCode >= 300 ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200' : 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'}">
                    {request.ResponseStatusCode} {HttpStatus.getStatusText(request.ResponseStatusCode)}
                  </span>
                  {#if request.IsReplayed}
                    <span class="flex items-center text-xs text-blue-500">
                      <RefreshCw class="w-3 h-3 mr-0.5" />
                      <span class="hidden sm:inline">Replayed</span>
                    </span>
                  {/if}
                </div>
                <div class="flex items-center text-xs text-gray-500 dark:text-gray-400">
                  <Clock class="w-3 h-3 mr-1" />
                  <span>{new Date(request.LoggedAt).toLocaleTimeString()}</span>
                </div>
              </div>
              <div class="text-sm font-medium text-gray-900 dark:text-gray-200 truncate">
                {request.Url}
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>
    <RequestDetails {viewParent} />
  </main>
</div>
