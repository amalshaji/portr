<script lang="ts">
  import * as Table from "$lib/components/ui/table";
  import type { Tunnel } from "$lib/types";
  import { onMount } from "svelte";
  import { navigate } from "svelte-routing";

  let tunnels: Tunnel[] = [];
  let searchQuery = "";

  const getRecentTunnels = async () => {
    const response = await fetch("/api/tunnels");
    tunnels = (await response.json())["tunnels"];
  };

  const gotoTunnel = (tunnel: Tunnel) => {
    navigate(`/${tunnel.Subdomain}-${tunnel.Localport}`);
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

<div class="mt-4 mx-auto grid place-items-center">
  <input
    type="text"
    placeholder="Search by subdomain or port..."
    bind:value={searchQuery}
    class="w-1/4 mx-auto px-3 py-2 border rounded-md focus:outline-none"
  />
</div>

<div
  class="place-items-center w-1/4 border mx-auto h-[750px] items-center overflow-auto mt-4 mb-12 rounded-lg"
>

  <Table.Root>
    <Table.Header>
      <Table.Row>
        <Table.Head>Subdomain</Table.Head>
        <Table.Head class="text-right">Port</Table.Head>
      </Table.Row>
    </Table.Header>
    <Table.Body>
      {#each filteredTunnels as tunnel, i (i)}
        <Table.Row
          on:click={() => gotoTunnel(tunnel)}
          class="hover:cursor-pointer hover:bg-gray-100"
        >
          <Table.Cell class="">{tunnel.Subdomain}</Table.Cell>
          <Table.Cell class="text-right">{tunnel.Localport}</Table.Cell>
        </Table.Row>
      {/each}
    </Table.Body>
  </Table.Root>
</div>
