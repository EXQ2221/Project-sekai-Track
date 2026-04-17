import { defineConfig } from "vite";

export default defineConfig({
  server: {
    host: true,
    port: 5173,
    proxy: {
      "/register": "http://localhost:8080",
      "/login": "http://localhost:8080",
      "/refresh": "http://localhost:8080",
      "/musics": "http://localhost:8080",
      "/me": "http://localhost:8080",
      "/records": "http://localhost:8080",
      "/static": "http://localhost:8080"
    }
  },
  build: {
    outDir: "dist",
    emptyOutDir: true
  }
});
