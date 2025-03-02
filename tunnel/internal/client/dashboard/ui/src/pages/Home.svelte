<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { Input } from "$lib/components/ui/input";
  import * as Table from "$lib/components/ui/table";
  import type { Tunnel } from "$lib/types";
  import { RefreshCw, Search } from "lucide-svelte";
  import { onMount } from "svelte";
  import { navigate } from "svelte-routing";

  let tunnels: Tunnel[] = [];
  let searchQuery = "";
  let loading = true;

  const getRecentTunnels = async () => {
    loading = true;
    try {
      const response = await fetch("/api/tunnels");
      tunnels = (await response.json())["tunnels"];
    } catch (error) {
      console.error("Failed to fetch tunnels:", error);
    } finally {
      loading = false;
    }
  };

  const gotoTunnel = (tunnel: Tunnel) => {
    navigate(`/${tunnel.Subdomain}-${tunnel.Localport}`);
  };

  const formatDate = (dateString: string) => {
    if (!dateString) return "";
    const date = new Date(dateString);
    return date.toLocaleDateString();
  };

  // Handle search input with debounce
  let searchTimeout: number | null = null;
  const handleSearchInput = (event: Event) => {
    if (searchTimeout) {
      clearTimeout(searchTimeout);
    }

    searchTimeout = setTimeout(() => {
      searchQuery = (event.target as HTMLInputElement).value;
    }, 300) as unknown as number;
  };

  $: filteredTunnels = tunnels.filter((tunnel) => {
    const query = searchQuery.toLowerCase();
    return (
      tunnel.Subdomain.toLowerCase().includes(query) ||
      tunnel.Localport.toString().includes(query)
    );
  });

  onMount(() => {
    getRecentTunnels();
  });
</script>

<div class="container mx-auto px-4 py-8 max-w-6xl">
  <header class="mb-8">
    <div class="flex flex-col md:flex-row md:items-center md:justify-between">
      <div>
        <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Portr Dashboard</h1>
        <p class="mt-1 text-gray-600 dark:text-gray-400">Manage and inspect your tunnels</p>
      </div>
      <Button variant="outline" class="mt-4 md:mt-0 flex items-center gap-2" on:click={getRecentTunnels}>
        <RefreshCw class="h-4 w-4" />
        Refresh
      </Button>
    </div>
  </header>

  <Card.Root class_list="mb-8">
    <Card.Header>
      <Card.Title>Search Tunnels</Card.Title>
      <Card.Description>Find tunnels by subdomain or port</Card.Description>
    </Card.Header>
    <Card.Content>
      <div class="relative">
        <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
        <Input
          type="text"
          placeholder="Search by subdomain or port..."
          value={searchQuery}
          on:input={handleSearchInput}
          class="pl-10 w-full"
        />
      </div>
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header>
      <Card.Title>Your Tunnels</Card.Title>
      <Card.Description>
        {filteredTunnels.length} {filteredTunnels.length === 1 ? 'tunnel' : 'tunnels'} found
      </Card.Description>
    </Card.Header>
    <Card.Content>
      {#if loading}
        <div class="flex justify-center items-center py-16">
          <RefreshCw class="h-8 w-8 animate-spin text-gray-400" />
        </div>
      {:else if filteredTunnels.length === 0}
        <div class="text-center py-20 text-gray-500">
          {searchQuery ? 'No tunnels match your search' : 'No tunnels found'}
        </div>
      {:else}
        <div class="rounded-md border">
          <Table.Root>
            <Table.Header>
              <Table.Row>
                <Table.Head class="font-medium">Subdomain</Table.Head>
                <Table.Head class="font-medium text-right">Port</Table.Head>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {#each filteredTunnels as tunnel, i (tunnel.Subdomain + tunnel.Localport)}
                <Table.Row
                  class="hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer"
                  on:click={() => gotoTunnel(tunnel)}
                >
                  <Table.Cell class="font-medium">{tunnel.Subdomain}</Table.Cell>
                  <Table.Cell class="text-right">{tunnel.Localport}</Table.Cell>
                </Table.Row>
              {/each}
            </Table.Body>
          </Table.Root>
        </div>
      {/if}
    </Card.Content>
  </Card.Root>
</div>
