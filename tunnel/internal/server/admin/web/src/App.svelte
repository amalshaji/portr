<script lang="ts">
  import { Route, Router } from "svelte-routing";
  import { Toaster } from "svelte-sonner";
  import "./app.pcss";
  import App from "./pages/app/app.svelte";
  import Home from "./pages/home.svelte";
  import InstanceSettings from "./pages/instance-settings/page.svelte";
  import Notfound from "./pages/notfound.svelte";
  export let url = "";
</script>

<Toaster position="top-right" />

<Router {url}>
  <Route path="/"><Home /></Route>
  <Route path="/instance-settings/*"><InstanceSettings /></Route>
  <Route path="/:team/*" let:params><App team={params.team} /></Route>
  <Route path=""><Notfound /></Route>
</Router>
