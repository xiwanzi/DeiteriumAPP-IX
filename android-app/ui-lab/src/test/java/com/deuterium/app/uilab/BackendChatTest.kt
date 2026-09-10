package com.deuterium.app.uilab

import kotlinx.coroutines.async
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

class BackendChatTest {
    @Test fun iconChangeHintsUseAnOptInHeaderAndDoNotBecomeChatMessages() {
        val server=MockWebServer();val changed=CountDownLatch(1);var messages=0
        server.enqueue(MockResponse().withWebSocketUpgrade(object:WebSocketListener(){
            override fun onOpen(webSocket:WebSocket,response:Response) {
                webSocket.send(JSONObject().put("type","app.launcher-icon.changed").put("payload",JSONObject().put("version",2)).toString())
            }
        }))
        server.start()
        val client=BackendChat(OkHttpClient(),server.url("/").toString(),"test-session",{messages++},{_,_->},{},{changed.countDown()})
        try {
            client.connect();assertTrue(changed.await(5,TimeUnit.SECONDS))
            assertEquals("1",server.takeRequest(5,TimeUnit.SECONDS)!!.getHeader("X-Deuterium-Launcher-Icon"))
            assertEquals(0,messages)
        } finally {client.close();server.close()}
    }

    @Test fun authenticatedSocketPreservesMessageIdentityAndQuote() = runBlocking {
        val server = MockWebServer()
        val opened = CountDownLatch(1)
        var received: JSONObject? = null
        server.enqueue(MockResponse().withWebSocketUpgrade(object : WebSocketListener() {
            override fun onMessage(webSocket: WebSocket, text: String) {
                val frame = JSONObject(text); received = frame.getJSONObject("payload")
                webSocket.send(JSONObject().put("type", "chat.send.result").put("requestId", frame.getString("requestId"))
                    .put("payload", JSONObject().put("status", "accepted").put("messageId", "msg-authoritative")).toString())
            }
        }))
        server.start()
        val client = BackendChat(OkHttpClient(), server.url("/").toString(), "session-private", {}, { _, _ -> }, { opened.countDown() })
        try {
            client.connect(); assertTrue(opened.await(5, TimeUnit.SECONDS))
            val ack = client.send("stable-client-key", "quoted message", "original-id", listOf("player-ref"))
            assertEquals("accepted", ack.getString("status"))
            assertEquals("stable-client-key", received!!.getString("clientMessageId"))
            assertEquals("original-id", received!!.getString("replyToMessageId"))
            assertEquals("player-ref", received!!.getJSONArray("mentionedPlayerRefs").getString(0))
            val request = server.takeRequest(5, TimeUnit.SECONDS)!!
            assertEquals("/api/v1/chat/ws", request.path)
            assertEquals("Bearer session-private", request.getHeader("Authorization"))
            assertNull(request.getHeader("X-Deuterium-Launcher-Icon"))
        } finally { client.close(); server.close() }
    }

    @Test fun serverRejectionIsNotAReceivedMessage() = runBlocking {
        val server = MockWebServer(); val opened = CountDownLatch(1); var messages = 0
        server.enqueue(MockResponse().withWebSocketUpgrade(object : WebSocketListener() {
            override fun onMessage(webSocket: WebSocket, text: String) {
                val frame = JSONObject(text)
                webSocket.send(JSONObject().put("type", "chat.send.result").put("requestId", frame.getString("requestId"))
                    .put("payload", JSONObject().put("status", "failed").put("error", JSONObject().put("code", "PLUGIN_BRIDGE_UNAVAILABLE"))).toString())
            }
        }))
        server.start()
        val client = BackendChat(OkHttpClient(), server.url("/").toString(), "token", { messages++ }, { _, _ -> }, { opened.countDown() })
        try {
            client.connect(); assertTrue(opened.await(5, TimeUnit.SECONDS))
            val result = client.send("original-key", "hello")
            assertEquals("failed", result.getString("status")); assertEquals(0, messages)
        } finally { client.close(); server.close() }
    }

    @Test fun disconnectionAfterSubmissionPreservesUnknownResult() = runBlocking {
        val server = MockWebServer(); val opened = CountDownLatch(1)
        server.enqueue(MockResponse().withWebSocketUpgrade(object : WebSocketListener() {
            override fun onMessage(webSocket: WebSocket, text: String) { webSocket.close(1001, "restart") }
        }))
        server.start()
        val client = BackendChat(OkHttpClient(), server.url("/").toString(), "token", {}, { _, _ -> }, { opened.countDown() })
        try {
            client.connect(); assertTrue(opened.await(5, TimeUnit.SECONDS))
            assertEquals("unknown", client.send("preserved-key", "hello").getString("status"))
        } finally { client.close(); server.close() }
    }
}
