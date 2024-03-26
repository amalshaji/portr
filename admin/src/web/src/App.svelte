<script lang="ts">
  import "./app.pcss";
  import { Router, Route } from "svelte-routing";
  import { Toaster } from "svelte-sonner";
  import Home from "./pages/home.svelte";
  import Notfound from "./pages/notfound.svelte";
  import App from "./pages/app/app.svelte";
  import InstanceSettings from "./pages/instance-settings.svelte";
  import NewTeam from "./pages/new-team.svelte";
  export let url = "";
</script>

<Toaster position="top-right" />

<Router {url}>
  <Route path="/"><Home /></Route>
  <Route path="/instance-settings"><InstanceSettings /></Route>
  <Route path="/new-team"><NewTeam /></Route>
  <Route path="/:team/*" let:params><App team={params.team} /></Route>
  <Route path=""><Notfound /></Route>
</Router>
