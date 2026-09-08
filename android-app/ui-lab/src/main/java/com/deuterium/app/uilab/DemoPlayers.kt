package com.deuterium.app.uilab

import androidx.compose.runtime.mutableStateListOf

data class PlayerProfile(val name: String, val qq: String = "", val bio: String = "", val online: Boolean = false, val lastSeen: String = "暂无记录", val playerRef: String = "",val avatar:String?=null)
/** Session-scoped server directory. Never populated with sample players. */
val Players = mutableStateListOf<PlayerProfile>()
fun searchPlayers(query: String): List<PlayerProfile> = Players.filter {
    query.isBlank() || it.name.contains(query.trim().removePrefix("@"), true) || it.qq.contains(query.trim())
}

data class DemoNotice(val id: Long, val title: String, val body: String, val route: String = "public",val topic:NotificationTopic=when{route=="wallet"->NotificationTopic.Wallet;route.startsWith("commission:")->NotificationTopic.Commissions;route.startsWith("order:")||route.startsWith("refund:")->NotificationTopic.Market;route=="announcement"->NotificationTopic.Announcements;route=="about"->NotificationTopic.Updates;route.startsWith("dm:")->NotificationTopic.Direct;else->NotificationTopic.Mentions})
