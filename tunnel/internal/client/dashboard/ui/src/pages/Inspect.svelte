<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  // @ts-ignore
  import HttpBadge from "$lib/components/HttpBadge.svelte";
  import InspectorIcon from "$lib/components/InspectorIcon.svelte";
  import { currentRequest } from "$lib/store";
  import type { Request } from "$lib/types";
  import HttpStatus from "http-status-codes";
  import { RefreshCw } from "lucide-svelte";
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

  const getRequests = async () => {
    const response = await fetch(`/api/tunnels/${subdomain}/${localport}`);
    requests = (await response.json())["requests"];

    console.log(`Logging ${requests.length} requests`);

    filteredRequests = requests;
    if (search) {
      filterRequestsBasedOnUrl();
    }

    if (!$currentRequest) {
      currentRequest.set(requests[0]);
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
      currentRequest.set(filteredRequests[0]);
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
    class="flex items-center justify-between px-6 py-2 border-b dark:border-gray-800 bg-white dark:bg-gray-800"
  >
    <Link to="/" class="flex items-center gap-2">
      <InspectorIcon /> <span class="text-lg">Portr Inspector</span>
    </Link>
    <div class="flex items-center space-x-4">
      {#if filterRequestError}
        <div class="text-red-500 text-sm">{filterRequestError}</div>
      {/if}
      <input
        class="flex h-10 rounded-md border outline-none px-3 py-1 text-sm w-64"
        placeholder="Filter URL"
        bind:value={search}
        on:input={(e) => filterRequestsBasedOnUrl()}
      />
    </div>
  </header>
  <main class="flex flex-1 overflow-hidden">
    <div
      class="w-80 border-r overflow-y-auto dark:border-gray-800 bg-white dark:bg-gray-800"
    >
      <div>
        {#each filteredRequests as request, i (i)}
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <div
            class="p-4 space-y-1 border-b transition-all hover:bg-accent {$currentRequest?.ID ===
            request.ID
              ? 'bg-[#F4F4F5]'
              : 'border-muted'} dark:bg-gray-700 hover:cursor-pointer"
            on:click={() => setCurrentRequest(request)}
          >
            <div
              class="text-sm text-gray-800 dark:text-gray-200 flex justify-between items-center text-clip"
            >
              <span class="flex items-center gap-2">
                <HttpBadge method={request.Method} />
                {#if request.IsReplayed}
                  <RefreshCw class="w-3 h-3" />
                {/if}
              </span>
              <span class="overflow-clip h-6 w-40 text-right"
                >{request.Url}</span
              >
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {request.ResponseStatusCode}
              {HttpStatus.getStatusText(request.ResponseStatusCode)}
            </div>
          </div>
        {/each}
      </div>
    </div>
    <RequestDetails {viewParent} />
  </main>
</div>
