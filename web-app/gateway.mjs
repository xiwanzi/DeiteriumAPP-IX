import http from "node:http";
import https from "node:https";
export function backendOrigin(raw = process.env.WEB_API_ORIGIN) {
  if (!raw) return null;
  const url = new URL(raw);
  if (
    url.username ||
    url.password ||
    url.search ||
    url.hash ||
    url.pathname !== "/" ||
    !["http:", "https:"].includes(url.protocol)
  )
    throw new Error(
      "WEB_API_ORIGIN must be an origin without credentials or path",
    );
  if (
    url.protocol === "http:" &&
    !["127.0.0.1", "[::1]"].includes(url.hostname)
  )
    throw new Error("Non-loopback backend connections require HTTPS");
  return url;
}
export const publicConfig = (target) => ({
  mode: "connected",
  version: "2.0.7",
  configured: Boolean(target),
});
function requestOptions(req, target) {
  const input = new URL(req.url, "http://localhost");
  return {
    protocol: target.protocol,
    hostname: target.hostname,
    port: target.port,
    method: req.method,
    path: input.pathname + input.search,
    headers: {
      ...req.headers,
      host: req.headers.host || target.host,
      "x-real-ip": (req.socket.remoteAddress || "127.0.0.1").replace(
        /^::ffff:/,
        "",
      ),
      "x-forwarded-for": undefined,
      forwarded: undefined,
    },
  };
}
function headers(req, target) {
  const options = requestOptions(req, target);
  for (const key of Object.keys(options.headers))
    if (options.headers[key] === undefined) delete options.headers[key];
  return options;
}
export function proxyHttp(req, res, target) {
  const transport = target.protocol === "https:" ? https : http;
  const upstream = transport.request(headers(req, target), (reply) => {
    res.writeHead(reply.statusCode, reply.headers);
    reply.pipe(res);
  });
  upstream.setTimeout(30000, () => upstream.destroy(new Error("timeout")));
  upstream.on("error", () => {
    if (!res.headersSent)
      res.writeHead(502, {
        "Content-Type": "application/json; charset=utf-8",
        "Cache-Control": "no-store",
      });
    res.end(
      JSON.stringify({
        error: {
          code: "UPSTREAM_UNAVAILABLE",
          message: "后端连接暂不可用，请稍后重试。",
        },
      }),
    );
  });
  req.on("aborted", () => upstream.destroy());
  res.on("close", () => upstream.destroy());
  req.pipe(upstream);
}
export function proxyUpgrade(req, socket, head, target) {
  if (
    !target ||
    new URL(req.url, "http://localhost").pathname !== "/api/v1/chat/ws"
  ) {
    socket.end("HTTP/1.1 404 Not Found\r\nConnection: close\r\n\r\n");
    return;
  }
  const transport = target.protocol === "https:" ? https : http,
    upstream = transport.request(headers(req, target));
  upstream.setTimeout(10000, () => upstream.destroy());
  upstream.on("upgrade", (response, peer, peerHead) => {
    upstream.setTimeout(0);
    socket.write(
      `HTTP/1.1 101 Switching Protocols\r\n${response.rawHeaders.reduce((s, h, i, all) => (i % 2 ? s : s + `${h}: ${all[i + 1]}\r\n`), "")}\r\n`,
    );
    if (peerHead.length) socket.write(peerHead);
    if (head.length) peer.write(head);
    peer.on("error", () => socket.destroy());
    socket.on("error", () => peer.destroy());
    peer.on("close", () => socket.destroy());
    socket.on("close", () => peer.destroy());
    socket.pipe(peer).pipe(socket);
  });
  upstream.on("response", (r) => {
    socket.write(
      `HTTP/1.1 ${r.statusCode} ${r.statusMessage}\r\nConnection: close\r\n\r\n`,
    );
    r.pipe(socket);
  });
  upstream.on("error", () =>
    socket.end("HTTP/1.1 502 Bad Gateway\r\nConnection: close\r\n\r\n"),
  );
  upstream.end();
}
