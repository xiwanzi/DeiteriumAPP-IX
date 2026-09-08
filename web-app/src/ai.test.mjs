import test from "node:test";
import assert from "node:assert/strict";
import { readAiEvents, aiSources } from "./ai-stream.js";
import { DeuteriumClient, mergeMessages } from "./api.js";
import { md5Base64 } from "./assets.js";

test("AI SSE preserves UTF-8 across chunks, CRLF and comments", async () => {
  const bytes = new TextEncoder().encode(': keepalive\r\n\r\nevent: delta\r\ndata: {"content":"中文回答"}\r\n\r\nevent: done\r\ndata: {"message":{"messageId":"ai-answer"}}\r\n\r\n');
  const response = new Response(new ReadableStream({ start(controller) { for (const byte of bytes) controller.enqueue(new Uint8Array([byte])); controller.close(); } }));
  const events = []; await readAiEvents(response, (type, payload) => events.push({ type, payload }));
  assert.deepEqual(events.map((item) => item.type), ["delta", "done"]);
  assert.equal(events[0].payload.content, "中文回答");
});
test("AI EOF without an authoritative done cannot become a completed answer", async () => {
  await assert.rejects(readAiEvents(new Response('event: delta\ndata: {"content":"partial"}\n\n'), () => {}), /中断/);
});
test("AI sources distinguish provider text links and reject executable or credential-bearing URLs", () => {
  const sources = aiSources([{ url: "https://example.com/report", title: "报告", origin: "annotation" }, { url: "https://example.com/page", origin: "provider_text" }, { url: "javascript:alert(1)" }, { url: "https://user:secret@example.com/" }]);
  assert.equal(sources.length, 2); assert.equal(sources[0].origin, "annotation"); assert.equal(sources[1].origin, "provider_text");
});
test("AI request uses the session CSRF and immutable request body without model credentials", async () => {
  const calls = []; const client = new DeuteriumClient({ fetchImpl: async (path, options) => {
    calls.push({ path, options });
    if (path.endsWith("/web/session")) return new Response(JSON.stringify({ data: { csrfToken: "only-memory" } }));
    return new Response('event: done\ndata: {"message":{"messageId":"answer","content":"hello"}}\n\n', { headers: { "Content-Type": "text/event-stream" } });
  } });
  await client.restore(); const body = { clientMessageId: "stable-ai-id", content: "question" };
  await client.streamAi(body, () => {}); await client.streamAi(body, () => {});
  assert.equal(calls[1].options.headers["X-CSRF-Token"], "only-memory");
  assert.equal(calls[1].options.body, calls[2].options.body);
  assert.deepEqual(Object.keys(JSON.parse(calls[1].options.body)).sort(), ["clientMessageId", "content"]);
  assert.equal(calls[1].options.credentials, "same-origin");
});
test("JSON quota errors before an SSE response stay clear service errors", async () => {
  const client = new DeuteriumClient({ fetchImpl: async () => new Response(JSON.stringify({ error: { code: "AI_QUOTA_EXCEEDED", message: "额度已用完" } }), { status: 429, headers: { "Content-Type": "application/json" } }) });
  await assert.rejects(client.streamAi({ clientMessageId: "key", content: "question" }, () => {}), (error) => error.code === "AI_QUOTA_EXCEEDED" && error.status === 429);
});
test("late private history reconciles a pending send by user and clientMessageId", () => {
  const pending = { id: "pending:one", channel: "conversation-one", clientMessageId: "one", mine: true, sentAt: "2026-09-08T01:00:00Z" };
  const actual = { ...pending, id: "message-authoritative", status: "sent" };
  assert.deepEqual(mergeMessages([pending], [actual]), [actual]);
  assert.equal(mergeMessages([pending], [{ ...actual, channel: "conversation-two" }]).length, 2);
});
test("browser S3 Content-MD5 uses the standard base64 digest", () => {
  assert.equal(md5Base64(new TextEncoder().encode("abc").buffer), "kAFQmDzST7DWlj99KOF/cg==");
  assert.equal(md5Base64(new ArrayBuffer(0)), "1B2M2Y8AsgTpgAmY7PhCfg==");
});
test("web registration uses a Cookie endpoint instead of requesting an App bearer", async () => {
  const calls = []; const client = new DeuteriumClient({ fetchImpl: async (path, options) => { calls.push({ path, options }); return new Response(JSON.stringify({ data: { user: { userId: "account" }, csrfToken: "registered-csrf" } })); } });
  await client.completeVerification("register", { verificationToken: "verified-game-identity", code: "123456", password: "test-only-password" });
  assert.equal(calls[0].path, "/api/v1/web/register");
  await client.request("/api/v1/account/me/profile", { method: "PATCH", body: {} });
  assert.equal(calls[1].options.headers["X-CSRF-Token"], "registered-csrf");
});
