import "@fontsource/geist/400.css";
import "@fontsource/geist/500.css";
import "@fontsource/geist/600.css";
import "@fontsource/geist/700.css";
import "./app.pcss";
import App from "./App.svelte";

const app = new App({
  target: document.getElementById("app")!,
});

export default app;
