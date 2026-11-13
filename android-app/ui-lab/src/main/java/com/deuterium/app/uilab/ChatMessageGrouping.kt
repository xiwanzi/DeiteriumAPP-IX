package com.deuterium.app.uilab

import java.time.Instant
import java.time.ZoneId
import java.util.Locale

internal data class ChatMessageGroup(val first: ChatLine, val last: ChatLine, val ids: List<Long>)

/** Display grouping only: server order, message keys, reply targets and sending are untouched. */
internal fun groupChatMessages(lines: List<ChatLine>, zone: ZoneId = ZoneId.systemDefault()): List<ChatMessageGroup> {
    fun identity(line: ChatLine) = line.playerRef.takeIf { it.isNotBlank() }
        ?: "name:${line.name.lowercase(Locale.ROOT)}"
    fun joins(a: ChatLine, b: ChatLine): Boolean {
        if (a.mine != b.mine || identity(a) != identity(b)) return false
        if (a.serverAt <= 0 || b.serverAt <= 0 || b.serverAt - a.serverAt !in 0L..300_000L) return false
        if (a.forwarded != null || b.forwarded != null) return false
        if (a.aiStatus != null && a.aiStatus != "completed" || b.aiStatus != null && b.aiStatus != "completed") return false
        return Instant.ofEpochMilli(a.serverAt).atZone(zone).toLocalDate() ==
            Instant.ofEpochMilli(b.serverAt).atZone(zone).toLocalDate()
    }
    val groups = ArrayList<ChatMessageGroup>()
    var start = 0
    for (index in lines.indices) {
        if (index == lines.lastIndex || !joins(lines[index], lines[index + 1])) {
            groups += ChatMessageGroup(lines[start], lines[index], lines.subList(start, index + 1).map { it.id })
            start = index + 1
        }
    }
    return groups
}

/** Bottom-attached, confined to its own group. Adjacent group boundaries create the push-off. */
internal fun groupAvatarTop(groupTop: Float, groupBottom: Float, viewportBottom: Float, avatarHeight: Float): Float =
    maxOf(groupTop, minOf(groupBottom, viewportBottom) - avatarHeight)
        .coerceAtMost(groupBottom - avatarHeight)

/** LazyListItemInfo precedes reverse placement and content padding. */
internal fun chatItemTop(offset: Int, size: Int, viewportStart: Int, viewportEnd: Int, reverse: Boolean): Float =
    (if (reverse) viewportEnd - offset - size else offset - viewportStart).toFloat()
