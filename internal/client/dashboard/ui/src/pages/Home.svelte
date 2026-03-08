<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import { Checkbox } from "$lib/components/ui/checkbox";
  import * as Dialog from "$lib/components/ui/dialog";
  import * as Card from "$lib/components/ui/card";
  import { Input } from "$lib/components/ui/input";
  import * as Table from "$lib/components/ui/table";
  import type { Tunnel } from "$lib/types";
  import { RefreshCw, Search, Trash2 } from "lucide-svelte";
  import { onMount } from "svelte";
  import { navigate } from "svelte-routing";
  import { toast } from "svelte-sonner";

  let tunnels: Tunnel[] = [];
  let filteredTunnels: Tunnel[] = [];
  let searchQuery = "";
  let loading = true;
  let selectedTunnelKeys = new Set<string>();
  let deleting = false;
  let deleteDialogOpen = false;

  const getTunnelKey = (tunnel: Tunnel) => {
    return `${tunnel.Subdomain}:${tunnel.Localport}`;
  };

  const parseTunnelKey = (key: string) => {
    const separatorIndex = key.lastIndexOf(":");
    return {
      subdomain: key.substring(0, separatorIndex),
      localport: key.substring(separatorIndex + 1),
    };
  };

  const syncSelectedTunnelKeys = () => {
    const tunnelKeys = new Set(tunnels.map((tunnel) => getTunnelKey(tunnel)));
    selectedTunnelKeys = new Set(
      Array.from(selectedTunnelKeys).filter((key) => tunnelKeys.has(key)),
    );
  };

  const getRecentTunnels = async () => {
    loading = true;
    try {
      const response = await fetch("/api/tunnels");
      tunnels = (await response.json())["tunnels"];
      syncSelectedTunnelKeys();
    } catch (error) {
      console.error("Failed to fetch tunnels:", error);
    } finally {
      loading = false;
    }
  };

  const gotoTunnel = (tunnel: Tunnel) => {
    navigate(`/${tunnel.Subdomain}-${tunnel.Localport}`);
  };

  const isTunnelSelected = (tunnel: Tunnel) => {
    return selectedTunnelKeys.has(getTunnelKey(tunnel));
  };

  const toggleTunnelSelection = (tunnel: Tunnel, checked: boolean) => {
    const tunnelKey = getTunnelKey(tunnel);
    const nextSelectedTunnelKeys = new Set(selectedTunnelKeys);
    if (checked) {
      nextSelectedTunnelKeys.add(tunnelKey);
    } else {
      nextSelectedTunnelKeys.delete(tunnelKey);
    }
    selectedTunnelKeys = nextSelectedTunnelKeys;
  };

  const onTunnelCheckedChange = (tunnel: Tunnel, checked: boolean | "indeterminate") => {
    toggleTunnelSelection(tunnel, checked === true);
  };

  const onTunnelRowClick = (tunnel: Tunnel, event: MouseEvent) => {
    const target = event.target as HTMLElement;
    if (target.closest("[data-tunnel-selector]")) {
      return;
    }
    gotoTunnel(tunnel);
  };

  const openDeleteDialog = () => {
    if (selectedTunnelKeys.size === 0 || deleting) {
      return;
    }
    deleteDialogOpen = true;
  };

  const deleteSelectedTunnels = async () => {
    if (selectedTunnelKeys.size === 0) {
      return;
    }
    const selection = Array.from(selectedTunnelKeys);

    deleting = true;
    try {
      const results = await Promise.allSettled(
        selection.map(async (key) => {
          const { subdomain, localport } = parseTunnelKey(key);
          const response = await fetch(`/api/tunnels/${encodeURIComponent(subdomain)}/${localport}`, {
            method: "DELETE",
          });

          if (!response.ok) {
            const data = await response.json().catch(() => null);
            throw new Error(data?.message || "Failed to delete tunnel logs");
          }

          const data = await response.json();
          return Number(data?.deleted_count || 0);
        }),
      );

      const successful = results.filter((result) => result.status === "fulfilled");
      const failed = results.filter((result) => result.status === "rejected");
      const deletedCount = successful.reduce((total, result) => {
        if (result.status !== "fulfilled") return total;
        return total + result.value;
      }, 0);

      if (failed.length > 0) {
        toast.error(`Deleted logs for ${successful.length} tunnel(s). Failed for ${failed.length}.`);
      } else {
        toast.success(`Deleted ${deletedCount} log${deletedCount === 1 ? "" : "s"}.`);
      }

      selectedTunnelKeys = new Set();
      await getRecentTunnels();
      deleteDialogOpen = false;
    } catch (error) {
      toast.error("Failed to delete selected tunnel logs");
    } finally {
      deleting = false;
    }
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
      <div class="mt-4 md:mt-0 flex items-center gap-2">
        <Button
          variant="destructive"
          class="flex items-center gap-2"
          on:click={openDeleteDialog}
          disabled={selectedTunnelKeys.size === 0 || deleting}
        >
          {#if deleting}
            <RefreshCw class="h-4 w-4 animate-spin" />
            Deleting...
          {:else}
            <Trash2 class="h-4 w-4" />
            Delete selected
          {/if}
        </Button>
        <Button variant="outline" class="flex items-center gap-2" on:click={getRecentTunnels}>
          <RefreshCw class="h-4 w-4" />
          Refresh
        </Button>
      </div>
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
        · {selectedTunnelKeys.size} selected
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
                <Table.Head class="font-medium w-12">Select</Table.Head>
                <Table.Head class="font-medium">Subdomain</Table.Head>
                <Table.Head class="font-medium text-right">Port</Table.Head>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {#each filteredTunnels as tunnel, i (tunnel.Subdomain + tunnel.Localport)}
                <Table.Row
                  class="hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer {isTunnelSelected(tunnel) ? 'bg-gray-100 dark:bg-gray-800/70' : ''}"
                  on:click={(event) => onTunnelRowClick(tunnel, event)}
                >
                  <Table.Cell>
                    <div data-tunnel-selector>
                      <Checkbox
                        checked={isTunnelSelected(tunnel)}
                        onCheckedChange={(checked) => onTunnelCheckedChange(tunnel, checked)}
                      />
                    </div>
                  </Table.Cell>
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

<Dialog.Root bind:open={deleteDialogOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Delete Tunnel Logs</Dialog.Title>
      <Dialog.Description>
        Delete all connection logs for {selectedTunnelKeys.size} selected {selectedTunnelKeys.size === 1 ? "tunnel" : "tunnels"}.
        This action cannot be undone.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Button
        variant="outline"
        on:click={() => (deleteDialogOpen = false)}
        disabled={deleting}
      >
        Cancel
      </Button>
      <Button
        variant="destructive"
        on:click={deleteSelectedTunnels}
        disabled={deleting || selectedTunnelKeys.size === 0}
        class="flex items-center gap-2"
      >
        {#if deleting}
          <RefreshCw class="h-4 w-4 animate-spin" />
          Deleting...
        {:else}
          <Trash2 class="h-4 w-4" />
          Confirm Delete
        {/if}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
