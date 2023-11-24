<script lang="ts">
  import { Cable, BookOpenText, Settings, LogOut, Users } from "lucide-svelte";

  // @ts-expect-error
  import { Router, Route, Link, navigate } from "svelte-routing";
  import SettingsPage from "./settings.svelte";
  import Connections from "./connections.svelte";
  import Notfound from "./notfound.svelte";
  import { createQuery } from "@tanstack/svelte-query";
  import { getLoggedInUser } from "../../lib/services/user";
  import * as Tooltip from "$lib/components/ui/tooltip";

  export let url = "";

  const loggedInUserQuery = createQuery({
    queryKey: ["me"],
    queryFn: () => getLoggedInUser(),
  });

  const logout = async () => {
    const res = await fetch("/api/users/me/logout", {
      method: "POST",
    });
    console.log(await res.text());
    if (res.ok) {
      navigate("/");
    }
  };
</script>

<div class="flex">
  <aside
    class="flex flex-col items-center w-16 h-screen py-8 overflow-y-auto bg-white border-r rtl:border-l rtl:border-r-0 dark:bg-gray-900 dark:border-gray-700"
  >
    <nav class="flex flex-col flex-1 space-y-6">
      <a href="/">
        <img class="w-auto h-6 mx-auto" src="/static/favicon.svg" alt="" />
      </a>

      <Link
        to="/connections"
        class="p-1.5 text-gray-700 focus:outline-nones transition-colors duration-200 rounded-lg dark:text-gray-200 dark:hover:bg-gray-800 hover:bg-gray-100"
      >
        <Tooltip.Root>
          <Tooltip.Trigger>
            <Cable />
          </Tooltip.Trigger>
          <Tooltip.Content>
            <p>Connections</p>
          </Tooltip.Content>
        </Tooltip.Root>
      </Link>

      <Link
        to="/setup-client"
        class="p-1.5 text-gray-700 focus:outline-nones transition-colors duration-200 rounded-lg dark:text-gray-200 dark:hover:bg-gray-800 hover:bg-gray-100"
      >
        <Tooltip.Root>
          <Tooltip.Trigger>
            <BookOpenText />
          </Tooltip.Trigger>
          <Tooltip.Content>
            <p>Client setup</p>
          </Tooltip.Content>
        </Tooltip.Root>
      </Link>

      <Link
        to="/users"
        class="p-1.5 text-gray-700 focus:outline-nones transition-colors duration-200 rounded-lg dark:text-gray-200 dark:hover:bg-gray-800 hover:bg-gray-100"
      >
        <Tooltip.Root>
          <Tooltip.Trigger>
            <Users />
          </Tooltip.Trigger>
          <Tooltip.Content>
            <p>Users</p>
          </Tooltip.Content>
        </Tooltip.Root>
      </Link>
    </nav>

    <div class="flex flex-col space-y-6">
      <Link
        to="/settings"
        class="p-1.5 text-gray-700 focus:outline-nones transition-colors duration-200 rounded-lg dark:text-gray-200 dark:hover:bg-gray-800 hover:bg-gray-100"
      >
        <Tooltip.Root>
          <Tooltip.Trigger>
            <Settings />
          </Tooltip.Trigger>
          <Tooltip.Content>
            <p>Settings</p>
          </Tooltip.Content>
        </Tooltip.Root>
      </Link>

      <button
        on:click={logout}
        class="p-1.5 text-gray-700 focus:outline-nones transition-colors duration-200 rounded-lg dark:text-gray-200 dark:hover:bg-gray-800 hover:bg-gray-100"
      >
        <Tooltip.Root>
          <Tooltip.Trigger>
            <LogOut />
          </Tooltip.Trigger>
          <Tooltip.Content>
            <p>Logout</p>
          </Tooltip.Content>
        </Tooltip.Root>
      </button>

      <button>
        <img
          class="object-cover w-8 h-8 rounded-full"
          src={$loggedInUserQuery.isSuccess
            ? $loggedInUserQuery.data.avatarUrl
            : ""}
          alt=""
        />
      </button>
    </div>
  </aside>
  <aside class="w-full">
    <Router {url}>
      <Route path="/connections"><Connections /></Route>
      <Route path="/settings"><SettingsPage /></Route>
      <Route path="*"><Notfound /></Route>
    </Router>
  </aside>
</div>
