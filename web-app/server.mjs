import { createServer } from "node:http";
import { readFile, stat } from "node:fs/promises";
import { dirname, extname, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import {
  backendOrigin,
  publicConfig,
  proxyHttp,
  proxyUpgrade,
} from "./gateway.mjs";
const root = resolve(dirname(fileURLToPath(import.meta.url)), "dist"),
  host = process.env.WEB_HOST || "127.0.0.1",
  port = Number(process.env.WEB_PORT || 5180),
  target = backendOrigin();
const types = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".webp": "image/webp",
  ".json": "application/json; charset=utf-8",
};
await stat(resolve(root, "index.html")).catch(() => {
  throw new Error("Build first: npm run build");
});
const server = createServer(async (req, res) => {
  res.setHeader("X-Content-Type-Options", "nosniff");
  res.setHeader("Referrer-Policy", "same-origin");
  try {
    const path = decodeURIComponent(
      new URL(req.url, "http://localhost").pathname,
    );
    if (path.startsWith("/api/")) {
      if (target) {
        proxyHttp(req, res, target);
        return;
      }
      res.writeHead(503, {
        "Content-Type": "application/json; charset=utf-8",
        "Cache-Control": "no-store",
      });
      res.end(
        JSON.stringify({
          error: {
            code: "SERVICE_UNAVAILABLE",
            message: "服务暂不可用，请稍后重试。",
          },
        }),
      );
      return;
    }
    if (!["GET", "HEAD"].includes(req.method)) {
      res.writeHead(405, { Allow: "GET, HEAD" });
      res.end();
      return;
    }
    if (path === "/web-config.json") {
      res.writeHead(200, {
        "Content-Type": "application/json; charset=utf-8",
        "Cache-Control": "no-store",
      });
      res.end(
        req.method === "HEAD"
          ? undefined
          : JSON.stringify(publicConfig(target)),
      );
      return;
    }
    let file = resolve(root, "." + path);
    if (
      !(file === root || file.startsWith(root + sep)) ||
      path.includes(String.fromCharCode(0))
    ) {
      res.writeHead(403);
      res.end();
      return;
    }
    if (!extname(path)) file = resolve(root, "index.html");
    const content = await readFile(file);
    res.writeHead(200, {
      "Content-Type": types[extname(file)] || "application/octet-stream",
      "Content-Length": content.length,
      "Cache-Control":
        extname(file) === ".html" ? "no-cache" : "public, max-age=3600",
    });
    res.end(req.method === "HEAD" ? undefined : content);
  } catch {
    res.writeHead(404, { "Content-Type": "text/plain; charset=utf-8" });
    res.end("Not found");
  }
});
server.on("upgrade", (req, socket, head) =>
  proxyUpgrade(req, socket, head, target),
);
server.listen(port, host, () =>
  console.log(
    `Deuterium Web 2.0.4 (${target ? "connected" : "unconfigured"}): http://${host}:${port}`,
  ),
);
