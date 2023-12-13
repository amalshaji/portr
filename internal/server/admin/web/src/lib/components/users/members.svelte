<script lang="ts">
  import DataTable from "$lib/components/data-table.svelte";
  // @ts-expect-error
  import { createTable, createRender } from "svelte-headless-table";
  import { users, usersLoading } from "$lib/store";
  import { getContext, onMount } from "svelte";
  import Avatar from "./avatar.svelte";
  import type { TeamUser, User } from "$lib/types";

  let team = getContext("team");

  const getUsers = async () => {
    usersLoading.set(true);
    try {
      const response = await fetch(`/api/${team}/user`);
      users.set(await response.json());
    } catch (err) {
      console.error(err);
    } finally {
      usersLoading.set(false);
    }
  };

  const table = createTable(users);

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
      accessor: (item: TeamUser) => item,
      header: "Avatar",
      cell: ({
        value: { GithubAvatarUrl, Email },
      }: {
        value: { GithubAvatarUrl: string; Email: string };
      }) => createRender(Avatar, { url: GithubAvatarUrl, fallback: Email }),
    }),
  ]);

  onMount(() => {
    getUsers();
  });
</script>

<DataTable {table} {columns} isLoading={$usersLoading} />
