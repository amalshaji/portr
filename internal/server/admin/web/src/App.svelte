<script lang="ts">
  import "./app.pcss";
  import { Router, Route } from "svelte-routing";
  import { Toaster } from "svelte-sonner";

  import Home from "./pages/home.svelte";
  import Notfound from "./pages/notfound.svelte";
  import App from "./pages/app/app.svelte";
  import Setup from "./pages/setup.svelte";
  export let url = "";
</script>

<Toaster position="top-right" />

<Router {url}>
  <Route path="/"><Home /></Route>
  <Route path="/setup"><Setup /></Route>
  <Route path="/:team/*" let:params><App team={params.team} /></Route>
  <Route path=""><Notfound /></Route>
</Router>
