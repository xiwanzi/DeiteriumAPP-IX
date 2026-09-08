package com.deuterium.app.repository

import com.deuterium.app.data.ChatFeedItem

data class ChatFeedMergeResult(
    val messages: List<ChatFeedItem>,
    val inserted: List<ChatFeedItem>
)

const val DefaultChatFeedMaxMessages = 500

fun mergeChatFeedItems(
    current: List<ChatFeedItem>,
    incoming: List<ChatFeedItem>,
    maxMessages: Int = DefaultChatFeedMaxMessages
): ChatFeedMergeResult {
    if (incoming.isEmpty()) return ChatFeedMergeResult(current, emptyList())

    val mergedById = LinkedHashMap<String, ChatFeedItem>()
    current.forEach { item -> mergedById[item.id] = item }

    val inserted = mutableListOf<ChatFeedItem>()
    incoming.forEach { item ->
        if (!mergedById.containsKey(item.id)) {
            mergedById[item.id] = item
            inserted.add(item)
        }
    }

    if (inserted.isEmpty()) return ChatFeedMergeResult(current, emptyList())

    val sorted = mergedById.values.sortedWith(
            compareBy<ChatFeedItem> { it.sentAt.orEmpty() }
                .thenBy { it.id }
        )
        .takeLast(maxMessages)

    return ChatFeedMergeResult(
        messages = sorted,
        inserted = inserted
    )
}

fun appendChatFeedItem(
    current: List<ChatFeedItem>,
    incoming: ChatFeedItem,
    maxMessages: Int = DefaultChatFeedMaxMessages
): ChatFeedMergeResult {
    if (current.any { it.id == incoming.id }) return ChatFeedMergeResult(current, emptyList())

    val last = current.lastOrNull()
    if (last == null || isChatFeedItemAtOrAfter(last, incoming)) {
        val trimmed = if (current.size >= maxMessages) {
            current.drop(current.size - maxMessages + 1)
        } else {
            current
        }
        return ChatFeedMergeResult(trimmed + incoming, listOf(incoming))
    }

    return mergeChatFeedItems(current, listOf(incoming), maxMessages)
}

fun isChatFeedItemAtOrAfter(previous: ChatFeedItem, incoming: ChatFeedItem): Boolean {
    val timeCompare = incoming.sentAt.orEmpty().compareTo(previous.sentAt.orEmpty())
    return timeCompare > 0 || (timeCompare == 0 && incoming.id >= previous.id)
}
