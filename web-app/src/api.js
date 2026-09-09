import { readAiEvents } from "./ai-stream.js";
export class ApiError extends Error {
  constructor(message, code = "REQUEST_FAILED", status = 0, retryAfter = 0) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    this.retryAfter = retryAfter;
  }
}
export class DeuteriumClient {
  #csrf = "";
  constructor({
    fetchImpl = globalThis.fetch.bind(globalThis),
    onUnauthorized = () => {},
    onSession = () => {},
  } = {}) {
    this.fetch = fetchImpl;
    this.onUnauthorized = onUnauthorized;
    this.onSession = onSession;
  }
  async request(
    path,
    { method = "GET", body, signal, sessionRequired = true, idempotencyKey } = {},
  ) {
    if (!path.startsWith("/api/v1/") || path.includes("://"))
      throw new ApiError("只允许本站 API 请求");
    const headers = { Accept: "application/json" };
    if (idempotencyKey) {
      if (!/^[A-Za-z0-9_.:-]{1,128}$/.test(idempotencyKey)) throw new ApiError("请求标识不正确。");
      headers["Idempotency-Key"] = idempotencyKey;
    }
    if (body !== undefined) headers["Content-Type"] = "application/json";
    if (
      !["GET", "HEAD"].includes(method) &&
      path !== "/api/v1/web/session" &&
      this.#csrf
    )
      headers["X-CSRF-Token"] = this.#csrf;
    if (method === "DELETE" && this.#csrf) headers["X-CSRF-Token"] = this.#csrf;
    let response;
    try {
      response = await this.fetch(path, {
        method,
        body: body === undefined ? undefined : JSON.stringify(body),
        headers,
        credentials: "same-origin",
        cache: "no-store",
        signal: signal || AbortSignal.timeout(15000),
      });
    } catch (e) {
      if (signal?.aborted) throw e;
      throw new ApiError(
        "无法连接服务，输入内容已保留，请稍后重试。",
        "NETWORK_ERROR",
      );
    }
    let value;
    try {
      value = await response.json();
    } catch {
      throw new ApiError(
        "服务返回格式不正确，请检查 API 代理配置。",
        "INVALID_RESPONSE",
        response.status,
      );
    }
    if (!response.ok || value.error) {
      const err = value.error || {};
      if (response.status === 401 && sessionRequired) {
        this.#csrf = "";
        this.onUnauthorized();
      }
      throw new ApiError(
        err.message || "请求未完成",
        err.code || "REQUEST_FAILED",
        response.status,
        Number(
          err.retryAfterSeconds || response.headers.get("Retry-After") || 0,
        ),
      );
    }
    return value;
  }
  async login(account, password) {
    const r = await this.request("/api/v1/web/session", {
      method: "POST",
      body: { account, password },
      sessionRequired: false,
    });
    this.#csrf = r.data.csrfToken;
    this.onSession(r.data);
    return r.data;
  }
  async restore() {
    const r = await this.request("/api/v1/web/session");
    this.#csrf = r.data.csrfToken;
    this.onSession(r.data);
    return r.data;
  }
  async logout() {
    const result = await this.request("/api/v1/web/session", {
      method: "DELETE",
    });
    this.#csrf = "";
    return result;
  }
  clear() {
    this.#csrf = "";
  }
  history(before, signal) {
    return this.request(
      `/api/v1/chat/messages?limit=50${before ? `&before=${encodeURIComponent(before)}` : ""}`,
      { signal },
    );
  }
  nodes() {
    return this.request("/api/v1/admin/core/nodes");
  }
  items(cursor) {
    return this.request(
      `/api/v1/admin/core/items${cursor ? `?afterItemRef=${encodeURIComponent(cursor.afterItemRef)}&afterRevision=${cursor.afterRevision}` : ""}`,
    );
  }
  balance(refresh = false) {
    return this.request(`/api/v1/wallet/balance${refresh ? "/refresh" : ""}`, { method: refresh ? "POST" : "GET" });
  }
  records(cursor) {
    return this.request(`/api/v1/wallet/records?limit=20${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`);
  }
  recipients(query) {
    return this.request(`/api/v1/wallet/recipients/search?query=${encodeURIComponent(query)}&type=auto`);
  }
  transfer(body) {
    return this.request("/api/v1/wallet/transfers", { method: "POST", body });
  }
  transferResult(transferId) {
    return this.request(`/api/v1/wallet/transfers/${encodeURIComponent(transferId)}`);
  }
  profile(playerRef) { return this.request(`/api/v1/players/${encodeURIComponent(playerRef)}`); }
  patchProfile(body) { return this.request("/api/v1/account/me/profile", { method: "PATCH", body }); }
  directory(query = "") { return this.request(`/api/v1/chat/player-directory?query=${encodeURIComponent(query)}`); }
  conversations(cursor) { return this.request(`/api/v1/chat/conversations?limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`); }
  createConversation(body) { return this.request("/api/v1/chat/conversations", { method: "POST", body }); }
  directMessages(conversationId, cursor) { return this.request(`/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/messages?limit=50${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`); }
  sendDirect(conversationId, body) { return this.request(`/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/messages`, { method: "POST", body }); }
  forwardDirect(conversationId, body) { return this.request(`/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/forwards`, { method: "POST", body }); }
  readDirect(conversationId, body) { return this.request(`/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/read`, { method: "POST", body }); }
  async streamAi(body, onEvent, signal) {
    let response;
    try { response = await this.fetch("/api/v1/ai/chat/stream", { method: "POST", credentials: "same-origin", cache: "no-store", headers: { Accept: "text/event-stream", "Content-Type": "application/json", "X-CSRF-Token": this.#csrf }, body: JSON.stringify(body), signal }); }
    catch { throw new ApiError("AI 连接暂不可用，可以继续查看同一次回复。", "NETWORK_ERROR"); }
    if (!response.ok || !response.headers.get("Content-Type")?.includes("text/event-stream")) {
      let value; try { value = await response.json(); } catch {}
      if (response.status === 401) { this.#csrf = ""; this.onUnauthorized(); }
      throw new ApiError(value?.error?.message || "AI 服务暂不可用，请稍后重试。", value?.error?.code || "AI_UNAVAILABLE", response.status);
    }
    await readAiEvents(response, async (event, payload) => {
      if (event === "error") throw new ApiError(payload.error?.message || "这次回复未能完成。", payload.error?.code || "AI_UNAVAILABLE");
      await onEvent(event, payload);
    });
  }
  async completeVerification(type, fields) {
    const result = await this.request(type === "register" ? "/api/v1/web/register" : "/api/v1/account/password-reset", {
      method: "POST", sessionRequired: false,
      body: { verificationToken: fields.verificationToken, code: fields.code,
        ...(type === "register" ? { password: fields.password } : { newPassword: fields.password }) },
    });
    if (type === "register") { this.#csrf = result.data.csrfToken; this.onSession(result.data); }
    return result;
  }
  requestVerification(type, fields) {
    return this.request(
      `/api/v1/account/${type === "register" ? "registration-code" : "password-reset-code"}`,
      {
        method: "POST",
        body: {
          gameId: fields.gameId,
          ...(fields.qq ? { qq: fields.qq } : {}),
          ...(type === "register" ? { password: fields.password } : {}),
        },
        sessionRequired: false,
      },
    );
  }
}
export function normalizeMessage(message, currentUser) {
  return {
    id: message.messageId,
    channel: message.conversationId || "public",
    text: message.content,
    sender: message.sender?.playerRef,
    senderName: message.sender?.gameId || "玩家",
    senderProfile: message.sender,
    mine: message.sender?.playerRef === currentUser.playerRef,
    source: message.sender?.source === "game" ? "游戏" : undefined,
    sentAt: message.sentAt,
    time: new Date(message.sentAt).toLocaleTimeString("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    }),
    status:
      message.sender?.playerRef === currentUser.playerRef ? "sent" : undefined,
    clientMessageId: message.clientMessageId,
    reply: message.reply ? { id: message.reply.messageId, text: message.reply.content, senderName: message.reply.sender?.gameId, availability: message.reply.availability } : null,
    forwarded: message.forwarded ? { id: message.forwarded.messageId, text: message.forwarded.content, senderName: message.forwarded.sender?.gameId, availability: message.forwarded.availability } : null,
  };
}
export function mergeMessages(old, incoming) {
  const map = new Map(old.map((m) => [m.id, m]));
  for (const m of incoming) {
    if (m.mine && m.clientMessageId && !m.id.startsWith("pending:")) {
      for (const [key, value] of map) if (key.startsWith("pending:") && value.clientMessageId === m.clientMessageId && value.channel === m.channel) map.delete(key);
    }
    map.set(m.id, { ...map.get(m.id), ...m });
  }
  return [...map.values()].sort(
    (a, b) => (Date.parse(a.sentAt) || 0) - (Date.parse(b.sentAt) || 0),
  );
}
export function recoveryCursor(previous, incoming, oldCursor, nextCursor) {
  const anchor = previous
    .filter((m) => !m.id.startsWith("pending:"))
    .at(-1)?.id;
  return anchor && incoming.some((m) => m.id === anchor)
    ? oldCursor
    : nextCursor;
}

export function createChatConnection({
  client,
  location = globalThis.location,
  WebSocketImpl = globalThis.WebSocket,
  onState,
  onMessage,
  onUnauthorized,
  onRecovery,
}) {
  let socket,
    stopped = false,
    timer,
    attempt = 0;
  const pending = new Map();
  const clearPending = () => {
    for (const p of pending.values()) {
      clearTimeout(p.timer);
      p.reject(
        new ApiError(
          "连接已中断，消息结果未知。重试会使用原消息编号。",
          "RESULT_UNKNOWN",
        ),
      );
    }
    pending.clear();
  };
  const connect = () => {
    if (stopped) return;
    clearTimeout(timer);
    onState(attempt ? "reconnecting" : "connecting");
    socket = new WebSocketImpl(
      `${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/api/v1/chat/ws`,
    );
    socket.onopen = () => {
      attempt = 0;
      onState("connected");
      onRecovery?.();
    };
    socket.onmessage = (event) => {
      let frame;
      try {
        frame = JSON.parse(event.data);
      } catch {
        return;
      }
      if (frame.type === "chat.message" && frame.payload?.message) {
        onMessage(frame.payload.message);
        return;
      }
      if (frame.type === "chat.send.result") {
        const p = pending.get(frame.requestId);
        if (!p) return;
        clearTimeout(p.timer);
        pending.delete(frame.requestId);
        if (frame.payload?.status === "accepted") p.resolve(frame.payload);
        else {
          const e = frame.payload?.error || {};
          if (e.code === "UNAUTHORIZED") onUnauthorized();
          p.reject(
            new ApiError(e.message || "消息未被接受", e.code || "SEND_FAILED"),
          );
        }
      }
    };
    socket.onerror = () => {};
    socket.onclose = async () => {
      clearPending();
      if (stopped) return;
      onState("reconnecting");
      try {
        await client.restore();
      } catch (e) {
        if (e.status === 401) {
          stopped = true;
          onState("disconnected");
          onUnauthorized();
          return;
        }
      }
      if (stopped) return;
      attempt++;
      timer = setTimeout(
        connect,
        Math.min(15000, 1000 * 2 ** Math.min(attempt - 1, 4)),
      );
    };
  };
  connect();
  return {
    send(content, clientMessageId, metadata = {}) {
      if (stopped || socket?.readyState !== 1)
        return Promise.reject(
          new ApiError("实时连接尚未就绪，请连接后再试。", "NOT_CONNECTED"),
        );
      if (
        !content.trim() ||
        [...content].length > 256 ||
        /[\r\n\t]/.test(content)
      )
        return Promise.reject(
          new ApiError(
            "当前公共频道支持 1–256 字的单段消息。",
            "INVALID_REQUEST",
          ),
        );
      const requestId = `send_${clientMessageId}`;
      if (pending.has(requestId))
        return Promise.reject(
          new ApiError("这条消息正在确认中，请稍候。", "IN_PROGRESS"),
        );
      return new Promise((resolve, reject) => {
        const timer = setTimeout(() => {
          pending.delete(requestId);
          reject(
            new ApiError(
              "未收到发送回执。重试将使用同一消息编号确认结果。",
              "RESULT_UNKNOWN",
            ),
          );
        }, 15000);
        pending.set(requestId, { resolve, reject, timer });
        try {
          socket.send(
            JSON.stringify({
              type: "chat.send",
              requestId,
              sentAt: new Date().toISOString(),
              payload: { clientMessageId, content, ...(metadata.replyToMessageId ? { replyToMessageId: metadata.replyToMessageId } : {}), ...(metadata.mentionedPlayerRefs?.length ? { mentionedPlayerRefs: metadata.mentionedPlayerRefs } : {}) },
            }),
          );
        } catch {
          clearTimeout(timer);
          pending.delete(requestId);
          reject(
            new ApiError("发送结果未知，请使用原消息重试。", "RESULT_UNKNOWN"),
          );
        }
      });
    },
    reconnect() {
      if (stopped) return;
      clearTimeout(timer);
      socket.onclose = null;
      socket.close();
      clearPending();
      connect();
    },
    close() {
      stopped = true;
      clearTimeout(timer);
      clearPending();
      socket?.close();
      onState("disconnected");
    },
  };
}
