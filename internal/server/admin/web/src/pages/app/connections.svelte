<script lang="ts">
  import DataTable from "../../lib/components/data-table.svelte";
  import { onMount } from "svelte";
  import { createTable, Render, Subscribe } from "svelte-headless-table";
  import { addSortBy } from "svelte-headless-table/plugins";
  import { readable } from "svelte/store";
  import * as Table from "$lib/components/ui/table";
  import { humanizeTimeMs } from "$lib/humanize";
  import DataTableSkeleton from "$lib/components/data-table-skeleton.svelte";

  let connectionsType = "recent";

  let connections = [];
  let loadingConnection = true;

  const getConnections = async (type: string = "") => {
    try {
      const response = await fetch(`/api/connections?type=${type}`);
      connections = await response.json();
      buildTable(connections);
    } catch (err) {
      console.error(err);
    } finally {
      loadingConnection = false;
    }
  };

  let table, columns;

  const buildTable = (connections) => {
    table = createTable(readable(connections), {
      sort: addSortBy(),
    });

    columns = table.createColumns([
      table.column({
        accessor: "ID",
        header: "ID",
        plugins: {
          sort: {
            disable: true,
          },
        },
      }),
      table.column({
        accessor: "Subdomain",
        header: "Subdomain",
        plugins: {
          sort: {
            disable: true,
          },
        },
      }),
      table.column({
        accessor: ({ ClosedAt }) => (ClosedAt === null ? "Active" : "Inactive"),
        header: "Status",
        plugins: {
          sort: {
            disable: true,
          },
        },
      }),
      table.column({
        header: "Created At",
        accessor: ({ CreatedAt }) =>
          new Date(CreatedAt).toLocaleString("en-US"),
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
        plugins: {
          sort: {
            disable: true,
          },
        },
      }),
    ]);
  };

  onMount(() => {
    getConnections(connectionsType);
  });
</script>

<div class="container mx-auto py-10">
  <p class="text-2xl py-4">Connections</p>
  {#if !loadingConnection}
    <DataTable props={table.createViewModel(columns)} />
  {:else}
    <div class="flex justify-center">
      <DataTableSkeleton />
    </div>
  {/if}
</div>
