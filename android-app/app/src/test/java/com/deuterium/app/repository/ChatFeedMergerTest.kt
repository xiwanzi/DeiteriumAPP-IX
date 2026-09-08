package com.deuterium.app.repository

import com.deuterium.app.data.ChatFeedItem
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ChatFeedMergerTest {
    @Test
    fun appendsNewestLiveMessageWithoutReorderingExistingItems() {
        val current = listOf(
            item("msg-1", "2026-04-29T12:00:00Z"),
            item("msg-2", "2026-04-29T12:01:00Z")
        )

        val result = appendChatFeedItem(current, item("msg-3", "2026-04-29T12:02:00Z"))

        assertEquals(listOf("msg-1", "msg-2", "msg-3"), result.messages.map { it.id })
        assertEquals(listOf("msg-3"), result.inserted.map { it.id })
    }

    @Test
    fun liveAppendSkipsDuplicateMessageIds() {
        val current = listOf(item("msg-1", "2026-04-29T12:00:00Z"))

        val result = appendChatFeedItem(current, item("msg-1", "2026-04-29T12:00:00Z"))

        assertEquals(listOf("msg-1"), result.messages.map { it.id })
        assertTrue(result.inserted.isEmpty())
    }

    @Test
    fun liveAppendFallsBackToTimeOrderingForOutOfOrderMessage() {
        val current = listOf(
            item("msg-1", "2026-04-29T12:00:00Z"),
            item("msg-3", "2026-04-29T12:02:00Z")
        )

        val result = appendChatFeedItem(current, item("msg-2", "2026-04-29T12:01:00Z"))

        assertEquals(listOf("msg-1", "msg-2", "msg-3"), result.messages.map { it.id })
        assertEquals(listOf("msg-2"), result.inserted.map { it.id })
    }

    @Test
    fun liveAppendCapsMessagesToNewestItems() {
        val current = (1..5).map { index ->
            item("msg-$index", "2026-04-29T12:0${index}:00Z")
        }

        val result = appendChatFeedItem(current, item("msg-6", "2026-04-29T12:06:00Z"), maxMessages = 3)

        assertEquals(listOf("msg-4", "msg-5", "msg-6"), result.messages.map { it.id })
        assertEquals(listOf("msg-6"), result.inserted.map { it.id })
    }

    @Test
    fun insertsNewMessagesAndKeepsTimeOrder() {
        val current = listOf(item("msg-1", "2026-04-29T12:00:00Z"))
        val incoming = listOf(
            item("msg-3", "2026-04-29T12:02:00Z"),
            item("msg-2", "2026-04-29T12:01:00Z")
        )

        val result = mergeChatFeedItems(current, incoming)

        assertEquals(listOf("msg-1", "msg-2", "msg-3"), result.messages.map { it.id })
        assertEquals(listOf("msg-3", "msg-2"), result.inserted.map { it.id })
    }

    @Test
    fun skipsDuplicateMessageIds() {
        val current = listOf(item("msg-1", "2026-04-29T12:00:00Z"))
        val incoming = listOf(item("msg-1", "2026-04-29T12:00:00Z"))

        val result = mergeChatFeedItems(current, incoming)

        assertEquals(listOf("msg-1"), result.messages.map { it.id })
        assertTrue(result.inserted.isEmpty())
    }

    @Test
    fun recentHttpCatchupDoesNotDeleteServerEvents() {
        val current = listOf(
            item("msg-1", "2026-04-29T12:00:00Z"),
            item("evt-1", "2026-04-29T12:01:00Z", event = true)
        )
        val incoming = listOf(item("msg-2", "2026-04-29T12:02:00Z"))

        val result = mergeChatFeedItems(current, incoming)

        assertEquals(listOf("msg-1", "evt-1", "msg-2"), result.messages.map { it.id })
        assertEquals(listOf("msg-2"), result.inserted.map { it.id })
    }

    @Test
    fun capsMergedMessagesToNewestItems() {
        val current = (1..5).map { index ->
            item("msg-$index", "2026-04-29T12:0${index}:00Z")
        }
        val incoming = listOf(item("msg-6", "2026-04-29T12:06:00Z"))

        val result = mergeChatFeedItems(current, incoming, maxMessages = 3)

        assertEquals(listOf("msg-4", "msg-5", "msg-6"), result.messages.map { it.id })
        assertEquals(listOf("msg-6"), result.inserted.map { it.id })
    }

    private fun item(id: String, sentAt: String, event: Boolean = false): ChatFeedItem =
        ChatFeedItem(
            id = id,
            sender = if (event) "服务器" else "Lantern",
            content = id,
            time = "12:00",
            event = event,
            sentAt = sentAt
        )
}
