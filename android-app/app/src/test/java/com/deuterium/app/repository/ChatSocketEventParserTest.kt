package com.deuterium.app.repository

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ChatSocketEventParserTest {
    private val parser = ChatSocketEventParser()

    @Test
    fun malformedJsonIsIgnored() {
        assertNull(parser.parse("{"))
    }

    @Test
    fun incompleteMessageIsIgnored() {
        assertNull(parser.parse("""{"type":"chat.message","payload":{}}"""))
    }

    @Test
    fun wrongPresenceShapeIsIgnored() {
        assertNull(
            parser.parse(
                """{"type":"presence.update","payload":{"onlineCount":"many","players":{}}}"""
            )
        )
    }

    @Test
    fun validPresenceIsParsed() {
        val event = parser.parse(
            """
            {
              "type": "presence.update",
              "payload": {
                "onlineCount": 1,
                "players": [
                  {"playerRef":"player_1","gameId":"Steve","registered":true}
                ]
              }
            }
            """.trimIndent()
        )

        assertTrue(event is ChatSocketEvent.Presence)
        event as ChatSocketEvent.Presence
        assertEquals(1, event.onlineCount)
        assertEquals("player_1", event.players?.single()?.playerRef)
    }

    @Test
    fun pendingTransferMatchesOnlyTheSameIntent() {
        val pending = PendingTransferRequest(
            clientRequestId = "android-request-1",
            recipientPlayerRef = "player_1",
            amount = "12.50",
            note = "test"
        )

        assertTrue(pending.matches("player_1", "12.50", "test"))
        assertTrue(!pending.matches("player_2", "12.50", "test"))
        assertTrue(!pending.matches("player_1", "12.51", "test"))
    }
}
