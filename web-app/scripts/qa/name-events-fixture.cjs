async (page) => {
  const counts = {}, sockets = [], errors = [];
  let gameId = "OldName", sequence = 0, latestText = "原始消息";
  const now = new Date().toISOString();
  const self = { userId: "fixture-self", playerRef: "player_self", gameId: "SelfPlayer", displayName: "保留的昵称", qq: "12345", gameVerified: true, permissions: [] };
  const peer = () => ({ playerRef: "player_peer", gameId, qq: "23456", bio: "", avatar: null, version: 1 });
  const message = () => ({ messageId: "msg_peer", kind: "private_chat", conversationId: "conv_fixture", sender: { ...peer(), registered: true }, content: latestText, sentAt: now });
  const conversation = () => ({ conversationId: "conv_fixture", otherPlayer: peer(), lastMessage: message(), unreadCount: 0, updatedAt: now, version: 1 });
  page.on("pageerror", error => errors.push(error.message));
  await page.routeWebSocket("**/api/v1/chat/ws", socket => { sockets.push(socket); });
  await page.route("**/web-config.json", route => route.fulfill({ json: { mode: "connected", version: "2.0.38", configured: true } }));
  await page.route("**/api/v1/**", async route => {
    const path = new URL(route.request().url()).pathname;
    counts[path] = (counts[path] || 0) + 1;
    const reply = data => route.fulfill({ json: { data, page: { hasMore: false, nextCursor: null }, serverTime: now } });
    if (path === "/api/v1/web/session") return reply({ user: self, csrfToken: "fixture-csrf", expiresAt: "2030-01-01T00:00:00Z" });
    if (path === "/api/v1/players/player_self") return reply({ ...self, bio: "保留的个人简介", version: 1 });
    if (path === "/api/v1/players/player_peer") return reply(peer());
    if (path === "/api/v1/account/deletions") return reply({ items: [], cursor: 0, hasMore: false });
    if (path === "/api/v1/chat/player-names") return reply({ items: sequence ? [{ sequence, uuid: "11111111-1111-4111-8111-111111111111", playerRef: "player_peer", gameId }] : [], cursor: sequence, hasMore: false });
    if (path === "/api/v1/chat/conversations") return reply([conversation()]);
    if (path === "/api/v1/chat/conversations/conv_fixture") return reply(conversation());
    if (path === "/api/v1/chat/conversations/conv_fixture/messages") return reply([message()]);
    if (path.endsWith("/read")) return reply({ unreadCount: 0, lastReadMessageId: "msg_peer" });
    if (path === "/api/v1/chat/messages") return reply({ messages: [] });
    if (path === "/api/v1/chat/stickers") return reply({ stickers: [] });
    if (path === "/api/v1/notifications") return reply([]);
    if (path === "/api/v1/chat/follows" || path === "/api/v1/chat/player-directory") return reply({ players: [] });
    return reply({});
  });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto("http://127.0.0.1:5278/information");
  const editor = page.getByRole("textbox", { name: "消息内容" });
  await editor.waitFor();
  await editor.fill("正在编辑的草稿不能丢失");
  await page.waitForTimeout(500);
  const before = { ...counts };
  await page.waitForTimeout(5500);
  for (const path of ["/api/v1/chat/conversations", "/api/v1/chat/player-names", "/api/v1/account/deletions", "/api/v1/chat/messages"]) {
    if (counts[path] !== before[path]) throw new Error(`idle polling remains: ${path}`);
  }
  if (sockets.length !== 1) throw new Error(`expected one shared socket, got ${sockets.length}`);
  gameId = "NewName"; sequence = 1;
  sockets[0].send(JSON.stringify({ type: "sync.changed", payload: { topics: ["players.names"] } }));
  await page.getByText("NewName", { exact: true }).first().waitFor();
  if (await editor.inputValue() !== "正在编辑的草稿不能丢失") throw new Error("rename lost active draft");
  if (counts["/api/v1/chat/conversations"] !== before["/api/v1/chat/conversations"]) throw new Error("rename reloaded full conversation list");
  latestText = "单个会话更新";
  sockets[0].send(JSON.stringify({ type: "chat.conversation.changed", payload: { conversationId: "conv_fixture" } }));
  await page.getByText(latestText, { exact: true }).first().waitFor();
  if (counts["/api/v1/chat/conversations/conv_fixture"] !== 1) throw new Error("single-row refresh not used");
  if (counts["/api/v1/chat/conversations"] !== before["/api/v1/chat/conversations"]) throw new Error("single change reloaded every conversation");
  if (errors.length) throw new Error(errors.join("\n"));
  return { passed: true, sockets: sockets.length, idleMillis: 5500, fullConversationReads: counts["/api/v1/chat/conversations"], singleConversationReads: counts["/api/v1/chat/conversations/conv_fixture"], nameReads: counts["/api/v1/chat/player-names"], draftPreserved: true, pageErrors: errors };
}
