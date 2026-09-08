package com.deuterium.app

import android.Manifest
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import com.deuterium.app.data.ChatMessage
import com.deuterium.app.data.WalletRecord
import com.deuterium.app.repository.formatIsoDateTimeUtc8
import kotlin.math.absoluteValue

class AppNotifier(private val context: Context) {
    private val manager = context.getSystemService(NotificationManager::class.java)

    init {
        createChannels()
    }

    fun canNotify(): Boolean {
        return Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
            context.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED
    }

    fun notifyFollowedChat(message: ChatMessage) {
        if (!canNotify()) return
        manager.notify(
            notificationId(message.messageId),
            notification(
                channelId = CHANNEL_FOLLOWED_CHAT,
                title = message.sender.gameId,
                text = message.content
            )
        )
    }

    fun notifyWalletRecord(record: WalletRecord) {
        if (!canNotify()) return
        val direction = if (record.direction == "income") "收入" else "支出"
        val sign = if (record.direction == "income") "+" else "-"
        val time = formatIsoDateTimeUtc8(record.occurredAt)
        manager.notify(
            notificationId(record.recordId),
            notification(
                channelId = CHANNEL_WALLET,
                title = "钱包变动：$direction ${sign}${record.amount} 信用点",
                text = listOf(record.otherPlayer.gameId, time, record.note)
                    .mapNotNull { it?.takeIf { value -> value.isNotBlank() } }
                    .joinToString(" · ")
            )
        )
    }

    fun notifyMention(message: ChatMessage) {
        if (!canNotify()) return
        manager.notify(
            notificationId("mention-${message.messageId}"),
            notification(
                channelId = CHANNEL_MENTION,
                title = "${message.sender.gameId} @ 了你",
                text = message.content
            )
        )
    }

    private fun notification(channelId: String, title: String, text: String): Notification {
        return Notification.Builder(context, channelId)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentTitle(title)
            .setContentText(text)
            .setStyle(Notification.BigTextStyle().bigText(text))
            .setContentIntent(openAppIntent())
            .setAutoCancel(true)
            .build()
    }

    private fun openAppIntent(): PendingIntent {
        val intent = Intent(context, MainActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP)
        return PendingIntent.getActivity(
            context,
            0,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
    }

    private fun createChannels() {
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_FOLLOWED_CHAT, "关心玩家消息", NotificationManager.IMPORTANCE_DEFAULT)
        )
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_WALLET, "钱包变动", NotificationManager.IMPORTANCE_DEFAULT)
        )
        manager.createNotificationChannel(
            NotificationChannel(CHANNEL_MENTION, "@ 提醒", NotificationManager.IMPORTANCE_DEFAULT)
        )
    }

    private fun notificationId(value: String): Int =
        value.hashCode().takeIf { it != Int.MIN_VALUE }?.absoluteValue ?: 0

    private companion object {
        const val CHANNEL_FOLLOWED_CHAT = "followed_chat"
        const val CHANNEL_WALLET = "wallet_changes"
        const val CHANNEL_MENTION = "chat_mentions"
    }
}
