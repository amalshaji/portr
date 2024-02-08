<script>
  import { Router, Link, Route } from "svelte-routing";
  import Home from "./pages/Home.svelte";
  import Inspect from "./pages/Inspect.svelte";

  export let url = "";
</script>

<Router {url}>
  <div>
    <Route path="/:id" let:params><Inspect id={params.id} /></Route>
    <Route path="/"><Home /></Route>
  </div>
</Router>
