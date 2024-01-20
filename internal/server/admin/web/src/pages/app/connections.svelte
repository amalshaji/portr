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

  const updateQueryParam = (key: string, value: string) => {
    urlParams.set(key, value);
    const newUrl = `${window.location.pathname}?${urlParams.toString()}`;
    window.history.pushState({}, "", newUrl);
  };

  let checked = false;
  const urlParams = new URLSearchParams(window.location.search);
  let connectionType = urlParams.get("type") || "recent";

  let pageNo = writable(1);
  let pageNoStr = urlParams.get("pageNo") || "1";
  pageNo.set(parseInt(pageNoStr, 10) || 1);

  $: if (checked) {
    connectionType = "active";
    $pageNo = 1;
  } else {
    connectionType = "recent";
    $pageNo = 1;
  }

  $: updateQueryParam("type", connectionType);
  $: updateQueryParam("pageNo", $pageNo.toString());
  $: getConnections(connectionType, $pageNo.toString());

  let team = getContext("team");
  let pagination = {
    pageNo: 1,
    pageSize: 10,
    total: 0,
  };

  const getConnections = async (
    type: string = "recent",
    pageNo: string = "1"
  ) => {
    connectionsLoading.set(true);
    try {
      const response = await fetch(
        `/api/${team}/connection?type=${type}&pageNo=${pageNo}`
      );
      const responseData = await response.json();
      connections.set(responseData["data"] || []);
      pagination = responseData["pagination"];
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
      accessor: (item: Connection) => item,
      header: "Created at",
      cell: ({ value: { CreatedAt } }: { value: { CreatedAt: string } }) =>
        createRender(DateField, { Date: CreatedAt }),
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

<div class="flex items-center my-6 justify-between w-full">
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
    <Pagination
      count={pagination.total}
      perPage={pagination.pageSize}
      currentPage={pageNo}
    />
  </div>
</div>

<DataTable {table} {columns} isLoading={$connectionsLoading} />
