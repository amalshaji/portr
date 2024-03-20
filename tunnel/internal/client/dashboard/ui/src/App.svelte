<script>
  import { Router, Route } from "svelte-routing";
  import Home from "./pages/Home.svelte";
  import Inspect from "./pages/Inspect.svelte";
  import { Toaster } from "svelte-sonner";

  export let url = "";
</script>

<Toaster position="top-center" />

<Router {url}>
  <div>
    <Route path="/:id" let:params><Inspect id={params.id} /></Route>
    <Route path="/"><Home /></Route>
  </div>
</Router>
