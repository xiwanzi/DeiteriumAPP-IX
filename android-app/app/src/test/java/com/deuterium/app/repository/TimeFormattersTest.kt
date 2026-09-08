package com.deuterium.app.repository

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TimeFormattersTest {
    @Test
    fun formatsChatAndWalletTimesInUtc8() {
        assertEquals("20:30", formatIsoTimeUtc8("2026-04-30T12:30:00Z"))
        assertEquals("2026-04-30 20:30", formatIsoDateTimeUtc8("2026-04-30T12:30:00Z"))
    }

    @Test
    fun followedChatNotificationsOnlyFireForBackgroundFollowedPublicMessages() {
        val followed = setOf("player-target")

        assertTrue(
            shouldNotifyFollowedChat(
                senderPlayerRef = "player-target",
                kind = "public_chat",
                mine = false,
                followedPlayerRefs = followed,
                appForeground = false
            )
        )
        assertFalse(
            shouldNotifyFollowedChat(
                senderPlayerRef = "player-target",
                kind = "public_chat",
                mine = false,
                followedPlayerRefs = followed,
                appForeground = true
            )
        )
        assertFalse(
            shouldNotifyFollowedChat(
                senderPlayerRef = "player-other",
                kind = "public_chat",
                mine = false,
                followedPlayerRefs = followed,
                appForeground = false
            )
        )
        assertFalse(
            shouldNotifyFollowedChat(
                senderPlayerRef = "player-target",
                kind = "server_event",
                mine = false,
                followedPlayerRefs = followed,
                appForeground = false
            )
        )
        assertFalse(
            shouldNotifyFollowedChat(
                senderPlayerRef = "player-target",
                kind = "public_chat",
                mine = true,
                followedPlayerRefs = followed,
                appForeground = false
            )
        )
    }

    @Test
    fun walletNotificationsRespectSettingAndForegroundState() {
        assertTrue(shouldNotifyWalletRecord(walletNotificationsEnabled = true, appForeground = false))
        assertFalse(shouldNotifyWalletRecord(walletNotificationsEnabled = true, appForeground = true))
        assertFalse(shouldNotifyWalletRecord(walletNotificationsEnabled = false, appForeground = false))
    }
}
