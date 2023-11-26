<script lang="ts">
  import {
    Settings,
    Users,
    Home,
    ArrowUpDown,
    User,
    MoreVertical,
  } from "lucide-svelte";

  // @ts-expect-error
  import { Router, Route, navigate } from "svelte-routing";
  import SettingsPage from "./settings.svelte";
  import Connections from "./connections.svelte";
  import Notfound from "./notfound.svelte";
  import UsersPage from "./users.svelte";
  import { getLoggedInUser } from "../../lib/services/user";
  import { onMount } from "svelte";
  import { currentUser } from "$lib/store";
  import Profile from "./profile.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
  import Sidebarlink from "$lib/components/sidebarlink.svelte";
  import Overview from "./overview.svelte";

  export let url = "";

  const logout = async () => {
    const res = await fetch("/api/user/me/logout", {
      method: "POST",
    });

    if (res.ok) {
      navigate("/");
    }
  };

  onMount(async () => {
    $currentUser = await getLoggedInUser();
  });
</script>

<div class="flex">
  <aside
    class="sticky top-0 flex flex-col w-64 h-screen px-5 py-8 overflow-y-auto bg-white border-r rtl:border-r-0 rtl:border-l dark:bg-gray-900 dark:border-gray-700"
  >
    <a href="/">
      <img class="w-auto h-7" src="/static/favicon.svg" alt="" />
    </a>

    <div class="flex flex-col justify-between flex-1 mt-6">
      <nav class="flex-1 -mx-3 space-y-2">
        <Sidebarlink url="/overview">
          <Home strokeWidth={1.5} class="h-4 w-4" />
          <span class="mx-2 text-sm">Overview</span>
        </Sidebarlink>

        <Sidebarlink url="/connections">
          <ArrowUpDown strokeWidth={1.5} class="h-4 w-4" />
          <span class="mx-2 text-sm">Connections</span>
        </Sidebarlink>

        <Sidebarlink url="/users">
          <Users strokeWidth={1.5} class="h-4 w-4" />
          <span class="mx-2 text-sm">Users</span>
        </Sidebarlink>

        <Sidebarlink url="/profile">
          <User strokeWidth={1.5} class="h-4 w-4" />
          <span class="mx-2 text-sm">Profile</span>
        </Sidebarlink>
        <Sidebarlink url="/settings">
          <Settings strokeWidth={1.5} class="h-4 w-4" />
          <span class="mx-2 text-sm">Settings</span>
        </Sidebarlink>
      </nav>

      <div class="mt-6 -mx-3">
        <DropdownMenu.Root>
          <DropdownMenu.Trigger asChild let:builder>
            <Button
              builders={[builder]}
              variant="ghost"
              class="space-x-1 justify-between w-full text-left"
            >
              <div class="flex items-center space-x-1">
                <img
                  class="object-cover rounded-full h-7 w-7"
                  src={$currentUser?.avatarUrl}
                  alt="avatar"
                />
                <span
                  class="text-sm font-medium text-gray-700 dark:text-gray-200 overflow-clip"
                  >{$currentUser?.FirstName
                    ? `${$currentUser?.FirstName} ${$currentUser?.LastName}`
                    : $currentUser?.Email}</span
                >
              </div>
              <MoreVertical strokeWidth={1.5} class="h-4 w-4" />
            </Button>
          </DropdownMenu.Trigger>
          <DropdownMenu.Content class="w-52">
            <DropdownMenu.Label>My Account</DropdownMenu.Label>
            <DropdownMenu.Separator />
            <DropdownMenu.Item on:click={logout} class="hover:cursor-pointer"
              >Logout</DropdownMenu.Item
            >
          </DropdownMenu.Content>
        </DropdownMenu.Root>
      </div>
    </div>
  </aside>
  <aside class="w-full">
    <div class="container mx-auto py-16 w-full lg:w-3/4">
      <Router {url}>
        <Route path="/overview"><Overview /></Route>
        <Route path="/connections"><Connections /></Route>
        <Route path="/settings"><SettingsPage /></Route>
        <Route path="/profile"><Profile /></Route>
        <Route path="/users"><UsersPage /></Route>
        <Route path="*"><Notfound /></Route>
      </Router>
    </div>
  </aside>
</div>
