package com.deuterium.app.uilab

import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.SubcomposeLayout
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.unit.dp

/**
 * List and avatars use one layout snapshot. Global-position callbacks arrive after placement,
 * so feeding them back into state makes avatars chase the rows by a frame at reuse boundaries.
 * Subcompose visible avatars in this measure pass, without a second scroll animation.
 */
@Composable
internal fun BoxScope.ChatGroupAvatars(
    list: LazyListState,
    groups: List<ChatMessageGroup>,
    avatar: @Composable (ChatLine, Modifier) -> Unit
) {
    val incoming = remember(groups) { groups.filterNot { it.last.mine } }
    val groupsByMessage = remember(incoming) { buildMap {
        incoming.forEach { group -> group.ids.forEach { put(it, group) } }
    } }
    val groupIndices = remember(incoming) { incoming.mapIndexed { index, group -> group.first.id to index }.toMap() }
    SubcomposeLayout(Modifier.fillMaxSize().clipToBounds()) { constraints ->
        fun rows(): Pair<Float, List<Triple<Long, ChatMessageGroup, Pair<Float, Float>>>> {
            val info = list.layoutInfo
            // Content padding is the sticky anchor, not a clip: messages also draw under the dock.
            val viewportBottom = info.viewportSize.height.toFloat()
            return viewportBottom to info.visibleItemsInfo.mapNotNull { item ->
                val id = item.key as? Long ?: return@mapNotNull null
                val group = groupsByMessage[id] ?: return@mapNotNull null
                val top = chatItemTop(item.offset, item.size, info.viewportStartOffset,
                    info.viewportEndOffset, info.reverseLayout)
                Triple(id, group, top to top + item.size)
            }
        }
        val (viewportBottom, visible) = rows()
        val activeGroups = visible.filter { it.third.second > 0f && it.third.first < viewportBottom }
            .map { it.second }.distinctBy { it.first.id }
        // Keep neighbours ready while LazyColumn performs a placement-only scroll this frame.
        val slots = activeGroups.flatMap { group ->
            val index = groupIndices.getValue(group.first.id)
            incoming.subList((index - 2).coerceAtLeast(0), (index + 3).coerceAtMost(incoming.size))
        }.distinctBy { it.first.id }
        val size = 42.dp.roundToPx()
        val avatars = slots.map { group ->
            val placeable = subcompose(group.first.id) {
                avatar(group.last, Modifier.graphicsLayer {
                    // A layer property reads the latest scroll directly, including the list's
                    // placement-only fast path. It also moves the hit target with the portrait.
                    val info = list.layoutInfo
                    val edge = if (info.reverseLayout) info.viewportEndOffset.toFloat()
                        else (info.viewportSize.height - info.afterContentPadding).toFloat()
                    val drawingBottom = info.viewportSize.height.toFloat()
                    var top = Float.NEGATIVE_INFINITY
                    var bottom = Float.POSITIVE_INFINITY
                    var visible = false
                    info.visibleItemsInfo.forEach { item ->
                        val id = item.key as? Long
                        if (id != null && groupsByMessage[id]?.first?.id == group.first.id) {
                            val y = chatItemTop(item.offset, item.size, info.viewportStartOffset,
                                info.viewportEndOffset, info.reverseLayout)
                            if (id == group.first.id) top = y
                            if (id == group.last.id) bottom = y + item.size
                            if (y < drawingBottom && y + item.size > 0f) visible = true
                        }
                    }
                    alpha = if (visible) 1f else 0f
                    translationY = if (visible) groupAvatarTop(top, bottom, edge, size.toFloat()) else -size * 2f
                }.testTag("chat-avatar-${group.first.id}"))
            }.single().measure(Constraints.fixed(size, size))
            placeable
        }
        layout(constraints.maxWidth, constraints.maxHeight) {
            avatars.forEach { it.placeRelative(20.dp.roundToPx(), 0) }
        }
    }
}
