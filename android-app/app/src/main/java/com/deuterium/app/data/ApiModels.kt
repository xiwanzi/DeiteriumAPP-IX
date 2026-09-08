package com.deuterium.app.data

import androidx.compose.runtime.Immutable

data class ApiResponse<T>(
    val requestId: String? = null,
    val data: T? = null,
    val page: Page? = null
)

data class ApiErrorResponse(
    val requestId: String? = null,
    val error: ApiErrorBody? = null
)

data class ApiErrorBody(
    val code: String? = null,
    val message: String? = null,
    val details: Map<String, Any>? = null,
    val retryAfterSeconds: Long? = null
)

data class Page(val nextCursor: String? = null)

data class RegistrationCodeRequest(val gameId: String, val qq: String, val password: String)

data class RegisterRequest(val verificationToken: String, val code: String, val password: String)

data class LoginRequest(val account: String, val password: String)

data class PasswordResetCodeRequest(val account: String)

data class PasswordResetRequest(val verificationToken: String, val code: String, val newPassword: String)

data class CreateTransferRequest(
    val clientRequestId: String,
    val recipientPlayerRef: String,
    val amount: String,
    val note: String?
)

@Immutable
data class UserProfile(
    val userId: String,
    val playerRef: String,
    val gameId: String,
    val qq: String,
    val identityStatus: String = "bound"
)

@Immutable
data class PlayerSummary(
    val playerRef: String,
    val gameId: String,
    val qq: String? = null,
    val online: Boolean = false,
    val registered: Boolean = false,
    val source: String = "search"
)

@Immutable
data class ResolvedPlayerRef(
    val playerRef: String,
    val gameId: String,
    val qq: String? = null,
    val online: Boolean = false,
    val registered: Boolean = false,
    val source: String = "search",
    val confirmedAt: String? = null,
    val expiresAt: String? = null
)

@Immutable
data class WalletBalance(
    val currency: String,
    val amount: String,
    val fresh: Boolean,
    val refreshedAt: String?
)

@Immutable
data class WalletRecord(
    val recordId: String,
    val direction: String,
    val otherPlayer: PlayerSummary,
    val amount: String,
    val currency: String,
    val status: String,
    val note: String? = null,
    val occurredAt: String
)

@Immutable
data class Transfer(
    val transferId: String,
    val clientRequestId: String,
    val recipient: PlayerSummary,
    val amount: String,
    val currency: String,
    val note: String? = null,
    val status: String,
    val createdAt: String,
    val updatedAt: String
)

@Immutable
data class ChatMessage(
    val messageId: String,
    val sender: PlayerSummary,
    val content: String,
    val kind: String = "public_chat",
    val sentAt: String
)

@Immutable
data class ServerEvent(
    val eventId: String,
    val eventType: String,
    val content: String,
    val occurredAt: String
)

@Immutable
data class OnlinePlayer(
    val playerRef: String,
    val gameId: String,
    val qq: String? = null,
    val registered: Boolean = false,
    val onlineSince: String? = null
)

@Immutable
data class PlayerDirectoryItem(
    val playerRef: String,
    val gameId: String,
    val qq: String? = null,
    val registered: Boolean = false,
    val serverOnline: Boolean = false,
    val appStatus: String = "long_offline",
    val appConnected: Boolean = false,
    val appForeground: Boolean = false,
    val appLastSeenAt: String? = null,
    val onlineSince: String? = null,
    val followed: Boolean = false,
    val self: Boolean = false
)

data class VerificationTokenData(
    val verificationToken: String,
    val expiresAt: String,
    val resendAfterSeconds: Int
)

data class AuthData(val token: String, val user: UserProfile)

data class PasswordResetData(val passwordReset: Boolean)

data class LogoutData(val loggedOut: Boolean)

data class UserProfileData(val user: UserProfile)

data class WalletBalanceData(val balance: WalletBalance)

data class RecipientSearchData(val candidates: List<ResolvedPlayerRef> = emptyList())

data class TransferData(val transfer: Transfer)

data class WalletRecordsData(val records: List<WalletRecord> = emptyList())

@Immutable
data class AppUpdateCheckData(
    val latest: Boolean,
    val message: String,
    val latestVersionCode: Int,
    val latestVersionName: String
)

data class ChatMessagesData(val messages: List<ChatMessage> = emptyList())

data class PresenceData(val onlineCount: Int, val available: Boolean, val updatedAt: String?)

data class OnlinePlayersData(val players: List<OnlinePlayer> = emptyList())

data class PlayerDirectoryData(val players: List<PlayerDirectoryItem> = emptyList())

data class PlayerFollowRequest(val playerRef: String)

data class PlayerFollowData(val followed: Boolean, val player: PlayerDirectoryItem? = null)

data class FollowedPlayersData(val players: List<PlayerDirectoryItem> = emptyList())

data class AppStatePayload(val foreground: Boolean)

@Immutable
data class AiQuota(
    val remaining: Int = 0,
    val limit: Int = 0,
    val used: Int = 0,
    val windowSeconds: Long? = null,
    val windowHours: Int? = null,
    val resetsAt: String? = null,
    val resetAt: String? = null,
    val restoresAt: String? = null,
    val restoreAt: String? = null,
    val retryAfterSeconds: Long? = null
)

@Immutable
data class AiPlan(
    val planId: String? = null,
    val id: String? = null,
    val code: String? = null,
    val name: String = "",
    val description: String? = null,
    val price: String? = null,
    val priceAmount: String? = null,
    val currency: String = "CREDIT",
    val durationHours: Int? = null,
    val durationDays: Int? = null,
    val validDays: Int? = null,
    val quotaPerWindow: Int? = null,
    val windowHours: Int? = null,
    val quotaPerFiveHours: Int? = null,
    val quota: AiQuota? = null,
    val modelTier: String? = null,
    val active: Boolean = true,
    val enabled: Boolean = true
)

@Immutable
data class AiConversation(
    val conversationId: String? = null,
    val id: String? = null,
    val status: String = "active",
    val createdAt: String? = null,
    val updatedAt: String? = null
)

@Immutable
data class AiMessage(
    val messageId: String? = null,
    val id: String? = null,
    val role: String = "assistant",
    val content: String = "",
    val createdAt: String? = null,
    val sentAt: String? = null
)

data class AiMeData(
    val assistantName: String = "Deuterium AI",
    val plan: AiPlan? = null,
    val currentPlan: AiPlan? = null,
    val quota: AiQuota? = null,
    val conversation: AiConversation? = null
)

data class AiPlansData(val plans: List<AiPlan> = emptyList())

data class AiMessagesData(val messages: List<AiMessage> = emptyList())

data class AiChatStreamRequest(val clientMessageId: String, val content: String)

data class AiConversationResetData(
    val conversation: AiConversation? = null,
    val quota: AiQuota? = null,
    val messages: List<AiMessage> = emptyList()
)

data class AiConversationResetRequest(val clientRequestId: String? = null)

data class AiPurchaseRequest(val clientRequestId: String, val planId: String)

@Immutable
data class AiPurchase(
    val purchaseId: String,
    val clientRequestId: String? = null,
    val planId: String? = null,
    val status: String = "processing",
    val amount: String? = null,
    val currency: String = "CREDIT",
    val failureCode: String? = null,
    val createdAt: String? = null,
    val updatedAt: String? = null
)

data class AiPurchaseData(val purchase: AiPurchase, val quota: AiQuota? = null)

data class AiStreamMetaData(
    val conversationId: String? = null,
    val userMessageId: String? = null,
    val assistantMessageId: String? = null,
    val quota: AiQuota? = null
)

data class AiStreamDeltaData(
    val delta: String? = null,
    val text: String? = null,
    val content: String? = null
)

data class AiStreamStatusData(
    val status: String? = null,
    val message: String? = null
)

data class AiKnowledgeSource(
    val query: String? = null,
    val title: String? = null,
    val category: String? = null,
    val headingPath: String? = null,
    val sourceUrl: String? = null,
    val excerpt: String? = null,
    val score: Int? = null
)

data class AiStreamDoneData(
    val message: AiMessage? = null,
    val assistantMessage: AiMessage? = null,
    val quota: AiQuota? = null
)

data class WalletRecordEventData(val record: WalletRecord)

data class ChatMentionEventData(val message: ChatMessage)

enum class ChatDeliveryState {
    Confirmed,
    Pending,
    Unknown
}

@Immutable
data class ChatFeedItem(
    val id: String,
    val sender: String,
    val senderPlayerRef: String? = null,
    val content: String,
    val time: String,
    val mine: Boolean = false,
    val event: Boolean = false,
    val sentAt: String? = null,
    val kind: String = "public_chat",
    val deliveryState: ChatDeliveryState = ChatDeliveryState.Confirmed
)

@Immutable
data class ChatHistoryItem(
    val messageId: String,
    val sender: String,
    val content: String,
    val kind: String,
    val sentAt: String,
    val displayTime: String,
    val mine: Boolean,
    val event: Boolean
)

sealed class RepoResult<out T> {
    data class Success<T>(val value: T) : RepoResult<T>()
    data class Error(
        val message: String,
        val code: String? = null,
        val retryAfterSeconds: Long? = null
    ) : RepoResult<Nothing>()
}
