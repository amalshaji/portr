<script lang="ts">
  import DataTable from "$lib/components/data-table.svelte";
  // @ts-expect-error
  import { createTable } from "svelte-headless-table";
  import { invites, invitesLoading } from "$lib/store";
  import { onMount } from "svelte";
  import { Button } from "$lib/components/ui/button";
  import * as AlertDialog from "$lib/components/ui/alert-dialog";
  import type { Invite, User } from "$lib/types";

  const getInvites = async () => {
    invitesLoading.set(true);
    try {
      const response = await fetch("/api/invite");
      invites.set(await response.json());
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
