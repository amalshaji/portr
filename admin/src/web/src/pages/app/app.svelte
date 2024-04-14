<script lang="ts">
  import Home from "lucide-svelte/icons/home";
  import Users from "lucide-svelte/icons/users";

  import Sidebarlink from "$lib/components/sidebarlink.svelte";
  import TeamSelector from "$lib/components/team-selector.svelte";
  import { Button } from "$lib/components/ui/button/index.js";
  import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
  import Separator from "$lib/components/ui/separator/separator.svelte";
  import { currentUser } from "$lib/store";
  import {
    ArrowUpDown,
    EllipsisVertical,
    LogOut,
    Settings,
    Settings2Icon,
    User,
  } from "lucide-svelte";
  import { onMount, setContext } from "svelte";
  import { Link, Route, Router, navigate } from "svelte-routing";
  import AppLayout from "../app-layout.svelte";
  import Notfound from "../notfound.svelte";
  import Connections from "./connections.svelte";
  import Myaccount from "./myaccount.svelte";
  import Overview from "./overview.svelte";
  import SettingsPage from "./settings.svelte";
  import UsersPage from "./users.svelte";

  export let team: string;
  export let url = "";

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
  <span slot="sidebar">
    <div class="flex h-full max-h-screen flex-col gap-2">
      <div class="flex h-14 items-center border-b px-4 lg:h-[60px] lg:px-6">
        <TeamSelector />
      </div>
      <div class="flex-1">
        <nav
          class="grid items-start px-2 text-sm font-medium lg:px-4 space-y-1"
        >
          <Sidebarlink url="/{team}/overview">
            <Home class="h-4 w-4" />
            Overview
          </Sidebarlink>

          <Sidebarlink url="/{team}/connections">
            <ArrowUpDown class="h-4 w-4" />
            Connections
          </Sidebarlink>

          <Sidebarlink url="/{team}/users">
            <Users class="h-4 w-4" />
            Users
          </Sidebarlink>

          <Sidebarlink url="/{team}/my-account">
            <User class="h-4 w-4" />
            My account
          </Sidebarlink>

          <Sidebarlink url="/{team}/settings">
            <Settings class="h-4 w-4" />
            Settings
          </Sidebarlink>
        </nav>
      </div>
      <div class="mt-auto mb-8 mx-auto">
        <div class="flex-1">
          <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild let:builder>
              <Button
                builders={[builder]}
                variant="ghost"
                class="justify-between w-[250px] text-left"
              >
                <div class="flex items-center space-x-1">
                  <img
                    class="object-cover rounded-full h-7 w-7"
                    src={$currentUser?.user.github_user?.github_avatar_url}
                    alt="avatar"
                  />
                  <span
                    class="text-sm font-medium text-gray-700 dark:text-gray-200 overflow-clip text-ellipsis w-4/5"
                    >{$currentUser?.user.first_name
                      ? `${$currentUser?.user.first_name} ${$currentUser?.user.last_name}`
                      : $currentUser?.user.email}</span
                  >
                </div>
                <div>
                  <EllipsisVertical class="h-4" />
                </div>
              </Button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Content class="w-[250px] space-y-1">
              {#if $currentUser?.user.is_superuser}
                <DropdownMenu.Item class="hover:cursor-pointer">
                  <Link
                    to="/instance-settings"
                    class="flex w-full items-center"
                  >
                    <Settings2Icon class="h-4 w-4" />
                    <span class="mx-2">Instance settings</span>
                  </Link>
                </DropdownMenu.Item>
                <Separator />
              {/if}
              <DropdownMenu.Item on:click={logout} class="hover:cursor-pointer">
                <LogOut class="h-4 w-4" />
                <span class="mx-2">Logout</span>
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Root>
        </div>
      </div>
    </div></span
  >

  <span slot="body">
    <Router {url}>
      <Route path="/overview"><Overview /></Route>
      <Route path="/connections"><Connections /></Route>
      <Route path="/settings"><SettingsPage /></Route>
      <Route path="/my-account"><Myaccount /></Route>
      <Route path="/users"><UsersPage /></Route>
      <Route path="*"><Notfound /></Route>
    </Router>
  </span>
</AppLayout>
