<script lang="ts">
  import "./app.pcss";
  // @ts-expect-error
  import { Router, Route } from "svelte-routing";
  import { QueryClient, QueryClientProvider } from "@tanstack/svelte-query";

  import Home from "./pages/home.svelte";
  import Notfound from "./pages/notfound.svelte";
  import App from "./pages/app/app.svelte";
  export let url = "";

  const queryClient = new QueryClient();
</script>

<QueryClientProvider client={queryClient}>
  <Router {url}>
    <Route path="/"><Home /></Route>
    <Route path="/*"><App /></Route>
    <Route path=""><Notfound /></Route>
  </Router>
</QueryClientProvider>
