<script lang="ts">
  import DataTable from "$lib/components/data-table.svelte";
  // @ts-expect-error
  import { createTable, createRender } from "svelte-headless-table";
  import { humanizeTimeMs } from "$lib/humanize";
  import { Checkbox } from "$lib/components/ui/checkbox";
  import Label from "$lib/components/ui/label/label.svelte";
  import { connections, connectionsLoading } from "$lib/store";
  import type { Connection } from "$lib/types";
  import { getContext } from "svelte";
  import ConnectionStatus from "$lib/components/ConnectionStatus.svelte";
  import ConnectionType from "$lib/components/ConnectionType.svelte";

  let checked = false;

  $: if (checked) {
    getConnections("active");
  } else {
    getConnections("recent");
  }

  let team = getContext("team");

  const getConnections = async (type: string = "") => {
    connectionsLoading.set(true);
    try {
      const response = await fetch(`/api/${team}/connection?type=${type}`);
      connections.set((await response.json()) || []);
    } catch (err) {
      console.error(err);
    } finally {
      connectionsLoading.set(false);
    }
  };

  const table = createTable(connections);

  const columns = table.createColumns([
    table.column({
      header: "Type",
      accessor: (item: Connection) => item,
      cell: ({ value: { Type } }: { value: { Type: string } }) =>
        createRender(ConnectionType, { Type }),
    }),
    table.column({
      header: "Port",
      accessor: (item: Connection) => {
        const { Port } = item;
        return Port ? Port : "-";
      },
    }),
    table.column({
      header: "Subdomain",
      accessor: (item: Connection) => {
        const { Subdomain } = item;
        return Subdomain ? Subdomain : "-";
      },
    }),
    table.column({
      accessor: (item: Connection) => item,
      header: "Status",
      cell: ({ value: { Status } }: { value: { Status: string } }) =>
        createRender(ConnectionStatus, { Status }),
    }),
    table.column({
      header: "Created at",
      accessor: ({ CreatedAt }: { CreatedAt: string }) =>
        new Date(CreatedAt).toLocaleString("en-US"),
    }),
    table.column({
      accessor: (item: Connection) => {
        const { StartedAt, ClosedAt, Status } = item;
        if (Status === "active") {
          return "-";
        }
        const startedAt = new Date(StartedAt as string);
        const closedAt = new Date(ClosedAt as string);
        const diff = closedAt.getTime() - startedAt.getTime();
        return humanizeTimeMs(diff);
      },
      header: "Duration",
    }),
    table.column({
      accessor: (item: any) => {
        const { Email, FirstName, LastName } = item;
        if (FirstName) {
          return `${FirstName} ${LastName}`;
        }
        return Email;
      },
      header: "Created by",
    }),
  ]);
</script>

<div class="flex items-center space-x-2 my-6">
  <Checkbox id="terms" bind:checked class="rounded-full" />
  <Label
    for="terms"
    class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
  >
    Show active connections
  </Label>
</div>

<DataTable {table} {columns} isLoading={$connectionsLoading} />
