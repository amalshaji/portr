<script lang="ts">
  import DataTable from "$lib/components/data-table.svelte";
  // @ts-expect-error
  import { createTable } from "svelte-headless-table";
  import { humanizeTimeMs } from "$lib/humanize";
  import { Checkbox } from "$lib/components/ui/checkbox";
  import Label from "$lib/components/ui/label/label.svelte";
  import { connections, connectionsLoading } from "$lib/store";
  import type { Connection, User } from "$lib/types";
  let checked = false;

  let connectionType = "Recent";

  $: if (checked) {
    connectionType = "Active";
    getConnections("active");
  } else {
    connectionType = "Recent";
    getConnections("recent");
  }

  const getConnections = async (type: string = "") => {
    connectionsLoading.set(true);
    try {
      const response = await fetch(`/api/connection?type=${type}`);
      connections.set(await response.json());
    } catch (err) {
      console.error(err);
    } finally {
      connectionsLoading.set(false);
    }
  };

  const table = createTable(connections);

  const columns = table.createColumns([
    table.column({
      accessor: "Subdomain",
      header: "Subdomain",
    }),
    table.column({
      accessor: ({ ClosedAt }: { ClosedAt: string | null }) =>
        ClosedAt === null ? "Active" : "Inactive",
      header: "Status",
    }),
    table.column({
      header: "Created at",
      accessor: ({ CreatedAt }: { CreatedAt: string }) =>
        new Date(CreatedAt).toLocaleString("en-US"),
    }),
    table.column({
      accessor: (item: Connection) => {
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
      accessor: ({ User }: { User: User }) => {
        const { Email, FirstName, LastName } = User;
        if (FirstName) {
          return `${FirstName} ${LastName}`;
        }
        return Email;
      },
      header: "Created by",
    }),
  ]);
</script>

<p class="text-2xl py-4">{connectionType} connections</p>
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
