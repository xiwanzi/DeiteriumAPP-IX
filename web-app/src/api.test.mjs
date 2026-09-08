import test from "node:test";
import assert from "node:assert/strict";
import {
  DeuteriumClient,
  ApiError,
  mergeMessages,
  createChatConnection,
  recoveryCursor,
} from "./api.js";

test("reconnect keeps a cursor through missing history when latest messages have no overlap", () => {
  assert.equal(
    recoveryCursor(
      [{ id: "old-last" }],
      [{ id: "new-last" }],
      "old-before",
      "gap-before",
    ),
    "gap-before",
  );
  assert.equal(
    recoveryCursor(
      [{ id: "old-last" }],
      [{ id: "old-last" }, { id: "new-last" }],
      "old-before",
      "new-before",
    ),
    "old-before",
  );
  assert.equal(
    recoveryCursor([], [{ id: "first" }], null, "initial-before"),
    "initial-before",
  );
});

const json = (value, status = 200, headers = {}) =>
  new Response(JSON.stringify(value), {
    status,
    headers: { "Content-Type": "application/json", ...headers },
  });
test("web login uses same-origin Cookie and keeps CSRF out of serialized storage", async () => {
  const calls = [];
  const c = new DeuteriumClient({
    fetchImpl: async (path, opts) => {
      calls.push({ path, opts });
      return json({
        data: {
          user: { userId: "id" },
          csrfToken: "memory-only-csrf",
          expiresAt: "2026-09-15T00:00:00Z",
        },
      });
    },
  });
  await c.login("Player", "test-only-password");
  assert.equal(calls[0].path, "/api/v1/web/session");
  assert.equal(calls[0].opts.credentials, "same-origin");
  assert.equal(calls[0].opts.headers.Authorization, undefined);
  assert.equal(calls[0].opts.headers["X-CSRF-Token"], undefined);
  assert.ok(!JSON.stringify(c).includes("memory-only-csrf"));
  await c.logout();
  assert.equal(calls[1].opts.headers["X-CSRF-Token"], "memory-only-csrf");
  assert.equal(calls[1].opts.method, "DELETE");
});
test("restore refreshes CSRF for mutation requests and 401 invalidates identity", async () => {
  let expired = false,
    invalidations = 0;
  const calls = [];
  const c = new DeuteriumClient({
    onUnauthorized: () => invalidations++,
    fetchImpl: async (path, opts) => {
      calls.push(opts);
      return expired
        ? json({ error: { code: "UNAUTHORIZED", message: "失效" } }, 401)
        : json({ data: { user: { userId: "a" }, csrfToken: "restored" } });
    },
  });
  await c.restore();
  await c.request("/api/v1/test", { method: "POST", body: { value: 1 } });
  assert.equal(calls.at(-1).headers["X-CSRF-Token"], "restored");
  expired = true;
  await assert.rejects(c.restore(), (e) => e.status === 401);
  assert.equal(invalidations, 1);
});
test("failed credentials do not create a session and rate limits expose retry timing", async () => {
  let invalidations = 0;
  const c = new DeuteriumClient({
    onUnauthorized: () => invalidations++,
    fetchImpl: async () =>
      json(
        {
          error: {
            code: "LOGIN_LOCKED",
            message: "稍后重试",
            retryAfterSeconds: 900,
          },
        },
        429,
      ),
  });
  await assert.rejects(
    c.login("x", "wrong"),
    (e) => e.code === "LOGIN_LOCKED" && e.retryAfter === 900,
  );
  assert.equal(invalidations, 0);
});
test("HTML proxy errors and unavailable network never fall back to demo identity", async () => {
  const c = new DeuteriumClient({
    fetchImpl: async () => new Response("<html>not an API</html>"),
  });
  await assert.rejects(c.restore(), (e) => e.code === "INVALID_RESPONSE");
  const n = new DeuteriumClient({
    fetchImpl: async () => {
      throw new TypeError("offline");
    },
  });
  await assert.rejects(n.restore(), (e) => e.code === "NETWORK_ERROR");
  await assert.rejects(c.request("https://outside.test/api"), ApiError);
});
test("history and realtime events merge once by server message ID", () => {
  const one = { id: "m1", text: "one", sentAt: "2026-09-08T01:00:00Z" },
    two = { id: "m2", text: "two", sentAt: "2026-09-08T01:00:01Z" };
  assert.deepEqual(
    mergeMessages([two], [one, two]).map((m) => m.id),
    ["m1", "m2"],
  );
  assert.equal(mergeMessages([one], [{ ...one, status: "sent" }]).length, 1);
});
test("WebSocket handshake has no URL token and sends immutable client message identifiers", async () => {
  let ws;
  class FakeSocket {
    constructor(url) {
      ws = this;
      this.url = url;
      this.readyState = 1;
      queueMicrotask(() => this.onopen?.());
    }
    send(raw) {
      this.last = JSON.parse(raw);
    }
    close() {
      this.readyState = 3;
    }
  }
  const states = [],
    transport = createChatConnection({
      client: { restore: async () => ({}) },
      location: { protocol: "https:", host: "chat.example.test" },
      WebSocketImpl: FakeSocket,
      onState: (s) => states.push(s),
      onMessage: () => {},
      onUnauthorized: () => {},
    });
  assert.equal(ws.url, "wss://chat.example.test/api/v1/chat/ws");
  const sending = transport.send("消息", "stable-id");
  assert.equal(ws.last.payload.clientMessageId, "stable-id");
  ws.onmessage({
    data: JSON.stringify({
      type: "chat.send.result",
      requestId: "send_stable-id",
      payload: { status: "accepted", messageId: "server-id" },
    }),
  });
  assert.equal((await sending).messageId, "server-id");
  transport.close();
});
test("rejected and disconnected sends never produce a success receipt", async () => {
  let ws;
  class FakeSocket {
    constructor() {
      ws = this;
      this.readyState = 1;
    }
    send(raw) {
      this.last = JSON.parse(raw);
    }
    close() {
      this.readyState = 3;
    }
  }
  const transport = createChatConnection({
    client: { restore: async () => ({}) },
    location: { protocol: "http:", host: "127.0.0.1:1" },
    WebSocketImpl: FakeSocket,
    onState: () => {},
    onMessage: () => {},
    onUnauthorized: () => {},
  });
  const p = transport.send("消息", "retry-id");
  ws.onmessage({
    data: JSON.stringify({
      type: "chat.send.result",
      requestId: "send_retry-id",
      payload: {
        status: "failed",
        error: { code: "PLUGIN_BRIDGE_UNAVAILABLE", message: "Core 离线" },
      },
    }),
  });
  await assert.rejects(p, (e) => e.code === "PLUGIN_BRIDGE_UNAVAILABLE");
  const unknown = transport.send("消息", "retry-id");
  transport.close();
  await assert.rejects(unknown, (e) => e.code === "RESULT_UNKNOWN");
});

test("session restoration reports changed browser identity before more account actions", async () => {
  const identities = []; let userId = "first-user";
  const client = new DeuteriumClient({ onSession: (session) => identities.push(session.user.userId), fetchImpl: async () => json({ data: { user: { userId }, csrfToken: `csrf-${userId}` } }) });
  await client.restore(); userId = "second-user"; await client.restore();
  assert.deepEqual(identities, ["first-user", "second-user"]);
});
