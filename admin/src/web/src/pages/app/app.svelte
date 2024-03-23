<script lang="ts">
  import {
    ArrowUpDown,
    BadgePlus,
    Home,
    MoreVertical,
    Settings,
    Settings2Icon,
    User,
    Users,
  } from "lucide-svelte";

  import Sidebarlink from "$lib/components/sidebarlink.svelte";
  import TeamSelector from "$lib/components/team-selector.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
  import Separator from "$lib/components/ui/separator/separator.svelte";
  import { currentUser } from "$lib/store";
  import { LogOut } from "lucide-svelte";
  import { onMount, setContext } from "svelte";
  import { Link, Route, Router, navigate } from "svelte-routing";
  import AppLayout from "../app-layout.svelte";
  import Connections from "./connections.svelte";
  import MyAccount from "./myaccount.svelte";
  import Notfound from "./notfound.svelte";
  import Overview from "./overview.svelte";
  import SettingsPage from "./settings.svelte";
  import UsersPage from "./users.svelte";

  export let url = "";
  export let team: string;

  setContext("team", team);

  const logout = async () => {
    const res = await fetch("/api/v1/auth/logout", {
      method: "POST",
    });

    if (res.ok) {
      navigate("/");
    }
  };

  const getLoggedInUser = async () => {
    const response = await fetch(`/api/v1/user/me`, {
      headers: {
        "Content-Type": "application/json",
        "x-team-slug": team,
      },
    });
    currentUser.set(await response.json());
  };

  onMount(async () => {
    getLoggedInUser();
  });
</script>

<AppLayout>
  <div slot="sidebar" class="flex flex-col h-full">
    <TeamSelector />

    <div class="flex flex-col justify-between flex-1 mt-6 mx-4">
      <nav class="flex-1 -mx-3 space-y-3">
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
        {#if $currentUser?.user.is_superuser}
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
                    src={$currentUser?.user.github_user?.github_avatar_url}
                    alt="avatar"
                  />
                  <span
                    class="text-sm font-medium text-gray-700 dark:text-gray-200 overflow-clip text-ellipsis"
                    >{$currentUser?.user.first_name
                      ? `${$currentUser?.user.first_name} ${$currentUser?.user.last_name}`
                      : $currentUser?.user.email}</span
                  >
                </div>
                <MoreVertical strokeWidth={1.5} class="h-4 w-4" />
              </Button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Content class="w-52 space-y-2">
              {#if $currentUser?.user.is_superuser}
                <DropdownMenu.Item class="hover:cursor-pointer">
                  <Link
                    to="/instance-settings"
                    class="flex w-full items-center"
                  >
                    <Settings2Icon strokeWidth={1.5} class="h-4 w-4" />
                    <span class="mx-2">Instance settings</span>
                  </Link>
                </DropdownMenu.Item>
                <Separator />
                <DropdownMenu.Item class="hover:cursor-pointer">
                  <Link to="/new-team" class="flex w-full items-center">
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
  </div>

  <span slot="body">
    <Router {url}>
      <Route path="/overview"><Overview /></Route>
      <Route path="/connections"><Connections /></Route>
      <Route path="/settings"><SettingsPage /></Route>
      <Route path="/my-account"><MyAccount /></Route>
      <Route path="/users"><UsersPage /></Route>
      <Route path="*"><Notfound /></Route>
    </Router>
  </span>
</AppLayout>
