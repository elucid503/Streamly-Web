import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import legacy from "@vitejs/plugin-legacy";
import path from "path";

// webOS 6.x ships Chromium 79; Vite 7's default baseline target white-screens that engine.
const tvBrowserTargets = ["chrome >= 79", "firefox >= 78", "safari >= 14"];

export default defineConfig({

  plugins: [

    react(),
    legacy({

      modernTargets: tvBrowserTargets,
      modernPolyfills: true,
      renderLegacyChunks: false,

    }),

  ],

  resolve: {

    alias: {

      "@": path.resolve(__dirname, "./src"),
      events: path.resolve(__dirname, "./node_modules/events/events.js"),

    },

    dedupe: ["react", "react-dom"],

  },

  optimizeDeps: {

    include: ["react", "react-dom", "react-dom/client", "framer-motion", "events", "tiny-typed-emitter"],

  },

  server: {

    port: 5173,

    proxy: {

      "/api": {

        target: "http://localhost:8080",
        changeOrigin: true,

      },

    },

  },

  build: {

    cssTarget: ["chrome79", "firefox78", "safari14"],

    rollupOptions: {

      output: {

        manualChunks: {

          player: ["hls.js"], // saves ~100KB in the main bundle by putting hls.js in a separate chunk

        },

      },

    },

  },

});
