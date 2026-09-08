package com.deuterium.app.uilab

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.withTimeoutOrNull
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONObject
import java.time.Instant
import java.util.UUID
import java.util.concurrent.ConcurrentHashMap

/** Acknowledgement is required; a timeout never becomes a synthetic sent message. */
class BackendChat(
    private val http: OkHttpClient,
    private val baseUrl: String,
    private val token: String,
    private val onMessage: (JSONObject) -> Unit,
    private val onState: (Boolean, String?) -> Unit,
    private val onOpen: () -> Unit,
) : WebSocketListener() {
    private var socket: WebSocket? = null
    private val pending = ConcurrentHashMap<String, CompletableDeferred<JSONObject>>()
    @Volatile var connected = false; private set
    @Volatile private var stopped = false
    fun connect() {
        if(stopped) return
        socket = http.newWebSocket(Request.Builder().url("${baseUrl.trimEnd('/')}/api/v1/chat/ws")
            .header("Authorization", "Bearer $token").build(), this)
    }
    override fun onOpen(webSocket: WebSocket, response: Response) {
        if(stopped) { webSocket.close(1000, "closed"); return }
        connected = true; onState(true, null); onOpen()
    }
    override fun onMessage(webSocket: WebSocket, text: String) {
        val frame = runCatching { JSONObject(text) }.getOrNull() ?: return
        val payload = frame.optJSONObject("payload") ?: return
        when(frame.optString("type")) {
            "chat.message" -> payload.optJSONObject("message")?.let(onMessage)
            "chat.send.result", "error" -> pending.remove(frame.optString("requestId"))?.complete(payload)
        }
    }
    override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) { disconnected(if(response?.code == 401) "登录已失效，请重新登录" else "连接已断开，正在重连") }
    override fun onClosed(webSocket: WebSocket, code: Int, reason: String) { disconnected("连接已断开，正在重连") }
    override fun onClosing(webSocket: WebSocket, code: Int, reason: String) { webSocket.close(code, reason) }
    private fun disconnected(message: String) {
        connected = false
        pending.values.forEach { it.complete(JSONObject().put("status", "unknown").put("error", JSONObject().put("code", "RESULT_UNKNOWN").put("message", "消息发送结果待确认，请使用原消息重试"))) }; pending.clear()
        if(!stopped) onState(false, message)
    }
    suspend fun send(clientId: String, text: String, replyTo: String? = null, mentionedRefs: List<String> = emptyList()): JSONObject {
        if(!connected) throw ApiFailure("CHAT_DISCONNECTED", "消息连接暂不可用，请稍后重试")
        val requestId = UUID.randomUUID().toString()
        val result = CompletableDeferred<JSONObject>()
        pending[requestId] = result
        val payload = JSONObject().put("clientMessageId", clientId).put("content", text)
        if(!replyTo.isNullOrBlank()) payload.put("replyToMessageId", replyTo)
        if(mentionedRefs.isNotEmpty()) payload.put("mentionedPlayerRefs", org.json.JSONArray(mentionedRefs))
        val sent = socket?.send(JSONObject().put("type", "chat.send").put("requestId", requestId)
            .put("sentAt", Instant.now().toString()).put("payload", payload).toString()) == true
        if(!sent) { pending.remove(requestId); throw ApiFailure("CHAT_DISCONNECTED", "消息连接暂不可用，请稍后重试") }
        return try { withTimeoutOrNull(15_000) { result.await() } ?: JSONObject().put("status", "unknown")
        } finally { pending.remove(requestId) }
    }
    fun close() { stopped = true; socket?.cancel(); socket = null; disconnected("") }
}
