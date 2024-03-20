<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  // @ts-ignore
  import HttpStatus from "http-status-codes";
  import type { Request } from "$lib/types";
  import { currentRequest } from "$lib/store";
  import RequestDetails from "./RequestDetails.svelte";
  import { Link } from "svelte-routing";

  export let id: string;

  const [subdomain, localport] = id.split("-");

  let requests: Request[] = [];
  let filteredRequests: Request[] = [];
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
      return request.Url.includes(search);
    });
    if (filteredRequests.length === 0) {
      currentRequest.set(null);
    } else {
      currentRequest.set(filteredRequests[0]);
    }
  };

  let interval: number | undefined;

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
    class="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800 bg-white dark:bg-gray-800"
  >
    <Link
      to="/"
      class="text-3xl font-semibold text-gray-800 dark:text-gray-200"
    >
      Portr Inspector 🚨
    </Link>
    <div class="flex items-center space-x-4">
      <input
        class="flex h-10 rounded-md border bg-background px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50 w-64"
        placeholder="Filter requests"
        bind:value={search}
        on:input={(e) => filterRequestsBasedOnUrl()}
      />
    </div>
  </header>
  <main class="flex flex-1 overflow-hidden">
    <div
      class="w-80 border-r overflow-y-auto dark:border-gray-800 bg-white dark:bg-gray-800"
    >
      <div class="p-4 space-y-2">
        {#each filteredRequests as request, i (i)}
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <!-- svelte-ignore a11y-no-static-element-interactions -->
          <div
            class="p-4 rounded-md bg-gray-100 {$currentRequest?.ID ===
            request.ID
              ? 'border border-gray-950'
              : ''} dark:bg-gray-700 m-1 hover:cursor-pointer"
            on:click={() => setCurrentRequest(request)}
          >
            <div
              class="font-medium text-gray-800 dark:text-gray-200 flex justify-between"
            >
              <span>{request.Method}</span>
              <span class="font-light text-sm overflow-clip"
                ><code>{request.Url}</code></span
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
    <RequestDetails />
  </main>
</div>
