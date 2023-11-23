<script lang="ts">
  // @ts-expect-error
  import { Router, Route } from "svelte-routing";
  import Home from "./pages/home.svelte";
  import Notfound from "./pages/notfound.svelte";
  import App from "./pages/app/app.svelte";
  export let url = "";
</script>

<Router {url}>
  <Route path="/"><Home /></Route>
  <Route path="/*"><App /></Route>
  <Route path=""><Notfound /></Route>
</Router>
