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
  import DateField from "$lib/components/DateField.svelte";
  import Pagination from "$lib/components/Pagination.svelte";
  import { writable } from "svelte/store";
  import { updateQueryParam } from "$lib/utils";

  let checked = false;
  const urlParams = new URLSearchParams(window.location.search);
  let connectionType = urlParams.get("type") || "recent";

  let pageNo = writable(1);
  let pageNoStr = urlParams.get("page") || "1";
  pageNo.set(parseInt(pageNoStr, 10) || 1);

  $: if (checked) {
    if (connectionType === "recent") {
      connectionType = "active";
      pageNo.set(1);
    }
  } else {
    if (connectionType === "active") {
      connectionType = "recent";
      $pageNo = 1;
    }
  }

  $: updateQueryParam(urlParams, "type", connectionType);
  $: updateQueryParam(urlParams, "page", $pageNo.toString());
  $: getConnections(connectionType, $pageNo.toString());

  let team = getContext("team") as string;

  let totalItems = 0;

  const getConnections = async (
    type: string = "recent",
    pageNo: string = "1"
  ) => {
    connectionsLoading.set(true);
    try {
      const response = await fetch(
        `/api/v1/connections/?type=${type}&page=${pageNo}`,
        {
          headers: {
            "x-team-slug": team,
          },
        }
      );
      const responseData = await response.json();
      connections.set(responseData["data"] || []);
      totalItems = responseData.count;
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
      cell: ({ value: { type } }: { value: { type: string } }) =>
        createRender(ConnectionType, { type }),
    }),
    table.column({
      header: "Port",
      accessor: (item: Connection) => {
        const { port } = item;
        return port ? port : "-";
      },
    }),
    table.column({
      header: "Subdomain",
      accessor: (item: Connection) => {
        const { subdomain } = item;
        return subdomain ? subdomain : "-";
      },
    }),
    table.column({
      accessor: (item: Connection) => item,
      header: "Status",
      cell: ({ value: { status } }: { value: { status: string } }) =>
        createRender(ConnectionStatus, { status }),
    }),
    table.column({
      accessor: (item: Connection) => item,
      header: "Created at",
      cell: (item: any) =>
        createRender(DateField, { Date: item.value.created_at }),
    }),
    table.column({
      accessor: (item: Connection) => {
        const { started_at, closed_at, status } = item;
        if (status === "active") {
          return "-";
        }
        const startedAt = new Date(started_at as string);
        const closedAt = new Date(closed_at as string);
        const diff = closedAt.getTime() - startedAt.getTime();
        return humanizeTimeMs(diff);
      },
      header: "Duration",
    }),
    table.column({
      accessor: (item: Connection) => {
        const { email, first_name, last_name } = item.created_by.user;
        if (first_name) {
          return `${first_name} ${last_name}`;
        }
        return email;
      },
      header: "Created by",
    }),
  ]);
</script>

<div class="p-6">
  <div class="flex items-center py-3 justify-between w-full">
    <div class="flex items-center space-x-2">
      <Checkbox id="terms" bind:checked class="rounded-full" />
      <Label
        for="terms"
        class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
      >
        Show active connections
      </Label>
    </div>
    <div>
      <Pagination count={totalItems} perPage={10} currentPage={pageNo} />
    </div>
  </div>

  <DataTable {table} {columns} isLoading={$connectionsLoading} />
</div>
