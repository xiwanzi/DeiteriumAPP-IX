package com.deuterium.app.uilab

import org.junit.Assert.*
import org.junit.Test
import java.time.Instant
import java.time.ZoneId

class ChatMessageGroupingTest {
    @Test fun listCoordinatesAccountForReverseLayoutAndBothContentInsets() {
        // A 1000 px list with 120 px bottom padding has a reverse viewport end of 880.
        assertEquals(780f, chatItemTop(0, 100, -120, 880, true), .01f)
        assertEquals(817f, chatItemTop(-37, 100, -120, 880, true), .01f)
        assertEquals(743f, chatItemTop(37, 100, -120, 880, true), .01f)
        assertEquals(72f, chatItemTop(0, 100, -72, 928, false), .01f)
    }

    private val start = Instant.parse("2026-11-13T10:00:00Z").toEpochMilli()
    private val utc = ZoneId.of("UTC")
    private fun line(id: Long, who: String = "player_a", time: Long = start + id * 1000, mine: Boolean = false) =
        ChatLine(id, "A", "message $id", mine, serverAt = time, playerRef = who)

    @Test fun stableIdentityGroupsRenamedSenderButNotSameDisplayName() {
        val groups = groupChatMessages(listOf(line(1), line(2).copy(name = "Renamed"), line(3, "player_b")), utc)
        assertEquals(listOf(listOf(1L, 2L), listOf(3L)), groups.map { it.ids })
        assertEquals("Renamed", groups.first().last.name)
    }

    @Test fun timeDirectionAndCalendarBoundarySeparateGroups() {
        assertEquals(2, groupChatMessages(listOf(line(1), line(2, mine = true)), utc).size)
        assertEquals(2, groupChatMessages(listOf(line(1), line(2, time = start + 301_001)), utc).size)
        assertEquals(1, groupChatMessages(listOf(line(1), line(2, time = start + 301_000)), utc).size)
        val midnight = Instant.parse("2026-11-14T00:00:00Z").toEpochMilli()
        assertEquals(2, groupChatMessages(listOf(line(1, time = midnight - 1000), line(2, time = midnight)), utc).size)
    }

    @Test fun unresolvedTimesForwardedAndStreamingMessagesRemainIndependent() {
        val samples = listOf(line(2, time = 0), line(2).copy(forwarded = ChatReply(9, "Other", "x")),
            line(2).copy(aiStatus = "streaming"))
        samples.forEach { assertEquals(2, groupChatMessages(listOf(line(1), it), utc).size) }
        assertEquals(0, groupChatMessages(emptyList(), utc).size)
    }

    @Test fun deletingMiddleSenderMergesOnlyTheRemainingAdjacentGroup() {
        val lines = listOf(line(1), line(2, "player_b"), line(3))
        assertEquals(3, groupChatMessages(lines, utc).size)
        assertEquals(listOf(1L, 3L), groupChatMessages(lines.filter { it.id != 2L }, utc).single().ids)
    }

    @Test fun avatarPinsToVisibleBottomAndIsPushedWithinItsOwnGroup() {
        assertEquals(558f, groupAvatarTop(100f, 1000f, 600f, 42f), .01f)
        assertEquals(358f, groupAvatarTop(100f, 400f, 600f, 42f), .01f)
        assertEquals(580f, groupAvatarTop(580f, 900f, 600f, 42f), .01f)
        assertEquals(-32f, groupAvatarTop(-100f, 10f, 600f, 42f), .01f)
    }

    @Test fun groupHeadPushesAvatarContinuouslyThroughAndBelowInputAnchor() {
        // The anchor is 600, while the drawing viewport extends to 800.
        val positions = (550..810).map { groupAvatarTop(it.toFloat(), it + 300f, 600f, 42f) }
        assertEquals(601f, groupAvatarTop(601f, 1000f, 600f, 42f), .01f)
        assertEquals(790f, groupAvatarTop(790f, 1000f, 600f, 42f), .01f)
        assertTrue(positions.zipWithNext().all { (a,b) -> b-a in 0f..1f })
    }
}
