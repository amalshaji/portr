<script lang="ts">
  import { Cable, BookOpenText, Settings, LogOut, Users } from "lucide-svelte";
  import { onMount } from "svelte";

  // @ts-expect-error
  import { Router, Route, Link, navigate } from "svelte-routing";
  import SettingsPage from "./settings.svelte";
  import Connections from "./connections.svelte";
  import Notfound from "./notfound.svelte";
  export let url = "";

  let user: any;

  const getMe = async () => {
    const res = await fetch("/api/users/me");
    user = await res.json();
  };

  const logout = async () => {
    const res = await fetch("/api/users/me/logout", {
      method: "POST",
      credentials: "include",
    });
    console.log(await res.text());
    if (res.ok) {
      navigate("/");
    }
  };

  onMount(() => {
    getMe();
  });
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
        <Cable></Cable>
      </Link>

      <Link
        to="/setup-client"
        class="p-1.5 text-gray-700 focus:outline-nones transition-colors duration-200 rounded-lg dark:text-gray-200 dark:hover:bg-gray-800 hover:bg-gray-100"
      >
        <BookOpenText></BookOpenText>
      </Link>

      <Link
        to="/invites"
        class="p-1.5 text-gray-700 focus:outline-nones transition-colors duration-200 rounded-lg dark:text-gray-200 dark:hover:bg-gray-800 hover:bg-gray-100"
      >
        <Users></Users>
      </Link>
    </nav>

    <div class="flex flex-col space-y-6">
      <Link
        to="/settings"
        class="p-1.5 text-gray-700 focus:outline-nones transition-colors duration-200 rounded-lg dark:text-gray-200 dark:hover:bg-gray-800 hover:bg-gray-100"
      >
        <Settings></Settings>
      </Link>

      <button
        on:click={logout}
        class="p-1.5 text-gray-700 focus:outline-nones transition-colors duration-200 rounded-lg dark:text-gray-200 dark:hover:bg-gray-800 hover:bg-gray-100"
      >
        <LogOut></LogOut>
      </button>

      <button>
        <img
          class="object-cover w-8 h-8 rounded-full"
          src={user?.avatarUrl}
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
