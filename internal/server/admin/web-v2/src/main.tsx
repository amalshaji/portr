import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
// add the beginning of your app entry
// @ts-expect-error vite modulepreload polyfill has no bundled type declaration
import "vite/modulepreload-polyfill";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
