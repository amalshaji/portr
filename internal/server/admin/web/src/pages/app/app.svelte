<script lang="ts">
  import {
    Settings,
    Users,
    Home,
    ArrowUpDown,
    User,
    MoreVertical,
    BadgePlus,
  } from "lucide-svelte";

  import { Router, Route, navigate, Link } from "svelte-routing";
  import SettingsPage from "./settings.svelte";
  import Connections from "./connections.svelte";
  import Notfound from "./notfound.svelte";
  import UsersPage from "./users.svelte";
  import { onMount } from "svelte";
  import { currentTeamUser, currentUser } from "$lib/store";
  import MyAccount from "./myaccount.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
  import Sidebarlink from "$lib/components/sidebarlink.svelte";
  import Overview from "./overview.svelte";
  import { setContext } from "svelte";
  import TeamSelector from "$lib/components/team-selector.svelte";
  import { LogOut } from "lucide-svelte";
  import NewTeam from "./new-team.svelte";
  import Separator from "$lib/components/ui/separator/separator.svelte";

  export let url = "";
  export let team: string;

  setContext("team", team);

  const logout = async () => {
    const res = await fetch("/api/user/me/logout", {
      method: "POST",
    });

    if (res.ok) {
      navigate("/");
    }
  };

  const getLoggedInTeamUser = async () => {
    const response = await fetch(`/api/${team}/user/me`);
    currentTeamUser.set(await response.json());
  };

  const getLoggedInUser = async () => {
    const response = await fetch(`/api/user/me`);
    currentUser.set(await response.json());
  };

  onMount(async () => {
    getLoggedInUser();
    getLoggedInTeamUser();
  });
</script>

<div class="flex">
  <aside
    class="sticky top-0 flex flex-col w-64 h-screen px-2 py-8 overflow-y-auto border-r rtl:border-r-0 rtl:border-l dark:bg-gray-900 dark:border-gray-700"
    style="background-color: #FCFCFD;"
  >
    <TeamSelector />

    <div class="flex flex-col justify-between flex-1 mt-6 mx-4">
      <nav class="flex-1 -mx-3 space-y-2">
        <Sidebarlink url="/{team}/overview">
          <Home class="h-4 w-4" />
          <span class="mx-2">Overview</span>
        </Sidebarlink>

        <Sidebarlink url="/{team}/connections">
          <ArrowUpDown class="h-4 w-4" />
          <span class="mx-2">Connections</span>
        </Sidebarlink>

        <Sidebarlink url="/{team}/users">
          <Users class="h-4 w-4" />
          <span class="mx-2">Users</span>
        </Sidebarlink>

        <Sidebarlink url="/{team}/my-account">
          <User class="h-4 w-4" />
          <span class="mx-2">My account</span>
        </Sidebarlink>
        {#if $currentUser?.IsSuperUser}
          <Sidebarlink url="/{team}/settings">
            <Settings class="h-4 w-4" />
            <span class="mx-2">Settings</span>
          </Sidebarlink>
        {/if}
      </nav>

      <div class="mt-6 -mx-3 space-y-2">
        <div class="-mx-3 px-3">
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
                    src={$currentUser?.GithubAvatarUrl}
                    alt="avatar"
                  />
                  <span
                    class="text-sm font-medium text-gray-700 dark:text-gray-200 overflow-clip text-ellipsis"
                    >{$currentUser?.FirstName
                      ? `${$currentUser?.FirstName} ${$currentUser?.LastName}`
                      : $currentUser?.Email}</span
                  >
                </div>
                <MoreVertical strokeWidth={1.5} class="h-4 w-4" />
              </Button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Content class="w-52 space-y-2">
              {#if $currentUser?.IsSuperUser}
                <DropdownMenu.Item class="hover:cursor-pointer">
                  <Link
                    to={`/${team}/new-team`}
                    class="flex w-full items-center"
                  >
                    <BadgePlus strokeWidth={1.5} class="h-4 w-4" />
                    <span class="mx-2">New team</span>
                  </Link>
                </DropdownMenu.Item>
                <Separator />
              {/if}
              <DropdownMenu.Item on:click={logout} class="hover:cursor-pointer">
                <LogOut strokeWidth={1.5} class="h-4 w-4" />
                <span class="mx-2">Logout</span>
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Root>
        </div>
      </div>
    </div>
  </aside>
  <aside class="w-full">
    <div class="mx-auto pb-16 pt-6 w-full px-16">
      <Router {url}>
        <Route path="/overview"><Overview /></Route>
        <Route path="/connections"><Connections /></Route>
        <Route path="/settings"><SettingsPage /></Route>
        <Route path="/my-account"><MyAccount /></Route>
        <Route path="/users"><UsersPage /></Route>
        <Route path="/new-team"><NewTeam /></Route>
        <Route path="*"><Notfound /></Route>
      </Router>
    </div>
  </aside>
</div>
