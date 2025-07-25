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
    User,
    HelpCircle,
    PlusCircle,
    Settings2Icon,
  } from "lucide-svelte";
  import { onMount, setContext } from "svelte";
  import { Link, Route, Router, navigate } from "svelte-routing";
  import AppLayout from "../app-layout.svelte";
  import Notfound from "../notfound.svelte";
  import Connections from "./connections.svelte";
  import Myaccount from "./myaccount.svelte";
  import Overview from "./overview.svelte";
  import UsersPage from "./users.svelte";
  import NewTeamDialog from "$lib/components/new-team-dialog.svelte";

  export let team: string;
  export let url = "";

  let newTeamDialogOpen = false;

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

<NewTeamDialog bind:isOpen={newTeamDialogOpen} />

<AppLayout>
  <span slot="sidebar">
    <div class="flex h-full max-h-screen flex-col">
      <div class="flex h-14 items-center border-b px-4 lg:h-[60px] lg:px-6">
        <TeamSelector />
      </div>

      <div class="flex-1 overflow-auto py-2">
        <div class="px-3 py-2">
          <h2
            class="mb-2 px-4 text-xs font-semibold tracking-tight text-gray-500 uppercase"
          >
            Main
          </h2>
          <nav class="grid gap-1 px-2">
            <Sidebarlink url="/{team}/overview" extraClass="hover:bg-gray-100">
              <Home class="h-4 w-4 mr-2" />
              Overview
            </Sidebarlink>

            <Sidebarlink
              url="/{team}/connections"
              extraClass="hover:bg-gray-100"
            >
              <ArrowUpDown class="h-4 w-4 mr-2" />
              Connections
            </Sidebarlink>
          </nav>
        </div>

        <div class="px-3 py-2">
          <h2
            class="mb-2 px-4 text-xs font-semibold tracking-tight text-gray-500 uppercase"
          >
            Management
          </h2>
          <nav class="grid gap-1 px-2">
            <Sidebarlink url="/{team}/users" extraClass="hover:bg-gray-100">
              <Users class="h-4 w-4 mr-2" />
              Users
            </Sidebarlink>

            <Sidebarlink
              url="/{team}/my-account"
              extraClass="hover:bg-gray-100"
            >
              <User class="h-4 w-4 mr-2" />
              Account & Settings
            </Sidebarlink>
          </nav>
        </div>

        {#if $currentUser?.user.is_superuser}
          <div class="px-3 py-2">
            <h2
              class="mb-2 px-4 text-xs font-semibold tracking-tight text-gray-500 uppercase"
            >
              Admin
            </h2>
            <nav class="grid gap-1 px-2">
              <div
                class="flex items-center gap-3 rounded-lg px-3 py-2 transition-all hover:text-primary text-sm text-muted-foreground hover:bg-gray-100 cursor-pointer"
                on:click={() => (newTeamDialogOpen = true)}
              >
                <PlusCircle class="h-4 w-4 mr-2" />
                New Team
              </div>
            </nav>
          </div>
        {/if}

        <div class="px-3 py-2">
          <div class="rounded-lg bg-gray-50 p-3 mt-2">
            <div class="flex items-center gap-3">
              <HelpCircle class="h-5 w-5 text-primary" />
              <div>
                <h3 class="text-sm font-medium">Need help?</h3>
                <p class="text-xs text-gray-500">Check our documentation</p>
              </div>
            </div>
            <Button
              class="mt-2 w-full text-xs"
              variant="outline"
              size="sm"
              href="https://portr.dev"
              target="_blank"
            >
              View Documentation
            </Button>
          </div>
        </div>
      </div>

      <div class="mt-auto border-t">
        <div class="flex items-center p-4">
          <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild let:builder>
              <Button
                builders={[builder]}
                variant="ghost"
                class="justify-between w-full text-left"
              >
                <div class="flex items-center gap-3">
                  <div class="relative">
                    <img
                      class="object-cover rounded-full h-8 w-8"
                      src={$currentUser?.user.github_user?.github_avatar_url}
                      alt="avatar"
                    />
                    <span
                      class="absolute bottom-0 right-0 h-2 w-2 rounded-full bg-green-500 ring-1 ring-white"
                    ></span>
                  </div>
                  <div class="flex flex-col justify-center">
                    <span
                      class="text-sm font-medium text-gray-700 dark:text-gray-200 overflow-clip text-ellipsis"
                      >{$currentUser?.user.first_name
                        ? `${$currentUser?.user.first_name} ${$currentUser?.user.last_name}`
                        : $currentUser?.user.email}</span
                    >
                    <span class="text-xs text-gray-500">Team Admin</span>
                  </div>
                </div>
                <div>
                  <EllipsisVertical class="h-4" />
                </div>
              </Button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Content class="w-[200px]">
              <DropdownMenu.Item
                on:click={logout}
                class="hover:cursor-pointer text-red-600"
              >
                <LogOut class="h-4 w-4 mr-2" />
                <span>Logout</span>
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Root>
        </div>
      </div>
    </div>
  </span>

  <span slot="body">
    <Router {url}>
      <Route path="/overview"><Overview /></Route>
      <Route path="/connections"><Connections /></Route>
      <Route path="/my-account"><Myaccount /></Route>
      <Route path="/users"><UsersPage /></Route>
      <Route path="/email-settings"><EmailSettings /></Route>
      <Route path="*"><Notfound /></Route>
    </Router>
  </span>
</AppLayout>
