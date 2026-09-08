package com.deuterium.app.repository

import android.os.SystemClock
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.deuterium.app.data.AiChatStreamRequest
import com.deuterium.app.data.AiConversation
import com.deuterium.app.data.AiConversationResetRequest
import com.deuterium.app.data.AiMeData
import com.deuterium.app.data.AiMessage
import com.deuterium.app.data.AiKnowledgeSource
import com.deuterium.app.data.AiPlan
import com.deuterium.app.data.AiPurchase
import com.deuterium.app.data.AiPurchaseRequest
import com.deuterium.app.data.AiQuota
import com.deuterium.app.data.ApiErrorBody
import com.deuterium.app.data.RepoResult
import com.deuterium.app.network.AiStreamEvent
import com.deuterium.app.network.ApiClient
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.withContext
import java.time.Instant
import java.util.UUID

data class AiUiMessage(
    val id: String,
    val role: String,
    val content: String,
    val time: String,
    val streaming: Boolean = false,
    val renderMarkdown: Boolean = role == "assistant" && !streaming,
    val statusText: String? = null,
    val sources: List<AiKnowledgeSource> = emptyList()
)

class AiRepository(
    private val apiClient: ApiClient,
    private val sessionStore: SessionStore,
    private val onUnauthorized: () -> Unit = {}
) {
    private var lastInitialLoadAt = 0L
    val messages = mutableStateListOf<AiUiMessage>()
    val plans = mutableStateListOf<AiPlan>()

    var assistantName by mutableStateOf("Deuterium AI")
        private set
    var currentPlan by mutableStateOf<AiPlan?>(null)
        private set
    var quota by mutableStateOf<AiQuota?>(null)
        private set
    var conversation by mutableStateOf<AiConversation?>(null)
        private set
    var loading by mutableStateOf(false)
        private set
    var activeRequestId by mutableStateOf<String?>(null)
        private set
    val sending: Boolean
        get() = activeRequestId != null
    var purchasing by mutableStateOf(false)
        private set
    var message by mutableStateOf<String?>(null)
        private set

    suspend fun loadInitial(force: Boolean = false): RepoResult<Unit> {
        if (!force && lastInitialLoadAt > 0L && SystemClock.elapsedRealtime() - lastInitialLoadAt < InitialLoadFreshMillis) {
            return RepoResult.Success(Unit)
        }
        loading = true
        message = null
        try {
            val meResult = apiClient.request { apiClient.backend.aiMe() }
            if (meResult is RepoResult.Success) applyMe(meResult.value)
            if (meResult is RepoResult.Error && handleUnauthorized(meResult)) return meResult

            val messagesResult = apiClient.request { apiClient.backend.aiMessages(limit = 50) }
            if (messagesResult is RepoResult.Success) replaceMessages(messagesResult.value.messages)
            if (messagesResult is RepoResult.Error && handleUnauthorized(messagesResult)) return messagesResult

            val plansResult = loadPlans()
            if (meResult is RepoResult.Error) {
                message = friendlyAiError(meResult.message, meResult.code, meResult.retryAfterSeconds)
                return meResult.copy(message = message ?: meResult.message)
            }
            lastInitialLoadAt = SystemClock.elapsedRealtime()
            return if (plansResult is RepoResult.Error) RepoResult.Success(Unit) else RepoResult.Success(Unit)
        } finally {
            loading = false
        }
    }

    suspend fun loadPlans(): RepoResult<List<AiPlan>> {
        return when (val result = apiClient.request { apiClient.backend.aiPlans() }) {
            is RepoResult.Success -> {
                plans.clear()
                plans.addAll(result.value.plans.filter { it.enabled })
                RepoResult.Success(plans.toList())
            }
            is RepoResult.Error -> {
                if (handleUnauthorized(result)) return result
                if (plans.isEmpty()) message = friendlyAiError(result.message, result.code, result.retryAfterSeconds)
                result
            }
        }
    }

    suspend fun resetConversation(): RepoResult<Unit> {
        if (sending) return RepoResult.Error("AI 正在回复，请稍后再新建对话。")
        message = null
        val request = AiConversationResetRequest(clientRequestId = "android-ai-reset-${UUID.randomUUID()}")
        return when (val result = apiClient.request { apiClient.backend.resetAiConversation(request) }) {
            is RepoResult.Success -> {
                conversation = result.value.conversation
                quota = result.value.quota ?: quota
                replaceMessages(result.value.messages)
                message = "已开启新对话。"
                RepoResult.Success(Unit)
            }
            is RepoResult.Error -> {
                if (handleUnauthorized(result)) return result
                message = friendlyAiError(result.message, result.code, result.retryAfterSeconds)
                result.copy(message = message ?: result.message)
            }
        }
    }

    suspend fun sendMessage(content: String): RepoResult<Unit> {
        val trimmed = content.trim()
        if (trimmed == "/new") return resetConversation()
        val localError = validateAiMessage(trimmed)
        if (localError != null) return RepoResult.Error(localError)
        if (loading) return RepoResult.Error("AI 正在同步消息，请稍后再试。")
        if (sending) return RepoResult.Error("上一条 AI 回复还在生成中。")

        message = null
        val clientMessageId = "android-ai-msg-${UUID.randomUUID()}"
        activeRequestId = clientMessageId
        messages += AiUiMessage(
            id = clientMessageId,
            role = "user",
            content = trimmed,
            time = formatIsoTimeUtc8(Instant.now().toString())
        )
        var assistantIndex = messages.size
        var assistantMessageId = "android-ai-assistant-${UUID.randomUUID()}"
        messages += AiUiMessage(
            id = assistantMessageId,
            role = "assistant",
            content = "",
            time = "生成中",
            streaming = true,
            renderMarkdown = false,
            statusText = "正在思考"
        )

        val deltaBuffer = StringBuilder()
        var lastDeltaFlushAt = 0L
        fun flushDeltaBuffer(force: Boolean = false) {
            if (deltaBuffer.isEmpty()) return
            val now = SystemClock.elapsedRealtime()
            if (!force && deltaBuffer.length < DeltaFlushMinChars && now - lastDeltaFlushAt < DeltaFlushIntervalMillis) return
            val chunk = deltaBuffer.toString()
            deltaBuffer.clear()
            lastDeltaFlushAt = now
            updateAssistantAt(assistantIndex) { current ->
                current.copy(content = current.content + chunk)
            }
        }

        var completed = false
        return try {
            var streamError: RepoResult.Error? = null
            val streamResult = apiClient.streamAiChat(
                AiChatStreamRequest(clientMessageId = clientMessageId, content = trimmed)
            ) { event ->
                withContext(Dispatchers.Main) {
                    when (event) {
                        is AiStreamEvent.Meta -> {
                            quota = event.data.quota ?: quota
                            conversation = AiConversation(conversationId = event.data.conversationId ?: conversation?.conversationId)
                            event.data.assistantMessageId?.takeIf { it.isNotBlank() }?.let { serverId ->
                                assistantMessageId = serverId
                                updateAssistantAt(assistantIndex) { it.copy(id = serverId) }
                            }
                        }
                        is AiStreamEvent.Status -> {
                            val statusText = when (event.data.status) {
                                "searching_knowledge" -> "正在查阅知识库"
                                "thinking" -> "正在思考"
                                else -> event.data.message ?: event.data.status
                            }
                            updateAssistantAt(assistantIndex) { it.copy(statusText = statusText) }
                        }
                        is AiStreamEvent.Sources -> {
                            updateAssistantAt(assistantIndex) {
                                val mergedSources = (it.sources + event.sources).distinctBy { source ->
                                    listOf(source.sourceUrl, source.headingPath, source.title).joinToString("|")
                                }
                                it.copy(sources = mergedSources, statusText = "已找到 ${mergedSources.size} 条参考")
                            }
                        }
                        is AiStreamEvent.Delta -> {
                            deltaBuffer.append(event.text)
                            flushDeltaBuffer()
                        }
                        is AiStreamEvent.Done -> {
                            flushDeltaBuffer(force = true)
                            completed = true
                            quota = event.data.quota ?: quota
                            val finalMessage = event.data.message ?: event.data.assistantMessage
                            if (finalMessage != null) {
                                updateAssistantAt(assistantIndex) {
                                    finalMessage.toUiMessage(streaming = false).copy(
                                        renderMarkdown = true,
                                        sources = it.sources
                                    )
                                }
                            } else {
                                updateAssistantAt(assistantIndex) { it.copy(streaming = false, renderMarkdown = true, time = "刚刚", statusText = null) }
                            }
                        }
                        is AiStreamEvent.Error -> {
                            flushDeltaBuffer(force = true)
                            completed = true
                            val errorMessage = friendlyAiError(event.error)
                            streamError = RepoResult.Error(
                                message = errorMessage,
                                code = event.error.code,
                                retryAfterSeconds = event.error.retryAfterSeconds
                            )
                            message = errorMessage
                            updateAssistantAt(assistantIndex) { current ->
                                current.copy(
                                    content = if (current.content.isBlank()) {
                                        "回复失败：$errorMessage"
                                    } else {
                                        current.content + "\n\n回复中断：$errorMessage"
                                    },
                                    streaming = false,
                                    renderMarkdown = false,
                                    statusText = null,
                                    time = "刚刚"
                                )
                            }
                        }
                    }
                    assistantIndex = messages.indexOfFirst { it.id == assistantMessageId }.takeIf { it >= 0 } ?: assistantIndex
                }
            }

            withContext(Dispatchers.Main) {
                flushDeltaBuffer(force = true)
            }

            val finalResult = streamError ?: streamResult
            if (finalResult is RepoResult.Error) {
                if (handleUnauthorized(finalResult)) return finalResult
                val errorMessage = friendlyAiError(finalResult.message, finalResult.code, finalResult.retryAfterSeconds)
                message = errorMessage
                if (streamError == null) {
                    updateAssistantAt(assistantIndex) { current ->
                        current.copy(
                            content = if (current.content.isBlank()) {
                                "回复失败：$errorMessage"
                            } else {
                                current.content + "\n\n回复中断：$errorMessage"
                            },
                            streaming = false,
                            renderMarkdown = false,
                            statusText = null,
                            time = "刚刚"
                        )
                    }
                }
                return finalResult.copy(message = errorMessage)
            }
            loadStatusSilently()
            completed = true
            RepoResult.Success(Unit)
        } catch (e: CancellationException) {
            withContext(NonCancellable + Dispatchers.Main) {
                flushDeltaBuffer(force = true)
                if (!completed) {
                    updateAssistantAt(assistantIndex) { current ->
                        current.copy(
                            content = if (current.content.isBlank()) {
                                "回复已中断。"
                            } else {
                                current.content + "\n\n回复已中断。"
                            },
                            streaming = false,
                            renderMarkdown = false,
                            statusText = null,
                            time = "刚刚"
                        )
                    }
                    message = "AI 回复已中断，请稍后重试。"
                }
            }
            throw e
        } finally {
            if (activeRequestId == clientMessageId) activeRequestId = null
        }
    }

    suspend fun purchase(plan: AiPlan): RepoResult<AiPurchase> {
        val planId = plan.stableId()
        if (planId.isBlank()) return RepoResult.Error("套餐暂时不可购买。")
        val userId = sessionStore.loadUser()?.userId ?: return RepoResult.Error("请先登录。", code = "UNAUTHORIZED")
        if (purchasing) return RepoResult.Error("购买正在处理中，请稍后。")
        purchasing = true
        message = null
        val pending = sessionStore.loadPendingAiPurchase(userId, planId)
        val clientRequestId = pending?.clientRequestId ?: "android-ai-purchase-${UUID.randomUUID()}"
        sessionStore.savePendingAiPurchase(userId, planId, PendingAiPurchaseRef(clientRequestId, pending?.purchaseId))
        val request = AiPurchaseRequest(
            clientRequestId = clientRequestId,
            planId = planId
        )
        return try {
            when (val result = apiClient.request { apiClient.backend.createAiPurchase(request) }) {
                is RepoResult.Success -> {
                    val purchase = result.value.purchase
                    result.value.quota?.let { quota = it }
                    if (purchase.status == "processing" || purchase.status == "unknown") {
                        sessionStore.savePendingAiPurchase(userId, planId, PendingAiPurchaseRef(purchase.clientRequestId ?: clientRequestId, purchase.purchaseId))
                    } else {
                        sessionStore.clearPendingAiPurchase(userId, planId)
                    }
                    message = purchaseStatusMessage(purchase.status)
                    loadStatusSilently()
                    RepoResult.Success(purchase)
                }
                is RepoResult.Error -> {
                    if (handleUnauthorized(result)) return result
                    if (result.code in setOf("BALANCE_INSUFFICIENT", "AI_PURCHASE_FAILED", "AI_PLAN_NOT_FOUND", "PLUGIN_BRIDGE_UNAVAILABLE")) {
                        sessionStore.clearPendingAiPurchase(userId, planId)
                    }
                    val errorMessage = friendlyAiError(result.message, result.code, result.retryAfterSeconds)
                    message = errorMessage
                    result.copy(message = errorMessage)
                }
            }
        } finally {
            purchasing = false
        }
    }

    fun clearLocal() {
        messages.clear()
        plans.clear()
        assistantName = "Deuterium AI"
        currentPlan = null
        quota = null
        conversation = null
        loading = false
        activeRequestId = null
        purchasing = false
        message = null
    }

    fun clearMessage() {
        message = null
    }

    private suspend fun loadStatusSilently() {
        val result = apiClient.request { apiClient.backend.aiMe() }
        if (result is RepoResult.Success) applyMe(result.value)
    }

    private fun applyMe(data: AiMeData) {
        assistantName = data.assistantName.ifBlank { "Deuterium AI" }
        currentPlan = data.currentPlan ?: data.plan
        quota = data.quota
        conversation = data.conversation
    }

    private fun replaceMessages(items: List<AiMessage>) {
        if (sending) return
        messages.clear()
        messages.addAll(items.map { it.toUiMessage(streaming = false) })
    }

    private fun updateAssistantAt(index: Int, transform: (AiUiMessage) -> AiUiMessage) {
        if (index !in messages.indices) return
        messages[index] = transform(messages[index])
    }

    private fun handleUnauthorized(result: RepoResult.Error): Boolean {
        if (result.code != "UNAUTHORIZED" && !result.message.contains("登录状态已失效")) return false
        sessionStore.clearSession()
        message = "登录状态已失效，请重新登录。"
        onUnauthorized()
        return true
    }

    private fun AiMessage.toUiMessage(streaming: Boolean): AiUiMessage =
        AiUiMessage(
            id = messageId ?: id ?: "ai-message-${UUID.randomUUID()}",
            role = role,
            content = content,
            time = formatIsoTimeUtc8(createdAt ?: sentAt),
            streaming = streaming
        )
}

fun validateAiMessage(content: String): String? {
    return when {
        content.isBlank() -> "请输入要问 AI 的内容。"
        content.length > 1000 -> "AI 消息不能超过 1000 字。"
        else -> null
    }
}

fun AiPlan.stableId(): String = planId ?: id ?: ""

fun AiPlan.priceLabel(): String {
    val amount = price ?: priceAmount ?: return "价格待定"
    return "$amount $currency"
}

fun AiPlan.durationLabel(): String {
    return when {
        durationHours != null -> "${durationHours} 小时"
        durationDays != null -> "${durationDays} 天"
        validDays != null -> "${validDays} 天"
        else -> "有效期以后端为准"
    }
}

fun AiPlan.quotaLabel(): String {
    val planQuota = quotaPerWindow ?: quotaPerFiveHours ?: quota?.limit
    val hours = windowHours ?: quota?.windowHours ?: 5
    return if (planQuota != null && planQuota > 0) "每 $hours 小时 $planQuota 次" else "额度以后端配置为准"
}

fun friendlyAiError(error: ApiErrorBody): String =
    friendlyAiError(error.message, error.code, error.retryAfterSeconds)

fun friendlyAiError(message: String?, code: String?, retryAfterSeconds: Long? = null): String {
    val retryText = retryAfterSeconds?.takeIf { it > 0 }?.let { "，约 ${it} 秒后再试" }.orEmpty()
    return when (code) {
        "UNAUTHORIZED" -> "登录状态已失效，请重新登录。"
        "AI_QUOTA_EXCEEDED" -> message?.takeIf { it.isNotBlank() } ?: "本轮 AI 额度已用完$retryText。"
        "AI_PROVIDER_UNAVAILABLE", "AI_PROVIDER_TIMEOUT" -> message?.takeIf { it.isNotBlank() } ?: "AI 服务暂时不可用，请稍后再试。"
        "AI_PROMPT_RISK_BLOCKED" -> message?.takeIf { it.isNotBlank() } ?: "这条消息触发了安全限制，请换个问法。"
        "AI_MESSAGE_EMPTY" -> "请输入要问 AI 的内容。"
        "AI_MESSAGE_TOO_LONG" -> "消息太长，请缩短后再发送。"
        "AI_PLAN_NOT_FOUND" -> "套餐不存在或已下架，请刷新套餐列表。"
        "AI_PURCHASE_RESULT_UNKNOWN" -> "购买结果暂时未知，请稍后查看余额和套餐状态。"
        "AI_PURCHASE_FAILED" -> message?.takeIf { it.isNotBlank() } ?: "购买失败，请稍后再试。"
        else -> message?.takeIf { it.isNotBlank() } ?: "AI 功能暂时不可用，请稍后再试。"
    }
}

private fun purchaseStatusMessage(status: String): String {
    return when (status) {
        "success", "active" -> "购买成功，套餐已生效。"
        "processing" -> "购买处理中，请稍后查看结果。"
        "unknown" -> "购买结果暂时未知，请稍后查看余额和套餐状态。"
        "failed" -> "购买失败，请稍后再试。"
        else -> "购买请求已提交。"
    }
}

private const val InitialLoadFreshMillis = 30_000L
private const val DeltaFlushMinChars = 64
private const val DeltaFlushIntervalMillis = 80L
