<script lang="ts">
  import Home from "lucide-svelte/icons/home";

  import Sidebarlink from "$lib/components/sidebarlink.svelte";
  import { ArrowUpDown, ArrowLeft } from "lucide-svelte";
  import { Route, Router, navigate } from "svelte-routing";
  import Newteam from "./newteam.svelte";
  import EmailSettings from "./emailSettings.svelte";
  import Index from "./index.svelte";
  import AppLayout from "../app-layout.svelte";
  import Notfound from "../notfound.svelte";

  export let url = "";
</script>

<AppLayout>
  <span slot="sidebar">
    <div class="flex h-full max-h-screen flex-col gap-2">
      <div class="flex h-14 items-center border-b px-4 lg:h-[60px] lg:px-6">
        <a href="/" class="flex items-center space-x-2">
          <ArrowLeft class="h-4 w-4" />
          <span class="font-semibold">Instance settings</span>
        </a>
      </div>
      <div class="flex-1">
        <nav
          class="grid items-start px-2 text-sm font-medium lg:px-4 space-y-1"
        >
          <Sidebarlink url="/instance-settings/email">
            <Home class="h-4 w-4" />
            Email
          </Sidebarlink>

          <Sidebarlink url="/instance-settings/team">
            <ArrowUpDown class="h-4 w-4" />
            Team
          </Sidebarlink>
        </nav>
      </div>
    </div>
  </span>

  <span slot="body">
    <Router {url}>
      <Route path="/email"><EmailSettings /></Route>
      <Route path="/team"><Newteam /></Route>
      <Route path="/"><Index /></Route>
      <Route path=""><Notfound /></Route>
    </Router>
  </span>
</AppLayout>
