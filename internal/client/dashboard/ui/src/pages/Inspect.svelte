<script lang="ts">
  import { onMount } from "svelte";
  import HttpStatus from "http-status-codes";
  import type { Request } from "$lib/types";
  import { currentRequest } from "$lib/store";
  import RequestDetails from "./RequestDetails.svelte";

  export let id: string;

  const [subdomain, localport] = id.split("-");

  let requests: Request[] = [];
  let filteredRequests: Request[] = [];

  const getRequests = async () => {
    const response = await fetch(`/api/tunnels/${subdomain}/${localport}`);
    requests = (await response.json())["requests"];
    filteredRequests = requests;
    if (!$currentRequest) {
      currentRequest.set(requests[0]);
    }
  };

  const setCurrentRequest = (request: Request) => {
    currentRequest.set(request);
  };

  const filterRequestsBasedOnUrl = (search: string) => {
    console.log(search);
    filteredRequests = requests.filter((request) => {
      return request.Url.includes(search);
    });
  };

  onMount(() => {
    getRequests();
  });
</script>

<div class="flex flex-col h-screen bg-gray-50 dark:bg-gray-900">
  <header
    class="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800 bg-white dark:bg-gray-800"
  >
    <h1 class="text-3xl font-semibold text-gray-800 dark:text-gray-200">
      Portr HTTP Inspector
    </h1>
    <div class="flex items-center space-x-4">
      <input
        class="flex h-10 rounded-md border bg-background px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50 w-64"
        placeholder="Filter requests"
        on:input={(e) => filterRequestsBasedOnUrl(e.target.value)}
      />
    </div>
  </header>
  <main class="flex flex-1 overflow-hidden">
    <div
      class="w-80 border-r overflow-y-auto dark:border-gray-800 bg-white dark:bg-gray-800"
    >
      <div class="p-4 space-y-2">
        {#each filteredRequests as request, i (i)}
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
            <div class="text-sm text-gray-500 dark:text-gray-400">
              {request.ResponseStatus}
              {HttpStatus.getStatusText(request.ResponseStatus)}
            </div>
          </div>
        {/each}
      </div>
    </div>
    <RequestDetails />
  </main>
</div>
