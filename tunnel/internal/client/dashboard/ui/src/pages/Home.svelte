<script lang="ts">
  import * as Table from "$lib/components/ui/table";
  import type { Tunnel } from "$lib/types";
  import { onMount } from "svelte";
  import { navigate } from "svelte-routing";

  let tunnels: Tunnel[] = [];

  const getRecentTunnels = async () => {
    const response = await fetch("/api/tunnels");
    tunnels = (await response.json())["tunnels"];
  };

  const gotoTunnel = (tunnel: Tunnel) => {
    navigate(`/${tunnel.Subdomain}-${tunnel.Localport}`);
  };

  onMount(() => {
    getRecentTunnels();
  });
</script>

<div
  class="place-items-center w-1/4 border mx-auto h-[750px] items-center overflow-auto my-12 rounded-lg"
>
  <Table.Root>
    <Table.Header>
      <Table.Row>
        <Table.Head>Subdomain</Table.Head>
        <Table.Head class="text-right">Port</Table.Head>
      </Table.Row>
    </Table.Header>
    <Table.Body>
      {#each tunnels as tunnel, i (i)}
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
