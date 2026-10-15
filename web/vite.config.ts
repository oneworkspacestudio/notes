import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";
export default defineConfig({
  base: "./",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "src"),
      events: "events/",
      punycode: "punycode/",
    },
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": "http://127.0.0.1:8795",
      "/auth": "http://127.0.0.1:8795",
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          joplin: ["@joplin/renderer/MdToHtml.js"],
          editor: ["@tiptap/react", "@tiptap/starter-kit", "@tiptap/markdown"],
        },
      },
    },
  },
});
