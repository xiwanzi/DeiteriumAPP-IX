package com.deuterium.app.repository

import android.os.SystemClock
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.deuterium.app.BuildConfig
import com.deuterium.app.data.AppUpdateCheckData
import com.deuterium.app.data.AuthData
import com.deuterium.app.data.AppStatePayload
import com.deuterium.app.data.ChatFeedItem
import com.deuterium.app.data.ChatDeliveryState
import com.deuterium.app.data.ChatHistoryItem
import com.deuterium.app.data.ChatMessage
import com.deuterium.app.data.CreateTransferRequest
import com.deuterium.app.data.LoginRequest
import com.deuterium.app.data.OnlinePlayer
import com.deuterium.app.data.PasswordResetCodeRequest
import com.deuterium.app.data.PasswordResetRequest
import com.deuterium.app.data.PlayerDirectoryItem
import com.deuterium.app.data.PlayerFollowRequest
import com.deuterium.app.data.PlayerSummary
import com.deuterium.app.data.RegisterRequest
import com.deuterium.app.data.RegistrationCodeRequest
import com.deuterium.app.data.RepoResult
import com.deuterium.app.data.ResolvedPlayerRef
import com.deuterium.app.data.Transfer
import com.deuterium.app.data.UserProfile
import com.deuterium.app.data.VerificationTokenData
import com.deuterium.app.data.WalletBalance
import com.deuterium.app.data.WalletRecord
import com.deuterium.app.network.ApiClient
import com.google.gson.Gson
import kotlinx.coroutines.async
import kotlinx.coroutines.cancel
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.withTimeoutOrNull
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.time.Instant
import java.time.Duration
import java.util.UUID
import java.util.concurrent.atomic.AtomicBoolean

private const val PlayerDirectoryRefreshMinIntervalMillis = 15_000L
private const val WalletSyncFreshMillis = 5_000L
private const val ChatSyncFreshMillis = 5_000L
private const val PresenceSyncFreshMillis = 5_000L
private const val FollowSyncFreshMillis = 30_000L
private const val HistoryWriteBatchMillis = 250L
private const val HistoryWriteBatchSize = 64
private const val PendingMessageTimeoutMillis = 12_000L
private const val PendingMessageMatchWindowMillis = 30_000L
private val TerminalTransferErrorCodes = setOf(
    "AMOUNT_INVALID",
    "RECIPIENT_NOT_FOUND",
    "RECIPIENT_IDENTITY_UNCONFIRMED",
    "BALANCE_INSUFFICIENT",
    "TRANSFER_DUPLICATE",
    "TRANSFER_FAILED",
    "PLUGIN_BRIDGE_UNAVAILABLE"
)

data class PendingAiPurchaseRef(
    val clientRequestId: String,
    val purchaseId: String? = null
)

data class PendingTransferRequest(
    val clientRequestId: String,
    val recipientPlayerRef: String,
    val amount: String,
    val note: String?
) {
    fun matches(recipientPlayerRef: String, amount: String, note: String?): Boolean =
        this.recipientPlayerRef == recipientPlayerRef && this.amount == amount && this.note == note
}

private data class HistoryWrite(
    val accountId: String,
    val item: ChatFeedItem
)

interface SessionStore {
    fun loadToken(): String?
    fun loadUser(): UserProfile?
    fun saveSession(token: String, user: UserProfile)
    fun saveUser(user: UserProfile)
    fun clearSession()
    fun loadLastWalletRecordId(userId: String): String? = null
    fun saveLastWalletRecordId(userId: String, recordId: String) = Unit
    fun loadPendingTransfer(userId: String): PendingTransferRequest? = null
    fun savePendingTransfer(userId: String, pending: PendingTransferRequest) = Unit
    fun clearPendingTransfer(userId: String) = Unit
    fun loadPendingAiPurchase(userId: String, planId: String): PendingAiPurchaseRef? = null
    fun savePendingAiPurchase(userId: String, planId: String, pending: PendingAiPurchaseRef) = Unit
    fun clearPendingAiPurchase(userId: String, planId: String) = Unit
}

class AuthRepository(
    private val apiClient: ApiClient,
    private val sessionStore: SessionStore
) {
    suspend fun restoreSession(): RepoResult<UserProfile> {
        if (sessionStore.loadToken().isNullOrBlank()) {
            return RepoResult.Error("请先登录。", code = "UNAUTHORIZED")
        }
        return when (val result = apiClient.request { apiClient.backend.me() }) {
            is RepoResult.Success -> {
                sessionStore.saveUser(result.value.user)
                RepoResult.Success(result.value.user)
            }
            is RepoResult.Error -> {
                if (result.code == "UNAUTHORIZED") sessionStore.clearSession()
                result
            }
        }
    }

    suspend fun login(account: String, password: String): RepoResult<UserProfile> {
        val localError = validateLogin(account, password)
        if (localError != null) return RepoResult.Error(localError)
        return authRequest {
            apiClient.request { apiClient.backend.login(LoginRequest(account.trim(), password)) }
        }
    }

    suspend fun requestRegisterCode(gameId: String, qq: String, password: String): RepoResult<VerificationTokenData> {
        val localError = when {
            gameId.isBlank() -> "请输入游戏内 ID。"
            qq.isBlank() -> "请输入 QQ 号。"
            else -> validatePassword(password)
        }
        if (localError != null) return RepoResult.Error(localError)
        return apiClient.request {
            apiClient.backend.requestRegistrationCode(
                RegistrationCodeRequest(gameId.trim(), qq.trim(), password)
            )
        }
    }

    suspend fun register(verificationToken: String, code: String, password: String): RepoResult<UserProfile> {
        val localError = validateCode(code) ?: validatePassword(password)
        if (localError != null) return RepoResult.Error(localError)
        if (verificationToken.isBlank()) return RepoResult.Error("请先获取服务器私聊验证码。")
        return authRequest {
            apiClient.request {
                apiClient.backend.register(RegisterRequest(verificationToken, code.trim(), password))
            }
        }
    }

    suspend fun requestResetCode(account: String): RepoResult<VerificationTokenData> {
        if (account.isBlank()) return RepoResult.Error("请输入玩家 ID 或 QQ 号。")
        return apiClient.request {
            apiClient.backend.requestPasswordResetCode(PasswordResetCodeRequest(account.trim()))
        }
    }

    suspend fun resetPassword(verificationToken: String, code: String, newPassword: String): RepoResult<String> {
        val localError = validateCode(code) ?: validatePassword(newPassword)
        if (localError != null) return RepoResult.Error(localError)
        if (verificationToken.isBlank()) return RepoResult.Error("请先获取改密验证码。")
        return when (val result = apiClient.request {
            apiClient.backend.resetPassword(PasswordResetRequest(verificationToken, code.trim(), newPassword))
        }) {
            is RepoResult.Success -> {
                sessionStore.clearSession()
                RepoResult.Success("密码已重设，请使用新密码登录。")
            }
            is RepoResult.Error -> result
        }
    }

    suspend fun logout() {
        runCatching { apiClient.request { apiClient.backend.logout() } }
        sessionStore.clearSession()
    }

    private suspend fun authRequest(call: suspend () -> RepoResult<AuthData>): RepoResult<UserProfile> {
        return when (val result = call()) {
            is RepoResult.Success -> {
                sessionStore.saveSession(result.value.token, result.value.user)
                RepoResult.Success(result.value.user)
            }
            is RepoResult.Error -> result
        }
    }
}

class AppUpdateRepository(private val apiClient: ApiClient) {
    suspend fun checkUpdate(): RepoResult<AppUpdateCheckData> {
        return apiClient.request {
            apiClient.backend.updateCheck(BuildConfig.VERSION_CODE, BuildConfig.VERSION_NAME)
        }
    }
}

class WalletRepository(
    private val apiClient: ApiClient,
    private val sessionStore: SessionStore
) {
    private val syncMutex = Mutex()
    private val transferMutex = Mutex()
    private var lastSyncAt = 0L
    var balance by mutableStateOf<WalletBalance?>(null)
        private set
    var walletMessage by mutableStateOf<String?>(null)
        private set
    val records = mutableStateListOf<WalletRecord>()

    suspend fun loadWallet(force: Boolean = false): RepoResult<Unit> = syncMutex.withLock {
        val now = SystemClock.elapsedRealtime()
        if (!force && balance != null && now - lastSyncAt < WalletSyncFreshMillis) {
            return@withLock RepoResult.Success(Unit)
        }
        val result = loadWalletUnlocked()
        lastSyncAt = SystemClock.elapsedRealtime()
        result
    }

    private suspend fun loadWalletUnlocked(): RepoResult<Unit> {
        val balanceResult = apiClient.request { apiClient.backend.walletBalance() }
        if (balanceResult is RepoResult.Success) {
            balance = balanceResult.value.balance
        } else if (balanceResult is RepoResult.Error) {
            walletMessage = balanceResult.message
        }

        val recordsResult = apiClient.request { apiClient.backend.walletRecords(limit = 20) }
        if (recordsResult is RepoResult.Success) {
            replaceRecords(recordsResult.value.records)
            updateSyncMarker(records)
        } else if (recordsResult is RepoResult.Error && walletMessage == null) {
            walletMessage = recordsResult.message
        }

        return if (balanceResult is RepoResult.Success || recordsResult is RepoResult.Success) {
            RepoResult.Success(Unit)
        } else {
            RepoResult.Error(walletMessage ?: "钱包数据加载失败，请稍后再试。")
        }
    }

    suspend fun refreshBalance(): RepoResult<WalletBalance> {
        return when (val result = apiClient.request { apiClient.backend.refreshWalletBalance() }) {
            is RepoResult.Success -> {
                balance = result.value.balance
                walletMessage = "余额已刷新：${result.value.balance.amount}"
                RepoResult.Success(result.value.balance)
            }
            is RepoResult.Error -> {
                walletMessage = result.message
                result
            }
        }
    }

    suspend fun syncNewRecords(force: Boolean = false): RepoResult<Int> = syncMutex.withLock {
        val now = SystemClock.elapsedRealtime()
        if (!force && now - lastSyncAt < WalletSyncFreshMillis) {
            return@withLock RepoResult.Success(0)
        }
        val userId = sessionStore.loadUser()?.userId
            ?: return@withLock RepoResult.Error("请先登录。", code = "UNAUTHORIZED")
        val afterRecordId = sessionStore.loadLastWalletRecordId(userId)
        val result = if (afterRecordId.isNullOrBlank()) {
            when (val loadResult = loadWalletUnlocked()) {
                is RepoResult.Success -> RepoResult.Success(records.size)
                is RepoResult.Error -> loadResult
            }
        } else when (val recordsResult = apiClient.request { apiClient.backend.walletRecords(limit = 100, afterRecordId = afterRecordId) }) {
            is RepoResult.Success -> {
                val inserted = mergeRecords(recordsResult.value.records)
                if (inserted > 0) updateSyncMarker(records)
                RepoResult.Success(inserted)
            }
            is RepoResult.Error -> recordsResult
        }
        lastSyncAt = SystemClock.elapsedRealtime()
        result
    }

    fun mergeRecord(record: WalletRecord) {
        val inserted = mergeRecords(listOf(record))
        if (inserted > 0) updateSyncMarker(records)
    }

    suspend fun findRecipient(query: String): RepoResult<ResolvedPlayerRef> {
        val normalized = query.trim()
        if (normalized.isBlank()) return RepoResult.Error("请输入游戏内 ID 或 QQ 号。")
        return when (val result = apiClient.request {
            apiClient.backend.searchRecipients(normalized, "auto")
        }) {
            is RepoResult.Success -> {
                val candidate = result.value.candidates.firstOrNull()
                    ?: return RepoResult.Error("未找到可确认的收款玩家。", code = "RECIPIENT_NOT_FOUND")
                RepoResult.Success(candidate)
            }
            is RepoResult.Error -> result
        }
    }

    suspend fun transfer(recipient: ResolvedPlayerRef?, amountText: String, note: String): RepoResult<Transfer> =
        transferMutex.withLock { transferUnlocked(recipient, amountText, note) }

    private suspend fun transferUnlocked(recipient: ResolvedPlayerRef?, amountText: String, note: String): RepoResult<Transfer> {
        if (recipient == null) return RepoResult.Error("请先确认收款玩家。")
        val amountError = validateAmount(amountText)
        if (amountError != null) return RepoResult.Error(amountError)
        val userId = sessionStore.loadUser()?.userId
            ?: return RepoResult.Error("请先登录。", code = "UNAUTHORIZED")
        val normalizedAmount = amountText.trim().toBigDecimal().setScale(2).toPlainString()
        val normalizedNote = note.trim().ifBlank { null }
        val pending = sessionStore.loadPendingTransfer(userId)
            ?.takeIf { it.matches(recipient.playerRef, normalizedAmount, normalizedNote) }
            ?: PendingTransferRequest(
                clientRequestId = "android-${UUID.randomUUID()}",
                recipientPlayerRef = recipient.playerRef,
                amount = normalizedAmount,
                note = normalizedNote
            ).also { sessionStore.savePendingTransfer(userId, it) }
        val request = CreateTransferRequest(
            clientRequestId = pending.clientRequestId,
            recipientPlayerRef = recipient.playerRef,
            amount = normalizedAmount,
            note = normalizedNote
        )
        return when (val result = apiClient.request { apiClient.backend.createTransfer(request) }) {
            is RepoResult.Success -> {
                val transfer = result.value.transfer
                if (transfer.status == "success" || transfer.status == "failed") {
                    sessionStore.clearPendingTransfer(userId)
                }
                loadWallet(force = true)
                RepoResult.Success(transfer)
            }
            is RepoResult.Error -> {
                if (result.code in TerminalTransferErrorCodes) {
                    sessionStore.clearPendingTransfer(userId)
                }
                result
            }
        }
    }

    fun clearMessage() {
        walletMessage = null
    }

    private fun mergeRecords(incoming: List<WalletRecord>): Int {
        if (incoming.isEmpty()) return 0
        val existingIds = records.map { it.recordId }.toHashSet()
        val inserted = incoming.filter { existingIds.add(it.recordId) }
        if (inserted.isEmpty()) return 0
        replaceRecords(records + inserted)
        return inserted.size
    }

    private fun replaceRecords(items: List<WalletRecord>) {
        records.clear()
        records.addAll(
            items
                .distinctBy { it.recordId }
                .sortedWith(compareByDescending<WalletRecord> { it.occurredAt }.thenByDescending { it.recordId })
                .take(100)
        )
    }

    private fun updateSyncMarker(items: List<WalletRecord>) {
        val userId = sessionStore.loadUser()?.userId ?: return
        val latest = items.maxWithOrNull(compareBy<WalletRecord> { it.occurredAt }.thenBy { it.recordId }) ?: return
        sessionStore.saveLastWalletRecordId(userId, latest.recordId)
    }
}

class ChatRepository(
    private val apiClient: ApiClient,
    private val sessionStore: SessionStore,
    private val historyStore: ChatHistoryStore,
    private val onUnauthorized: () -> Unit = {},
    private val walletNotificationsEnabled: () -> Boolean = { false },
    private val onNotificationPermissionNeeded: () -> Unit = {},
    private val onFollowedChatNotification: (ChatMessage) -> Unit = {},
    private val onWalletNotification: (WalletRecord) -> Unit = {},
    private val onMentionNotification: (ChatMessage) -> Unit = {},
    private val onWalletRecord: (WalletRecord) -> Unit = {},
    private val onSocketConnected: () -> Unit = {}
) {
    private val gson = Gson()
    private val socketEventParser = ChatSocketEventParser(gson)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private val messageSyncMutex = Mutex()
    private val presenceSyncMutex = Mutex()
    private val followSyncMutex = Mutex()
    private val historyWriteChannel = Channel<HistoryWrite>(Channel.UNLIMITED)
    private val historyWriteScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val historyWriterJob = historyWriteScope.launch { writeHistoryBatches() }
    private var webSocket: WebSocket? = null
    private var reconnecting = AtomicBoolean(false)
    private var includeEvents = false
    private var shouldReconnect = false
    private val knownMessageIds = LinkedHashSet<String>()
    private val pendingClientMessages = LinkedHashSet<String>()
    private val pendingMessageTimeoutJobs = mutableMapOf<String, Job>()
    private var playerDirectoryRefreshPending = false
    private var lastPlayerDirectoryRefreshAt = 0L
    private var lastMessageSyncAt = 0L
    private var lastPresenceSyncAt = 0L
    private var lastFollowSyncAt = 0L
    private var closed = false

    val messages = mutableStateListOf<ChatFeedItem>()
    val onlinePlayers = mutableStateListOf<OnlinePlayer>()
    val playerDirectory = mutableStateListOf<PlayerDirectoryItem>()
    private val followedPlayerRefs = mutableStateListOf<String>()
    var onlineCount by mutableStateOf<Int?>(null)
        private set
    var connectionMessage by mutableStateOf<String?>(null)
        private set
    var appForeground by mutableStateOf(false)
        private set

    suspend fun loadInitial(): RepoResult<Unit> = coroutineScope {
        val chat = async { syncRecentMessages(surfaceErrors = true) }
        val presence = async { refreshPresenceAndDirectory() }
        val follows = async { refreshFollows() }
        val chatResult = chat.await()
        presence.await()
        follows.await()
        if (chatResult is RepoResult.Success) {
            RepoResult.Success(Unit)
        } else {
            RepoResult.Error(connectionMessage ?: "聊天记录加载失败。")
        }
    }

    suspend fun refreshPlayerDirectory(): RepoResult<List<PlayerDirectoryItem>> {
        lastPlayerDirectoryRefreshAt = SystemClock.elapsedRealtime()
        return when (val result = apiClient.request { apiClient.backend.playerDirectory() }) {
            is RepoResult.Success -> {
                replacePlayerDirectory(result.value.players)
                refreshFollowedRefsFromDirectory()
                RepoResult.Success(result.value.players)
            }
            is RepoResult.Error -> {
                if (handleUnauthorized(result)) return result
                val fallback = onlinePlayers.map { player ->
                    PlayerDirectoryItem(
                        playerRef = player.playerRef,
                        gameId = player.gameId,
                        qq = player.qq,
                        registered = player.registered,
                        serverOnline = true,
                        appStatus = "long_offline",
                        appLastSeenAt = null,
                        onlineSince = player.onlineSince,
                        followed = followedPlayerRefs.contains(player.playerRef),
                        self = player.playerRef == sessionStore.loadUser()?.playerRef
                    )
                }
                if (fallback.isNotEmpty()) {
                    replacePlayerDirectory(fallback)
                    RepoResult.Success(fallback)
                } else {
                    result
                }
            }
        }
    }

    suspend fun refreshFollows(force: Boolean = false): RepoResult<List<PlayerDirectoryItem>> = followSyncMutex.withLock {
        if (!force && isFresh(lastFollowSyncAt, FollowSyncFreshMillis)) {
            return@withLock RepoResult.Success(playerDirectory.filter { it.followed })
        }
        val result = when (val apiResult = apiClient.request { apiClient.backend.follows() }) {
            is RepoResult.Success -> {
                followedPlayerRefs.clear()
                followedPlayerRefs.addAll(apiResult.value.players.map { it.playerRef })
                RepoResult.Success(apiResult.value.players)
            }
            is RepoResult.Error -> {
                if (handleUnauthorized(apiResult)) return@withLock apiResult
                apiResult
            }
        }
        lastFollowSyncAt = SystemClock.elapsedRealtime()
        result
    }

    suspend fun refreshPresenceAndDirectory(force: Boolean = false): RepoResult<Unit> = presenceSyncMutex.withLock {
        if (!force && isFresh(lastPresenceSyncAt, PresenceSyncFreshMillis)) {
            return@withLock RepoResult.Success(Unit)
        }
        val presenceResult = apiClient.request { apiClient.backend.presence() }
        if (presenceResult is RepoResult.Success) {
            onlineCount = presenceResult.value.onlineCount
        } else if (presenceResult is RepoResult.Error && handleUnauthorized(presenceResult)) {
            return@withLock presenceResult
        }
        val playersResult = apiClient.request { apiClient.backend.onlinePlayers() }
        if (playersResult is RepoResult.Success) {
            replaceOnlinePlayers(playersResult.value.players)
        } else if (playersResult is RepoResult.Error && handleUnauthorized(playersResult)) {
            return@withLock playersResult
        }
        refreshPlayerDirectory()
        lastPresenceSyncAt = SystemClock.elapsedRealtime()
        RepoResult.Success(Unit)
    }

    suspend fun syncRecentMessages(surfaceErrors: Boolean = false, force: Boolean = false): RepoResult<Int> =
        messageSyncMutex.withLock {
        if (!force && isFresh(lastMessageSyncAt, ChatSyncFreshMillis)) {
            return@withLock RepoResult.Success(0)
        }
        val result = when (val apiResult = apiClient.request { apiClient.backend.chatMessages(limit = 100) }) {
            is RepoResult.Success -> {
                val incoming = apiResult.value.messages.map(::toFeedItem)
                reconcilePendingMessages(incoming)
                val inserted = mergeIncomingMessages(incoming)
                saveHistoryNow(inserted)
                RepoResult.Success(inserted.size)
            }
            is RepoResult.Error -> {
                if (handleUnauthorized(apiResult)) return@withLock apiResult
                if (surfaceErrors) connectionMessage = apiResult.message
                apiResult
            }
        }
        lastMessageSyncAt = SystemClock.elapsedRealtime()
        result
    }

    fun connect(showServerEvents: Boolean) {
        if (closed) return
        if (webSocket != null && includeEvents == showServerEvents) return
        includeEvents = showServerEvents
        shouldReconnect = true
        webSocket?.close(1000, "Reconnect")
        webSocket = apiClient.openChatWebSocket(showServerEvents, listener())
    }

    fun disconnect() {
        shouldReconnect = false
        webSocket?.close(1000, "Screen disposed")
        webSocket = null
    }

    fun close() {
        if (closed) return
        closed = true
        shouldReconnect = false
        reconnecting.set(false)
        webSocket?.cancel()
        webSocket = null
        historyWriteChannel.close()
        historyWriteScope.launch {
            historyWriterJob.join()
            historyStore.close()
            historyWriteScope.cancel()
        }
        scope.cancel()
    }

    fun reportAppForeground(foreground: Boolean) {
        appForeground = foreground
        sendAppState()
    }

    suspend fun toggleFollow(player: PlayerDirectoryItem): RepoResult<Boolean> {
        if (player.self) return RepoResult.Error("不能关心自己。")
        val result = if (player.followed) {
            apiClient.request { apiClient.backend.unfollowPlayer(player.playerRef) }
        } else {
            apiClient.request { apiClient.backend.followPlayer(PlayerFollowRequest(player.playerRef)) }
        }
        return when (result) {
            is RepoResult.Success -> {
                val followed = result.value.followed
                updateFollowState(player.playerRef, followed)
                if (followed) onNotificationPermissionNeeded()
                refreshPlayerDirectory()
                RepoResult.Success(followed)
            }
            is RepoResult.Error -> {
                if (handleUnauthorized(result)) return result
                result
            }
        }
    }

    fun mentionCandidates(query: String?): List<PlayerDirectoryItem> {
        return playerDirectory
            .asSequence()
            .filter { !it.self }
            .filter { it.followed || it.serverOnline || it.appConnected }
            .filter { query.isNullOrBlank() || it.gameId.contains(query, ignoreCase = true) }
            .sortedWith(
                compareByDescending<PlayerDirectoryItem> { it.followed }
                    .thenByDescending { it.serverOnline }
                    .thenByDescending { it.appForeground }
                    .thenByDescending { it.appConnected }
                    .thenBy { it.gameId.lowercase() }
            )
            .take(30)
            .toList()
    }

    fun send(content: String, mentionedPlayerRefs: List<String> = emptyList()): RepoResult<Unit> {
        val messageError = validateChatMessage(content)
        if (messageError != null) return RepoResult.Error(messageError)
        val socket = webSocket ?: return RepoResult.Error("聊天连接暂不可用，请稍后再试。")
        val requestId = "req_ws_${UUID.randomUUID()}"
        val clientMessageId = "android-msg-${UUID.randomUUID()}"
        val trimmed = content.trim()
        val payload = linkedMapOf<String, Any>(
            "clientMessageId" to clientMessageId,
            "content" to trimmed
        )
        val mentions = mentionedPlayerRefs.filter { it.isNotBlank() }.distinct().take(20)
        if (mentions.isNotEmpty()) payload["mentionedPlayerRefs"] = mentions
        val body = mapOf(
            "type" to "chat.send",
            "requestId" to requestId,
            "sentAt" to Instant.now().toString(),
            "payload" to payload
        )
        return if (socket.send(gson.toJson(body))) {
            connectionMessage = null
            appendPendingMessage(clientMessageId, trimmed)
            RepoResult.Success(Unit)
        } else {
            RepoResult.Error("消息发送失败，聊天连接暂不可用。")
        }
    }

    private fun sendAppState() {
        val socket = webSocket ?: return
        val body = mapOf(
            "type" to "app.state",
            "sentAt" to Instant.now().toString(),
            "payload" to AppStatePayload(appForeground)
        )
        socket.send(gson.toJson(body))
    }

    private fun listener(): WebSocketListener {
        return object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                scope.launch {
                    reconnecting.set(false)
                    connectionMessage = null
                    sendAppState()
                    syncRecentMessages()
                    refreshPresenceAndDirectory()
                    refreshFollows()
                    onSocketConnected()
                }
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                scope.launch {
                    val event = withContext(Dispatchers.Default) { socketEventParser.parse(text) } ?: return@launch
                    handleSocketEvent(event)
                }
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                scope.launch {
                    if (this@ChatRepository.webSocket !== webSocket) return@launch
                    this@ChatRepository.webSocket = null
                    if (!shouldReconnect) return@launch
                    scheduleReconnect()
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                scope.launch {
                    if (this@ChatRepository.webSocket !== webSocket) return@launch
                    this@ChatRepository.webSocket = null
                    if (!shouldReconnect) return@launch
                    if (response?.code == 401) {
                        shouldReconnect = false
                        sessionStore.clearSession()
                        connectionMessage = "登录状态已失效，请重新登录。"
                        onUnauthorized()
                        return@launch
                    }
                    scheduleReconnect()
                }
            }
        }
    }

    private suspend fun scheduleReconnect() {
        if (!shouldReconnect) return
        if (!reconnecting.compareAndSet(false, true)) return
        delay(1800)
        reconnecting.set(false)
        if (shouldReconnect && !sessionStore.loadToken().isNullOrBlank()) connect(includeEvents)
    }

    private fun handleSocketEvent(event: ChatSocketEvent) {
        when (event) {
            is ChatSocketEvent.Message -> {
                val message = event.message
                val feedItem = toFeedItem(message)
                appendLiveFeedItem(feedItem)
                if (shouldNotifyFollowedChat(message.sender.playerRef, message.kind, feedItem.mine, followedPlayerRefs.toSet(), appForeground)) {
                    onFollowedChatNotification(message)
                }
            }
            is ChatSocketEvent.Mention -> {
                val payload = event.data
                val currentPlayerRef = sessionStore.loadUser()?.playerRef
                if (payload.message.sender.playerRef != currentPlayerRef) {
                    onNotificationPermissionNeeded()
                    onMentionNotification(payload.message)
                }
            }
            is ChatSocketEvent.Server -> {
                val serverEvent = event.event
                val feedItem = ChatFeedItem(
                    id = serverEvent.eventId,
                    sender = "服务器",
                    content = stripMinecraftFormattingCodes(serverEvent.content),
                    time = formatTime(serverEvent.occurredAt),
                    event = true,
                    sentAt = serverEvent.occurredAt,
                    kind = serverEvent.eventType
                )
                appendLiveFeedItem(feedItem)
            }
            is ChatSocketEvent.Presence -> {
                onlineCount = event.onlineCount ?: onlineCount
                event.players?.let { updatedPlayers ->
                    replaceOnlinePlayers(updatedPlayers)
                    refreshPlayerDirectorySoon()
                }
            }
            is ChatSocketEvent.SendResult -> {
                if (event.status == "accepted") {
                    confirmPendingMessage(event.clientMessageId, event.messageId.orEmpty())
                } else {
                    removePendingMessage(event.clientMessageId)
                    connectionMessage = event.errorMessage ?: "消息发送失败，请稍后再试。"
                }
            }
            is ChatSocketEvent.ServiceError -> connectionMessage = event.message
            is ChatSocketEvent.WalletRecord -> {
                val payload = event.data
                onWalletRecord(payload.record)
                if (shouldNotifyWalletRecord(walletNotificationsEnabled(), appForeground)) {
                    onWalletNotification(payload.record)
                }
            }
        }
    }

    suspend fun searchHistory(query: String): List<ChatHistoryItem> {
        return withContext(Dispatchers.IO) {
            historyStore.search(currentAccountId(), query)
        }
    }

    suspend fun deleteHistory() {
        withContext(Dispatchers.IO) {
            historyStore.deleteAll(currentAccountId())
        }
    }

    private suspend fun saveHistoryNow(items: List<ChatFeedItem>) {
        val accountId = currentAccountId()
        withContext(Dispatchers.IO) {
            historyStore.saveAll(accountId, items)
        }
    }

    private fun saveHistory(item: ChatFeedItem) {
        val accountId = currentAccountId() ?: return
        historyWriteChannel.trySend(HistoryWrite(accountId, item))
    }

    private suspend fun writeHistoryBatches() {
        while (true) {
            val first = historyWriteChannel.receiveCatching().getOrNull() ?: break
            val batches = linkedMapOf<String, LinkedHashMap<String, ChatFeedItem>>()
            fun add(write: HistoryWrite) {
                batches.getOrPut(write.accountId, ::LinkedHashMap)[write.item.id] = write.item
            }
            add(first)
            var total = 1
            val deadline = SystemClock.elapsedRealtime() + HistoryWriteBatchMillis
            while (total < HistoryWriteBatchSize) {
                val remaining = deadline - SystemClock.elapsedRealtime()
                if (remaining <= 0L) break
                val next = withTimeoutOrNull(remaining) {
                    historyWriteChannel.receiveCatching().getOrNull()
                } ?: break
                add(next)
                total += 1
            }
            batches.forEach { (accountId, itemsById) ->
                historyStore.saveAll(accountId, itemsById.values.toList())
            }
        }
    }

    private fun currentAccountId(): String? = sessionStore.loadUser()?.userId

    private fun isFresh(lastSyncAt: Long, freshnessMillis: Long): Boolean =
        lastSyncAt > 0L && SystemClock.elapsedRealtime() - lastSyncAt < freshnessMillis

    private fun replaceMessages(items: List<ChatFeedItem>) {
        if (messages.size == items.size && messages.zip(items).all { (current, incoming) -> current == incoming }) {
            val itemIds = items.map { it.id }
            if (knownMessageIds.size != items.size || !knownMessageIds.containsAll(itemIds)) {
                knownMessageIds.clear()
                knownMessageIds.addAll(itemIds)
            }
            return
        }
        messages.clear()
        messages.addAll(items)
        knownMessageIds.clear()
        knownMessageIds.addAll(items.map { it.id })
    }

    private fun replaceOnlinePlayers(items: List<OnlinePlayer>) {
        if (onlinePlayers.size == items.size && onlinePlayers.zip(items).all { (current, incoming) -> current == incoming }) return
        onlinePlayers.clear()
        onlinePlayers.addAll(items)
    }

    private fun replacePlayerDirectory(items: List<PlayerDirectoryItem>) {
        if (playerDirectory.size == items.size && playerDirectory.zip(items).all { (current, incoming) -> current == incoming }) return
        playerDirectory.clear()
        playerDirectory.addAll(items)
    }

    private fun refreshPlayerDirectorySoon() {
        if (playerDirectoryRefreshPending) return
        playerDirectoryRefreshPending = true
        scope.launch {
            try {
                val now = SystemClock.elapsedRealtime()
                val waitMillis = (PlayerDirectoryRefreshMinIntervalMillis - (now - lastPlayerDirectoryRefreshAt)).coerceAtLeast(0L)
                if (waitMillis > 0) delay(waitMillis)
                refreshPlayerDirectory()
                lastPlayerDirectoryRefreshAt = SystemClock.elapsedRealtime()
            } finally {
                playerDirectoryRefreshPending = false
            }
        }
    }

    private fun mergeIncomingMessages(items: List<ChatFeedItem>): List<ChatFeedItem> {
        val result = mergeChatFeedItems(messages.toList(), items)
        if (result.inserted.isNotEmpty()) {
            replaceMessages(result.messages)
        }
        return result.inserted
    }

    private fun appendLiveFeedItem(item: ChatFeedItem) {
        if (knownMessageIds.contains(item.id)) return
        if (appendLiveFeedItemFast(item)) {
            saveHistory(item)
            return
        }
        val inserted = mergeIncomingMessages(listOf(item))
        inserted.forEach(::saveHistory)
    }

    private fun appendPendingMessage(clientMessageId: String, content: String) {
        val user = sessionStore.loadUser() ?: return
        if (knownMessageIds.contains(clientMessageId)) return
        val now = Instant.now().toString()
        val item = ChatFeedItem(
            id = clientMessageId,
            sender = user.gameId,
            senderPlayerRef = user.playerRef,
            content = content,
            time = formatTime(now),
            mine = true,
            sentAt = now,
            kind = "public_chat",
            deliveryState = ChatDeliveryState.Pending
        )
        pendingClientMessages.add(clientMessageId)
        pendingMessageTimeoutJobs.remove(clientMessageId)?.cancel()
        pendingMessageTimeoutJobs[clientMessageId] = scope.launch {
            delay(PendingMessageTimeoutMillis)
            if (!pendingClientMessages.contains(clientMessageId)) return@launch
            val pendingIndex = messages.indexOfFirst { it.id == clientMessageId }
            if (pendingIndex >= 0) {
                messages[pendingIndex] = messages[pendingIndex].copy(deliveryState = ChatDeliveryState.Unknown)
            }
            connectionMessage = "消息发送结果暂未确认，已自动同步最近聊天。"
            syncRecentMessages(force = true)
        }
        if (!appendLiveFeedItemFast(item)) {
            mergeIncomingMessages(listOf(item))
        }
    }

    private fun confirmPendingMessage(clientMessageId: String, messageId: String) {
        if (clientMessageId.isBlank()) return
        pendingClientMessages.remove(clientMessageId)
        pendingMessageTimeoutJobs.remove(clientMessageId)?.cancel()
        val pendingIndex = messages.indexOfFirst { it.id == clientMessageId }
        if (messageId.isBlank() || messageId == clientMessageId) {
            if (pendingIndex >= 0) {
                messages[pendingIndex] = messages[pendingIndex].copy(deliveryState = ChatDeliveryState.Confirmed)
                saveHistory(messages[pendingIndex])
            }
            return
        }
        val officialIndex = messages.indexOfFirst { it.id == messageId }
        when {
            pendingIndex >= 0 && officialIndex >= 0 -> {
                messages.removeAt(pendingIndex)
                knownMessageIds.remove(clientMessageId)
            }
            pendingIndex >= 0 -> {
                messages[pendingIndex] = messages[pendingIndex].copy(
                    id = messageId,
                    deliveryState = ChatDeliveryState.Confirmed
                )
                knownMessageIds.remove(clientMessageId)
                knownMessageIds.add(messageId)
                saveHistory(messages[pendingIndex])
            }
        }
    }

    private fun removePendingMessage(clientMessageId: String) {
        if (clientMessageId.isBlank()) return
        pendingClientMessages.remove(clientMessageId)
        pendingMessageTimeoutJobs.remove(clientMessageId)?.cancel()
        val index = messages.indexOfFirst { it.id == clientMessageId }
        if (index >= 0) {
            messages.removeAt(index)
            knownMessageIds.remove(clientMessageId)
        }
    }

    private fun reconcilePendingMessages(incoming: List<ChatFeedItem>) {
        if (pendingClientMessages.isEmpty() || incoming.isEmpty()) return
        val availableOfficial = incoming.filter { it.mine }.toMutableList()
        pendingClientMessages.toList().forEach { clientMessageId ->
            val pendingIndex = messages.indexOfFirst { it.id == clientMessageId }
            if (pendingIndex < 0) return@forEach
            val pending = messages[pendingIndex]
            val match = availableOfficial
                .filter { it.content == pending.content }
                .mapNotNull { official ->
                    val distance = timestampDistanceMillis(pending.sentAt, official.sentAt) ?: return@mapNotNull null
                    official to distance
                }
                .filter { (_, distance) -> distance <= PendingMessageMatchWindowMillis }
                .minByOrNull { (_, distance) -> distance }
                ?.first
                ?: return@forEach
            availableOfficial.remove(match)
            messages.removeAt(pendingIndex)
            knownMessageIds.remove(clientMessageId)
            pendingClientMessages.remove(clientMessageId)
            pendingMessageTimeoutJobs.remove(clientMessageId)?.cancel()
        }
    }

    private fun timestampDistanceMillis(first: String?, second: String?): Long? {
        val firstValue = first ?: return null
        val secondValue = second ?: return null
        return runCatching {
            kotlin.math.abs(Duration.between(Instant.parse(firstValue), Instant.parse(secondValue)).toMillis())
        }.getOrNull()
    }

    private fun appendLiveFeedItemFast(item: ChatFeedItem): Boolean {
        val last = messages.lastOrNull()
        if (last != null && !isChatFeedItemAtOrAfter(last, item)) return false

        if (messages.size >= DefaultChatFeedMaxMessages) {
            val removeCount = messages.size - DefaultChatFeedMaxMessages + 1
            repeat(removeCount) {
                val removed = messages.removeAt(0)
                knownMessageIds.remove(removed.id)
            }
        }
        messages.add(item)
        knownMessageIds.add(item.id)
        return true
    }

    private fun handleUnauthorized(result: RepoResult.Error): Boolean {
        if (result.code != "UNAUTHORIZED" && !result.message.contains("登录状态已失效")) return false
        shouldReconnect = false
        sessionStore.clearSession()
        connectionMessage = "登录状态已失效，请重新登录。"
        onUnauthorized()
        return true
    }

    private fun toFeedItem(message: ChatMessage): ChatFeedItem {
        val currentPlayerRef = sessionStore.loadUser()?.playerRef
        return ChatFeedItem(
            id = message.messageId,
            sender = message.sender.gameId,
            senderPlayerRef = message.sender.playerRef,
            content = message.content,
            time = formatTime(message.sentAt),
            mine = message.sender.playerRef == currentPlayerRef,
            sentAt = message.sentAt,
            kind = message.kind
        )
    }

    private fun stripMinecraftFormattingCodes(value: String): String {
        return value
            .replace(Regex("(?i)[&§][0-9A-FK-OR]"), "")
            .trim()
    }

    private fun formatTime(value: String?): String {
        return formatIsoTimeUtc8(value)
    }

    private fun updateFollowState(playerRef: String, followed: Boolean) {
        if (followed && !followedPlayerRefs.contains(playerRef)) {
            followedPlayerRefs.add(playerRef)
        }
        if (!followed) {
            followedPlayerRefs.remove(playerRef)
        }
        val index = playerDirectory.indexOfFirst { it.playerRef == playerRef }
        if (index >= 0) {
            playerDirectory[index] = playerDirectory[index].copy(followed = followed)
        }
    }

    private fun refreshFollowedRefsFromDirectory() {
        followedPlayerRefs.clear()
        followedPlayerRefs.addAll(playerDirectory.filter { it.followed }.map { it.playerRef })
    }
}

fun validateLogin(account: String, password: String): String? {
    return when {
        account.isBlank() -> "请输入玩家 ID 或 QQ 号。"
        else -> validatePassword(password)
    }
}

fun validatePassword(password: String): String? {
    return when {
        password.length < 8 -> "密码至少 8 位。"
        password.length > 64 -> "密码最多 64 位。"
        else -> null
    }
}

fun validateCode(code: String): String? {
    return when {
        !code.matches(Regex("^\\d{6}$")) -> "验证码必须是 6 位数字。"
        else -> null
    }
}

fun validateAmount(amountText: String): String? {
    val validFormat = amountText.matches(Regex("^\\d+(\\.\\d{1,2})?$"))
    val amount = amountText.toBigDecimalOrNull()
    return when {
        !validFormat || amount == null -> "金额必须是数字，且最多两位小数。"
        amount <= java.math.BigDecimal.ZERO -> "金额必须大于 0。"
        else -> null
    }
}

fun validateChatMessage(content: String): String? {
    val trimmed = content.trim()
    return when {
        trimmed.isEmpty() -> "消息不能为空。"
        trimmed.length > 256 -> "消息不能超过 256 字符。"
        else -> null
    }
}

fun playerSummary(player: ResolvedPlayerRef): PlayerSummary =
    PlayerSummary(
        playerRef = player.playerRef,
        gameId = player.gameId,
        qq = player.qq,
        online = player.online,
        registered = player.registered,
        source = player.source
    )
