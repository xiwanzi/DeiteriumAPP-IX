package com.deuterium.app.repository

import com.deuterium.app.data.ChatMentionEventData
import com.deuterium.app.data.ChatMessage
import com.deuterium.app.data.OnlinePlayer
import com.deuterium.app.data.ServerEvent
import com.deuterium.app.data.WalletRecordEventData
import com.google.gson.Gson
import com.google.gson.JsonObject

internal sealed interface ChatSocketEvent {
    data class Message(val message: ChatMessage) : ChatSocketEvent
    data class Mention(val data: ChatMentionEventData) : ChatSocketEvent
    data class Server(val event: ServerEvent) : ChatSocketEvent
    data class Presence(val onlineCount: Int?, val players: List<OnlinePlayer>?) : ChatSocketEvent
    data class SendResult(
        val clientMessageId: String,
        val status: String,
        val messageId: String?,
        val errorMessage: String?
    ) : ChatSocketEvent
    data class ServiceError(val message: String) : ChatSocketEvent
    data class WalletRecord(val data: WalletRecordEventData) : ChatSocketEvent
}

internal class ChatSocketEventParser(
    private val gson: Gson = Gson()
) {
    fun parse(text: String): ChatSocketEvent? = runCatching {
        val envelope = gson.fromJson(text, JsonObject::class.java) ?: return@runCatching null
        when (envelope.get("type")?.asString) {
            "chat.message" -> {
                val message = gson.fromJson(envelope.payloadObject("message"), ChatMessage::class.java)
                    ?: return@runCatching null
                require(message.messageId.isNotBlank() && message.sender.playerRef.isNotBlank())
                ChatSocketEvent.Message(message)
            }
            "chat.mention.event" -> {
                val data = gson.fromJson(envelope.getAsJsonObject("payload"), ChatMentionEventData::class.java)
                    ?: return@runCatching null
                require(data.message.messageId.isNotBlank() && data.message.sender.playerRef.isNotBlank())
                ChatSocketEvent.Mention(data)
            }
            "server.event" -> {
                val event = gson.fromJson(envelope.payloadObject("event"), ServerEvent::class.java)
                    ?: return@runCatching null
                require(event.eventId.isNotBlank() && event.occurredAt.isNotBlank())
                ChatSocketEvent.Server(event)
            }
            "presence.update" -> {
                val payload = envelope.getAsJsonObject("payload") ?: return@runCatching null
                val players = payload.getAsJsonArray("players")?.map { item ->
                    gson.fromJson(item, OnlinePlayer::class.java).also {
                        require(it.playerRef.isNotBlank() && it.gameId.isNotBlank())
                    }
                }
                ChatSocketEvent.Presence(payload.get("onlineCount")?.asInt, players)
            }
            "chat.send.result" -> {
                val payload = envelope.getAsJsonObject("payload") ?: return@runCatching null
                val clientMessageId = payload.get("clientMessageId")?.asString.orEmpty()
                val status = payload.get("status")?.asString.orEmpty()
                require(clientMessageId.isNotBlank() && status in setOf("accepted", "failed"))
                ChatSocketEvent.SendResult(
                    clientMessageId = clientMessageId,
                    status = status,
                    messageId = payload.get("messageId")?.asString,
                    errorMessage = payload.getAsJsonObject("error")?.get("message")?.asString
                )
            }
            "error" -> {
                val message = envelope.getAsJsonObject("payload")?.get("message")?.asString
                    ?.takeIf { it.isNotBlank() }
                    ?: "聊天服务返回错误。"
                ChatSocketEvent.ServiceError(message)
            }
            "wallet.record.event" -> {
                val data = gson.fromJson(envelope.getAsJsonObject("payload"), WalletRecordEventData::class.java)
                    ?: return@runCatching null
                require(data.record.recordId.isNotBlank())
                ChatSocketEvent.WalletRecord(data)
            }
            else -> null
        }
    }.getOrNull()

    private fun JsonObject.payloadObject(key: String): JsonObject? =
        getAsJsonObject("payload")?.getAsJsonObject(key)
}
