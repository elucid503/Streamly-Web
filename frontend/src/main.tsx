import { App } from "@/App";
import "@/index.css";

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

if ("serviceWorker" in navigator) {

  window.addEventListener("load", () => {

    navigator.serviceWorker.register("/sw.js");

  });

}

createRoot(document.getElementById("root")!).render(

  <StrictMode>

    <App />

  </StrictMode>

);