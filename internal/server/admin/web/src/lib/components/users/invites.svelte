<script lang="ts">
  import DataTable from "$lib/components/data-table.svelte";
  // @ts-expect-error
  import { createTable } from "svelte-headless-table";
  import { invites, invitesLoading } from "$lib/store";
  import { getContext, onMount } from "svelte";
  import type { Invite } from "$lib/types";

  let team = getContext("team");

  const getInvites = async () => {
    invitesLoading.set(true);
    try {
      const response = await fetch(`/api/${team}/invite`);
      invites.set((await response.json()) || []);
    } catch (err) {
      console.error(err);
    } finally {
      invitesLoading.set(false);
    }
  };

  const table = createTable(invites);

  const columns = table.createColumns([
    // table.column({
    //   accessor: "ID",
    //   header: "ID",
    // }),
    table.column({
      accessor: "Email",
      header: "Email",
    }),
    table.column({
      accessor: "Role",
      header: "Role",
    }),
    table.column({
      accessor: "Status",
      header: "Status",
    }),
    table.column({
      accessor: (item: Invite) => {
        const { Email, FirstName, LastName } = item.InvitedByUser;
        if (FirstName) {
          return `${FirstName} ${LastName}`;
        }
        return Email;
      },
      header: "Invited by",
    }),
  ]);

  onMount(() => {
    getInvites();
  });
</script>

<DataTable {table} {columns} isLoading={$invitesLoading} />
