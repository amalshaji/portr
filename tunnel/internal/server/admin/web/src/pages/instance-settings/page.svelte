<script lang="ts">
  import Sidebarlink from "$lib/components/sidebarlink.svelte";
  import { ArrowLeft, LayoutGrid, Mail } from "lucide-svelte";
  import { Route, Router } from "svelte-routing";
  import AppLayout from "../app-layout.svelte";
  import Notfound from "../notfound.svelte";
  import Index from "./index.svelte";
  import Newteam from "./newteam.svelte";

  export let url = "";
</script>

<AppLayout>
  <span slot="sidebar">
    <div class="flex h-full max-h-screen flex-col gap-2">
      <div class="flex h-14 items-center border-b border-gray-300 px-4 lg:h-[60px] lg:px-6">
        <a href="/" class="flex items-center space-x-2 text-black hover:text-gray-600">
          <ArrowLeft class="h-4 w-4" />
          <span class="font-semibold">Instance settings</span>
        </a>
      </div>
      <div class="flex-1">
        <nav
          class="grid items-start px-2 text-sm font-medium lg:px-4 space-y-1"
        >
          <Sidebarlink url="/instance-settings/team">
            <LayoutGrid class="h-4 w-4" />
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
