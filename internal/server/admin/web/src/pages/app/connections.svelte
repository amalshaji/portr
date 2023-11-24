<script lang="ts">
  import DataTable from "../../lib/components/data-table.svelte";
  import { onMount } from "svelte";
  import { createTable, Render, Subscribe } from "svelte-headless-table";
  import { addSortBy } from "svelte-headless-table/plugins";
  import { readable } from "svelte/store";
  import * as Table from "$lib/components/ui/table";
  import { humanizeTimeMs } from "$lib/humanize";
  import DataTableSkeleton from "$lib/components/data-table-skeleton.svelte";
  import { Checkbox } from "$lib/components/ui/checkbox";
  import Label from "$lib/components/ui/label/label.svelte";
  import { connections, connectionsLoading } from "$lib/store";
  let checked = false;

  let connectionsType = "recent";

  $: if (checked) {
    getConnections("active");
  } else {
    getConnections();
  }

  let loadingConnection = true;

  const getConnections = async (type: string = "") => {
    $connectionsLoading = true;
    try {
      const response = await fetch(`/api/connections?type=${type}`);
      $connections = await response.json();
    } catch (err) {
      console.error(err);
    } finally {
      $connectionsLoading = false;
    }
  };

  let table, columns;

  table = createTable(connections);

  columns = table.createColumns([
    table.column({
      accessor: "ID",
      header: "ID",
    }),
    table.column({
      accessor: "Subdomain",
      header: "Subdomain",
    }),
    table.column({
      accessor: ({ ClosedAt }) => (ClosedAt === null ? "Active" : "Inactive"),
      header: "Status",
    }),
    table.column({
      header: "Created At",
      accessor: ({ CreatedAt }) => new Date(CreatedAt).toLocaleString("en-US"),
    }),
    table.column({
      accessor: (item) => {
        const { CreatedAt, ClosedAt } = item;
        if (ClosedAt === null) {
          return "-";
        }
        const createdAt = new Date(CreatedAt);
        const closedAt = new Date(ClosedAt);
        const diff = closedAt.getTime() - createdAt.getTime();
        return humanizeTimeMs(diff);
      },
      header: "Duration",
    }),
    table.column({
      accessor: ({ User }) => {
        const { Email, FirstName, LastName } = User;
        if (FirstName && LastName) {
          return `${FirstName} ${LastName}`;
        }
        return Email;
      },
      header: "User",
    }),
  ]);

  onMount(() => {
    getConnections(connectionsType);
  });
</script>

<div class="container mx-auto py-10">
  <p class="text-2xl py-4">Connections</p>
  <div class="flex items-center space-x-2 my-3">
    <Checkbox id="terms" bind:checked />
    <Label
      for="terms"
      class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
    >
      Show active connections
    </Label>
  </div>

  <DataTable {table} {columns} isLoading={$connectionsLoading} />
</div>
