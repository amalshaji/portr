<script lang="ts">
  import DataTable from "$lib/components/data-table.svelte";
  // @ts-expect-error
  import { createTable, createRender } from "svelte-headless-table";
  import { users, usersLoading } from "$lib/store";
  import { getContext, onMount } from "svelte";
  import Avatar from "./avatar.svelte";
  import type { TeamUser } from "$lib/types";
  import UserEmail from "./user-email.svelte";
  import { updateQueryParam } from "$lib/utils";
  import { writable } from "svelte/store";
  import Pagination from "../Pagination.svelte";
  import InviteUser from "$lib/components/users/invite-user.svelte";
  import { currentUser } from "$lib/store";
  let addMemberModalOpen = false;
  import { Button } from "$lib/components/ui/button";

  const urlParams = new URLSearchParams(window.location.search);

  let pageNo = writable(1);
  let pageNoStr = urlParams.get("page") || "1";
  pageNo.set(parseInt(pageNoStr, 10) || 1);

  $: updateQueryParam(urlParams, "page", $pageNo.toString());
  $: getUsers($pageNo.toString());

  let team = getContext("team") as string;

  let totalItems = 0;

  const getUsers = async (pageNo: string = "1") => {
    usersLoading.set(true);
    try {
      const response = await fetch(
        `/api/v1/team/users?page=${pageNo}&page_size=10`,
        {
          headers: {
            "x-team-slug": team,
          },
        }
      );
      const resp = await response.json();
      users.set(resp["data"]);
      totalItems = resp["count"];
    } catch (err) {
      console.error(err);
    } finally {
      usersLoading.set(false);
    }
  };

  const table = createTable(users);

  const columns = table.createColumns([
    table.column({
      header: "Email",
      accessor: (item: TeamUser) => item,
      cell: (item: any) =>
        createRender(UserEmail, {
          email: item.value.user.email,
          is_superuser: item.value.user.is_superuser,
        }),
    }),

    table.column({
      header: "Role",
      accessor: (item: TeamUser) => item.role,
    }),
    table.column({
      accessor: (item: TeamUser) => item,
      header: "Avatar",
      cell: (item: any) =>
        createRender(Avatar, {
          url: item.value.user.github_user?.github_avatar_url,
          fallback: item.value.user.email,
        }),
    }),
  ]);

  onMount(() => {
    getUsers();
  });
</script>

<InviteUser bind:open={addMemberModalOpen} />

<div class="flex w-full justify-between items-center py-4">
  <div>
    <Button
      on:click={() => (addMemberModalOpen = !addMemberModalOpen)}
      disabled={$currentUser?.role === "member"}>Add member</Button
    >
  </div>
  <div>
    <Pagination count={totalItems} perPage={10} currentPage={pageNo} />
  </div>
</div>

<DataTable {table} {columns} isLoading={$usersLoading} />
