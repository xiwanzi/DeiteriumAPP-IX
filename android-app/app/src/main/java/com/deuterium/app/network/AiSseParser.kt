package com.deuterium.app.network

import com.deuterium.app.data.AiStreamDeltaData
import com.deuterium.app.data.AiStreamDoneData
import com.deuterium.app.data.AiStreamMetaData
import com.deuterium.app.data.AiStreamStatusData
import com.deuterium.app.data.AiKnowledgeSource
import com.deuterium.app.data.ApiErrorBody
import com.google.gson.Gson
import com.google.gson.JsonObject
import com.google.gson.reflect.TypeToken

sealed class AiStreamEvent {
    data class Meta(val data: AiStreamMetaData) : AiStreamEvent()
    data class Status(val data: AiStreamStatusData) : AiStreamEvent()
    data class Sources(val sources: List<AiKnowledgeSource>) : AiStreamEvent()
    data class Delta(val text: String) : AiStreamEvent()
    data class Done(val data: AiStreamDoneData) : AiStreamEvent()
    data class Error(val error: ApiErrorBody) : AiStreamEvent()
}

class AiSseParser(private val gson: Gson = Gson()) {
    private var eventName: String? = null
    private val dataLines = mutableListOf<String>()

    fun accept(line: String): AiStreamEvent? {
        val normalized = line.removeSuffix("\r")
        if (normalized.isEmpty()) return emit()
        if (normalized.startsWith(":")) return null

        val separator = normalized.indexOf(':')
        val field = if (separator >= 0) normalized.substring(0, separator) else normalized
        val rawValue = if (separator >= 0) normalized.substring(separator + 1) else ""
        val value = rawValue.removePrefix(" ")

        when (field) {
            "event" -> eventName = value
            "data" -> dataLines += value
        }
        return null
    }

    fun finish(): AiStreamEvent? {
        if (eventName == null && dataLines.isEmpty()) return null
        return emit()
    }

    private fun emit(): AiStreamEvent? {
        val event = eventName ?: "message"
        val data = dataLines.joinToString("\n")
        eventName = null
        dataLines.clear()
        if (data.isBlank()) return null

        return runCatching {
            when (event) {
                "meta" -> AiStreamEvent.Meta(gson.fromJson(data, AiStreamMetaData::class.java))
                "status" -> AiStreamEvent.Status(gson.fromJson(data, AiStreamStatusData::class.java))
                "sources" -> AiStreamEvent.Sources(parseSources(data))
                "delta" -> AiStreamEvent.Delta(parseDelta(data))
                "done" -> AiStreamEvent.Done(gson.fromJson(data, AiStreamDoneData::class.java))
                "error" -> AiStreamEvent.Error(parseError(data))
                else -> null
            }
        }.getOrNull()
    }

    private fun parseDelta(data: String): String {
        val json = runCatching { gson.fromJson(data, JsonObject::class.java) }.getOrNull()
        if (json == null || !json.isJsonObject) return data
        val payload = gson.fromJson(json, AiStreamDeltaData::class.java)
        return payload.delta ?: payload.text ?: payload.content ?: ""
    }

    private fun parseError(data: String): ApiErrorBody {
        val json = runCatching { gson.fromJson(data, JsonObject::class.java) }.getOrNull()
        val body = json?.getAsJsonObject("error") ?: json
        return if (body != null) {
            ApiErrorBody(
                code = body.get("code")?.asString,
                message = body.get("message")?.asString,
                retryAfterSeconds = body.get("retryAfterSeconds")?.asLong
            )
        } else {
            ApiErrorBody(message = "AI 服务返回错误。")
        }
    }

    private fun parseSources(data: String): List<AiKnowledgeSource> {
        val type = object : TypeToken<List<AiKnowledgeSource>>() {}.type
        return runCatching { gson.fromJson<List<AiKnowledgeSource>>(data, type) }
            .getOrDefault(emptyList())
    }
}
