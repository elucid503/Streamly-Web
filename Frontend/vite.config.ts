import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({

  plugins: [react()],

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

    rollupOptions: {

      output: {

        manualChunks: {

          player: ["hls.js"], // saves ~100KB in the main bundle by putting hls.js in a separate chunk

        },

      },

    },

  },

});
