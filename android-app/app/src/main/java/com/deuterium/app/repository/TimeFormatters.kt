package com.deuterium.app.repository

import java.time.OffsetDateTime
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter

private val ChinaZone: ZoneOffset = ZoneOffset.ofHours(8)
private val TimeFormatter: DateTimeFormatter = DateTimeFormatter.ofPattern("HH:mm")
private val DateTimeFormatterUtc8: DateTimeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm")

fun formatIsoTimeUtc8(value: String?): String {
    if (value.isNullOrBlank()) return ""
    return runCatching {
        OffsetDateTime.parse(value)
            .atZoneSameInstant(ChinaZone)
            .format(TimeFormatter)
    }.getOrElse { value }
}

fun formatIsoDateTimeUtc8(value: String?): String {
    if (value.isNullOrBlank()) return ""
    return runCatching {
        OffsetDateTime.parse(value)
            .atZoneSameInstant(ChinaZone)
            .format(DateTimeFormatterUtc8)
    }.getOrElse { value }
}

fun shouldNotifyFollowedChat(
    senderPlayerRef: String?,
    kind: String,
    mine: Boolean,
    followedPlayerRefs: Set<String>,
    appForeground: Boolean
): Boolean =
    !appForeground &&
        !mine &&
        kind == "public_chat" &&
        !senderPlayerRef.isNullOrBlank() &&
        followedPlayerRefs.contains(senderPlayerRef)

fun shouldNotifyWalletRecord(walletNotificationsEnabled: Boolean, appForeground: Boolean): Boolean =
    walletNotificationsEnabled && !appForeground
