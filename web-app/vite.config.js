import { defineConfig } from "vite";
import { backendOrigin, publicConfig } from "./gateway.mjs";
const target = backendOrigin();
export default defineConfig({
  plugins: [
    {
      name: "deuterium-runtime-configuration",
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          if (req.url === "/web-config.json") {
            res.setHeader("Content-Type", "application/json");
            res.setHeader("Cache-Control", "no-store");
            res.end(JSON.stringify(publicConfig(target)));
          } else next();
        });
      },
    },
  ],
  server: {
    proxy: target
      ? { "/api/v1": { target: target.origin, ws: true, changeOrigin: false } }
      : undefined,
  },
});
