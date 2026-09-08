@file:OptIn(ExperimentalMaterial3Api::class, ExperimentalFoundationApi::class, ExperimentalLayoutApi::class)

package com.deuterium.app

import android.content.Context
import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.LifecycleOwner
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateContentSize
import androidx.compose.animation.expandVertically
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkVertically
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.AlternateEmail
import androidx.compose.material.icons.filled.Chat
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.ExitToApp
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Login
import androidx.compose.material.icons.filled.People
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.PersonAdd
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Reply
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Send
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Divider
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.deuterium.app.data.ChatFeedItem
import com.deuterium.app.data.ChatDeliveryState
import com.deuterium.app.data.ChatHistoryItem
import com.deuterium.app.data.OnlinePlayer
import com.deuterium.app.data.PlayerSummary
import com.deuterium.app.data.PlayerDirectoryItem
import com.deuterium.app.data.RepoResult
import com.deuterium.app.data.ResolvedPlayerRef
import com.deuterium.app.data.UserProfile
import com.deuterium.app.data.WalletRecord
import com.deuterium.app.network.ApiClient
import com.deuterium.app.data.AiPlan
import com.deuterium.app.repository.AppUpdateRepository
import com.deuterium.app.repository.AuthRepository
import com.deuterium.app.repository.AiRepository
import com.deuterium.app.repository.AiUiMessage
import com.deuterium.app.repository.ChatHistoryStore
import com.deuterium.app.repository.ChatRepository
import com.deuterium.app.repository.PendingAiPurchaseRef
import com.deuterium.app.repository.PendingTransferRequest
import com.deuterium.app.repository.SessionStore
import com.deuterium.app.repository.WalletRepository
import com.deuterium.app.repository.durationLabel
import com.deuterium.app.repository.formatIsoDateTimeUtc8
import com.deuterium.app.repository.playerSummary
import com.deuterium.app.repository.priceLabel
import com.deuterium.app.repository.quotaLabel
import com.deuterium.app.repository.validateAmount
import com.deuterium.app.repository.validateAiMessage
import com.deuterium.app.repository.validateChatMessage
import com.deuterium.app.ui.theme.DeuteriumColorPreset
import com.deuterium.app.ui.theme.DeuteriumTheme
import com.deuterium.app.ui.theme.DeuteriumThemeMode
import kotlinx.coroutines.delay
import kotlinx.coroutines.Job
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val preferences = AppPreferences(this)
        val chatHistoryStore = ChatHistoryStore(this)
        setContent {
            var themeMode by rememberSaveable { mutableStateOf(preferences.loadThemeMode()) }
            var colorPreset by rememberSaveable { mutableStateOf(preferences.loadColorPreset()) }
            var dynamicColor by rememberSaveable { mutableStateOf(preferences.loadDynamicColor()) }
            DeuteriumTheme(
                themeMode = themeMode,
                colorPreset = colorPreset,
                dynamicColor = dynamicColor
            ) {
                DeuteriumApp(
                    preferences = preferences,
                    chatHistoryStore = chatHistoryStore,
                    appearance = AppAppearance(themeMode, colorPreset, dynamicColor),
                    onThemeModeChange = {
                        themeMode = it
                        preferences.saveThemeMode(it)
                    },
                    onColorPresetChange = {
                        colorPreset = it
                        preferences.saveColorPreset(it)
                    },
                    onDynamicColorChange = {
                        dynamicColor = it
                        preferences.saveDynamicColor(it)
                    }
                )
            }
        }
    }

    fun requestNotificationPermissionIfNeeded() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return
        if (checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED) return
        requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), NOTIFICATION_PERMISSION_REQUEST)
    }

    private companion object {
        const val NOTIFICATION_PERMISSION_REQUEST = 2001
    }
}

private data class AppAppearance(
    val themeMode: DeuteriumThemeMode,
    val colorPreset: DeuteriumColorPreset,
    val dynamicColor: Boolean
)

private class AppPreferences(context: Context) : SessionStore {
    private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    fun loadThemeMode(): DeuteriumThemeMode {
        return prefs.getString(KEY_THEME_MODE, null)
            ?.let { runCatching { DeuteriumThemeMode.valueOf(it) }.getOrNull() }
            ?: DeuteriumThemeMode.System
    }

    fun saveThemeMode(value: DeuteriumThemeMode) {
        prefs.edit().putString(KEY_THEME_MODE, value.name).apply()
    }

    fun loadColorPreset(): DeuteriumColorPreset {
        return prefs.getString(KEY_COLOR_PRESET, null)
            ?.let { runCatching { DeuteriumColorPreset.valueOf(it) }.getOrNull() }
            ?: DeuteriumColorPreset.Emerald
    }

    fun saveColorPreset(value: DeuteriumColorPreset) {
        prefs.edit().putString(KEY_COLOR_PRESET, value.name).apply()
    }

    fun loadDynamicColor(): Boolean {
        return prefs.getBoolean(KEY_DYNAMIC_COLOR, false)
    }

    fun saveDynamicColor(value: Boolean) {
        prefs.edit().putBoolean(KEY_DYNAMIC_COLOR, value).apply()
    }

    fun loadShowServerEvents(): Boolean {
        return prefs.getBoolean(KEY_SHOW_SERVER_EVENTS, false)
    }

    fun saveShowServerEvents(value: Boolean) {
        prefs.edit().putBoolean(KEY_SHOW_SERVER_EVENTS, value).apply()
    }

    fun loadWalletNotifications(): Boolean {
        return prefs.getBoolean(KEY_WALLET_NOTIFICATIONS, false)
    }

    fun saveWalletNotifications(value: Boolean) {
        prefs.edit().putBoolean(KEY_WALLET_NOTIFICATIONS, value).apply()
    }

    override fun loadToken(): String? {
        return prefs.getString(KEY_SESSION_TOKEN, null)?.takeIf { it.isNotBlank() }
    }

    override fun loadUser(): UserProfile? {
        val userId = prefs.getString(KEY_USER_ID, null)?.takeIf { it.isNotBlank() } ?: return null
        val playerRef = prefs.getString(KEY_USER_PLAYER_REF, null)?.takeIf { it.isNotBlank() } ?: return null
        val gameId = prefs.getString(KEY_USER_GAME_ID, null)?.takeIf { it.isNotBlank() } ?: return null
        val qq = prefs.getString(KEY_USER_QQ, null) ?: return null
        val identityStatus = prefs.getString(KEY_USER_IDENTITY_STATUS, null) ?: "bound"
        return UserProfile(userId, playerRef, gameId, qq, identityStatus)
    }

    override fun saveSession(token: String, user: UserProfile) {
        prefs.edit()
            .putString(KEY_SESSION_TOKEN, token)
            .apply()
        saveUser(user)
    }

    override fun saveUser(user: UserProfile) {
        prefs.edit()
            .putString(KEY_USER_ID, user.userId)
            .putString(KEY_USER_PLAYER_REF, user.playerRef)
            .putString(KEY_USER_GAME_ID, user.gameId)
            .putString(KEY_USER_QQ, user.qq)
            .putString(KEY_USER_IDENTITY_STATUS, user.identityStatus)
            .apply()
    }

    override fun clearSession() {
        prefs.edit()
            .remove(KEY_SESSION_TOKEN)
            .remove(KEY_USER_ID)
            .remove(KEY_USER_PLAYER_REF)
            .remove(KEY_USER_GAME_ID)
            .remove(KEY_USER_QQ)
            .remove(KEY_USER_IDENTITY_STATUS)
            .apply()
    }

    override fun loadLastWalletRecordId(userId: String): String? {
        return prefs.getString(walletRecordKey(userId), null)?.takeIf { it.isNotBlank() }
    }

    override fun saveLastWalletRecordId(userId: String, recordId: String) {
        prefs.edit().putString(walletRecordKey(userId), recordId).apply()
    }

    override fun loadPendingTransfer(userId: String): PendingTransferRequest? {
        val clientRequestId = prefs.getString(transferKey(userId, "client"), null)
            ?.takeIf { it.isNotBlank() }
            ?: return null
        val recipientPlayerRef = prefs.getString(transferKey(userId, "recipient"), null)
            ?.takeIf { it.isNotBlank() }
            ?: return null
        val amount = prefs.getString(transferKey(userId, "amount"), null)
            ?.takeIf { it.isNotBlank() }
            ?: return null
        val note = prefs.getString(transferKey(userId, "note"), null)?.takeIf { it.isNotBlank() }
        return PendingTransferRequest(clientRequestId, recipientPlayerRef, amount, note)
    }

    override fun savePendingTransfer(userId: String, pending: PendingTransferRequest) {
        prefs.edit()
            .putString(transferKey(userId, "client"), pending.clientRequestId)
            .putString(transferKey(userId, "recipient"), pending.recipientPlayerRef)
            .putString(transferKey(userId, "amount"), pending.amount)
            .putString(transferKey(userId, "note"), pending.note.orEmpty())
            .apply()
    }

    override fun clearPendingTransfer(userId: String) {
        prefs.edit()
            .remove(transferKey(userId, "client"))
            .remove(transferKey(userId, "recipient"))
            .remove(transferKey(userId, "amount"))
            .remove(transferKey(userId, "note"))
            .apply()
    }

    override fun loadPendingAiPurchase(userId: String, planId: String): PendingAiPurchaseRef? {
        val clientRequestId = prefs.getString(aiPurchaseKey(userId, planId, "client"), null)
            ?.takeIf { it.isNotBlank() }
            ?: return null
        val purchaseId = prefs.getString(aiPurchaseKey(userId, planId, "purchase"), null)
            ?.takeIf { it.isNotBlank() }
        return PendingAiPurchaseRef(clientRequestId, purchaseId)
    }

    override fun savePendingAiPurchase(userId: String, planId: String, pending: PendingAiPurchaseRef) {
        prefs.edit()
            .putString(aiPurchaseKey(userId, planId, "client"), pending.clientRequestId)
            .putString(aiPurchaseKey(userId, planId, "purchase"), pending.purchaseId.orEmpty())
            .apply()
    }

    override fun clearPendingAiPurchase(userId: String, planId: String) {
        prefs.edit()
            .remove(aiPurchaseKey(userId, planId, "client"))
            .remove(aiPurchaseKey(userId, planId, "purchase"))
            .apply()
    }

    private fun walletRecordKey(userId: String): String = "$KEY_LAST_WALLET_RECORD_ID_PREFIX$userId"
    private fun transferKey(userId: String, suffix: String): String =
        "$KEY_PENDING_TRANSFER_PREFIX${userId}_$suffix"
    private fun aiPurchaseKey(userId: String, planId: String, suffix: String): String =
        "$KEY_PENDING_AI_PURCHASE_PREFIX${userId}_${planId}_$suffix"

    private companion object {
        const val PREFS_NAME = "deuterium_app_preferences"
        const val KEY_THEME_MODE = "theme_mode"
        const val KEY_COLOR_PRESET = "color_preset"
        const val KEY_DYNAMIC_COLOR = "dynamic_color"
        const val KEY_SHOW_SERVER_EVENTS = "show_server_events"
        const val KEY_WALLET_NOTIFICATIONS = "wallet_notifications"
        const val KEY_SESSION_TOKEN = "session_token"
        const val KEY_USER_ID = "user_id"
        const val KEY_USER_PLAYER_REF = "user_player_ref"
        const val KEY_USER_GAME_ID = "user_game_id"
        const val KEY_USER_QQ = "user_qq"
        const val KEY_USER_IDENTITY_STATUS = "user_identity_status"
        const val KEY_LAST_WALLET_RECORD_ID_PREFIX = "last_wallet_record_id_"
        const val KEY_PENDING_TRANSFER_PREFIX = "pending_transfer_"
        const val KEY_PENDING_AI_PURCHASE_PREFIX = "pending_ai_purchase_"
    }
}

private data class PendingTransfer(
    val recipient: ResolvedPlayerRef?,
    val amount: String,
    val note: String
)

private data class MentionSelection(
    val playerRef: String,
    val gameId: String
)

private enum class AuthMode { Login, Register, ResetPassword }
private enum class MainSection { Wallet, Transfer, Chat, Ai, Profile }
private enum class TransferFeedbackPhase { Idle, Loading, Success, Notice, Error }

private object AppShapes {
    val small = RoundedCornerShape(12.dp)
    val medium = RoundedCornerShape(16.dp)
    val large = RoundedCornerShape(24.dp)
    val extraLarge = RoundedCornerShape(28.dp)
    val list = RoundedCornerShape(20.dp)
    val pill = RoundedCornerShape(999.dp)
}

private object AppMotion {
    const val PressedScale = 0.96f
    const val CanaryPressedScale = 0.98f
    const val Micro = 110
    const val CanaryFast = 130
    const val Fast = 160
    const val Page = 240
    const val Dialog = 280
    val Easing = FastOutSlowInEasing
}

private fun MainSection.motionOrder(): Int = when (this) {
    MainSection.Wallet -> 0
    MainSection.Chat -> 1
    MainSection.Ai -> 2
    MainSection.Profile -> 3
    MainSection.Transfer -> 4
}

private fun chatMatchesQuery(item: ChatFeedItem, query: String): Boolean {
    if (query.isBlank()) return true
    return item.sender.contains(query, ignoreCase = true) ||
        item.content.contains(query, ignoreCase = true) ||
        item.time.contains(query, ignoreCase = true)
}

private fun composeReplyContent(replyTarget: ChatFeedItem?, body: String): String {
    if (replyTarget == null) return body
    val quote = compactChatSnippet(replyTarget.content, 72)
    return "回复 ${replyTarget.sender}: $quote\n${body.trim()}"
}

private fun compactChatSnippet(text: String, maxChars: Int): String {
    val compact = text.replace(Regex("\\s+"), " ").trim()
    if (compact.length <= maxChars) return compact
    val safeMax = (maxChars - 3).coerceAtLeast(1)
    return compact.take(safeMax) + "..."
}

private fun playerInitial(gameId: String): String =
    gameId.trim().firstOrNull()?.uppercaseChar()?.toString() ?: "?"

@Composable
private fun XiaoxiangAvatar(modifier: Modifier = Modifier) {
    Image(
        painter = painterResource(id = R.drawable.xiaoxiang_avatar),
        contentDescription = null,
        contentScale = ContentScale.Crop,
        modifier = modifier.clip(CircleShape)
    )
}

@Composable
private fun DeuteriumApp(
    preferences: AppPreferences,
    chatHistoryStore: ChatHistoryStore,
    appearance: AppAppearance,
    onThemeModeChange: (DeuteriumThemeMode) -> Unit,
    onColorPresetChange: (DeuteriumColorPreset) -> Unit,
    onDynamicColorChange: (Boolean) -> Unit
) {
    val context = LocalContext.current
    val lifecycleOwner = context as? LifecycleOwner
    val notifier = remember(context) { AppNotifier(context.applicationContext) }
    val apiClient = remember { ApiClient { preferences.loadToken() } }
    val authRepository = remember { AuthRepository(apiClient, preferences) }
    val walletRepository = remember { WalletRepository(apiClient, preferences) }
    val appUpdateRepository = remember { AppUpdateRepository(apiClient) }
    val appScope = rememberCoroutineScope()
    var user by remember { mutableStateOf(preferences.loadUser()) }
    var showServerEvents by rememberSaveable { mutableStateOf(preferences.loadShowServerEvents()) }
    var walletNotifications by rememberSaveable { mutableStateOf(preferences.loadWalletNotifications()) }
    val chatRepository = remember {
        ChatRepository(
            apiClient = apiClient,
            sessionStore = preferences,
            historyStore = chatHistoryStore,
            onUnauthorized = { user = null },
            walletNotificationsEnabled = { preferences.loadWalletNotifications() },
            onNotificationPermissionNeeded = { (context as? MainActivity)?.requestNotificationPermissionIfNeeded() },
            onFollowedChatNotification = { notifier.notifyFollowedChat(it) },
            onWalletNotification = { notifier.notifyWalletRecord(it) },
            onMentionNotification = { notifier.notifyMention(it) },
            onWalletRecord = { walletRepository.mergeRecord(it) },
            onSocketConnected = {
                appScope.launch { walletRepository.syncNewRecords() }
            }
        )
    }
    val aiRepository = remember {
        AiRepository(
            apiClient = apiClient,
            sessionStore = preferences,
            onUnauthorized = { user = null }
        )
    }

    DisposableEffect(chatRepository) {
        onDispose { chatRepository.close() }
    }

    LaunchedEffect(Unit) {
        if (!preferences.loadToken().isNullOrBlank()) {
            when (val result = authRepository.restoreSession()) {
                is RepoResult.Success -> user = result.value
                is RepoResult.Error -> if (result.code == "UNAUTHORIZED") user = null
            }
        }
    }

    LaunchedEffect(user?.userId, showServerEvents) {
        if (user != null) {
            chatRepository.connect(showServerEvents)
            chatRepository.loadInitial()
            walletRepository.syncNewRecords()
        } else {
            chatRepository.disconnect()
            aiRepository.clearLocal()
        }
    }

    DisposableEffect(lifecycleOwner, user?.userId, showServerEvents) {
        if (lifecycleOwner == null || user == null) {
            onDispose {}
        } else {
            val observer = LifecycleEventObserver { _, event ->
                when (event) {
                    Lifecycle.Event.ON_RESUME -> {
                        chatRepository.connect(showServerEvents)
                        chatRepository.reportAppForeground(true)
                        appScope.launch {
                            chatRepository.refreshPresenceAndDirectory()
                            walletRepository.syncNewRecords()
                        }
                    }
                    Lifecycle.Event.ON_PAUSE -> chatRepository.reportAppForeground(false)
                    else -> Unit
                }
            }
            lifecycleOwner.lifecycle.addObserver(observer)
            chatRepository.reportAppForeground(lifecycleOwner.lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED))
            onDispose {
                lifecycleOwner.lifecycle.removeObserver(observer)
            }
        }
    }

    AppBackground {
        AnimatedContent(
            targetState = user,
            transitionSpec = {
                (fadeIn(tween(AppMotion.Fast, easing = AppMotion.Easing)) +
                    slideInHorizontally(tween(AppMotion.Page, easing = AppMotion.Easing)) { it / 8 })
                    .togetherWith(
                        fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing)) +
                            slideOutHorizontally(tween(AppMotion.Fast, easing = AppMotion.Easing)) { -it / 10 }
                    )
            },
            label = "SessionTransition"
        ) { activeUser ->
            if (activeUser == null) {
                AuthScreen(
                    repository = authRepository,
                    onAuthenticated = { user = it }
                )
            } else {
                MainShell(
                    user = activeUser,
                    authRepository = authRepository,
                    walletRepository = walletRepository,
                    chatRepository = chatRepository,
                    aiRepository = aiRepository,
                    appUpdateRepository = appUpdateRepository,
                    appearance = appearance,
                    onThemeModeChange = onThemeModeChange,
                    onColorPresetChange = onColorPresetChange,
                    onDynamicColorChange = onDynamicColorChange,
                    showServerEvents = showServerEvents,
                    onShowServerEventsChange = {
                        showServerEvents = it
                        preferences.saveShowServerEvents(it)
                    },
                    walletNotifications = walletNotifications,
                    onWalletNotificationsChange = {
                        walletNotifications = it
                        preferences.saveWalletNotifications(it)
                        if (it) (context as? MainActivity)?.requestNotificationPermissionIfNeeded()
                    },
                    onLogout = {
                        chatRepository.disconnect()
                        aiRepository.clearLocal()
                        user = null
                    }
                )
            }
        }
    }
}

@Composable
private fun AppBackground(content: @Composable () -> Unit) {
    Surface(
        modifier = Modifier.fillMaxSize(),
        color = MaterialTheme.colorScheme.background,
        contentColor = MaterialTheme.colorScheme.onBackground
    ) {
        content()
    }
}

@Composable
private fun AppSurface(
    modifier: Modifier = Modifier,
    shape: Shape = AppShapes.large,
    color: Color = MaterialTheme.colorScheme.surface,
    contentColor: Color = MaterialTheme.colorScheme.onSurface,
    content: @Composable () -> Unit
) {
    Surface(
        modifier = modifier,
        shape = shape,
        color = color,
        contentColor = contentColor,
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.55f)),
        tonalElevation = 0.dp,
        shadowElevation = 0.dp
    ) {
        content()
    }
}

@Composable
private fun AppIconButtonSurface(
    modifier: Modifier = Modifier,
    selected: Boolean = false,
    onClick: () -> Unit,
    content: @Composable () -> Unit
) {
    val colors = MaterialTheme.colorScheme
    val interactionSource = remember { MutableInteractionSource() }
    val pressed by interactionSource.collectIsPressedAsState()
    val scale by animateFloatAsState(
        targetValue = if (pressed) {
            if (BuildConfig.CANARY_UI) AppMotion.CanaryPressedScale else AppMotion.PressedScale
        } else {
            1f
        },
        animationSpec = spring(
            dampingRatio = if (BuildConfig.CANARY_UI) {
                Spring.DampingRatioNoBouncy
            } else {
                Spring.DampingRatioMediumBouncy
            },
            stiffness = Spring.StiffnessHigh
        ),
        label = "IconButtonPressScale"
    )
    AppSurface(
        modifier = modifier.graphicsLayer {
            scaleX = scale
            scaleY = scale
        },
        shape = AppShapes.pill,
        color = if (selected) colors.primaryContainer else colors.surfaceVariant,
        contentColor = if (selected) colors.onPrimaryContainer else colors.onSurfaceVariant
    ) {
        IconButton(
            onClick = onClick,
            modifier = Modifier.fillMaxSize(),
            interactionSource = interactionSource
        ) {
            content()
        }
    }
}

@Composable
private fun AuthScreen(
    repository: AuthRepository,
    onAuthenticated: (UserProfile) -> Unit
) {
    var mode by remember { mutableStateOf(AuthMode.Login) }
    val title = when (mode) {
        AuthMode.Login -> "登录 Deuterium"
        AuthMode.Register -> "注册并绑定服务器身份"
        AuthMode.ResetPassword -> "通过游戏内验证码重设密码"
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(20.dp),
        verticalArrangement = Arrangement.Center
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp)
        ) {
            Text(
                text = if (BuildConfig.CANARY_UI) "Deuterium VIII Canary" else "Deuterium VIII",
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Bold
            )
            if (BuildConfig.CANARY_UI) {
                Surface(
                    color = MaterialTheme.colorScheme.tertiaryContainer,
                    contentColor = MaterialTheme.colorScheme.onTertiaryContainer,
                    shape = AppShapes.pill,
                    tonalElevation = 0.dp,
                    shadowElevation = 0.dp
                ) {
                    Text(
                        BuildConfig.VERSION_NAME,
                        modifier = Modifier.padding(horizontal = 9.dp, vertical = 4.dp),
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
            }
        }
        Text(
            text = "账号、信用点与聊天的手机入口",
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Spacer(Modifier.height(24.dp))
        AppSurface(modifier = Modifier.fillMaxWidth(), shape = AppShapes.large) {
            Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
                Text(title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                AnimatedContent(
                    targetState = mode,
                    transitionSpec = {
                        val motion = if (BuildConfig.CANARY_UI) AppMotion.CanaryFast else AppMotion.Fast
                        val enterDistance = if (BuildConfig.CANARY_UI) 12 else 8
                        val exitDistance = if (BuildConfig.CANARY_UI) 14 else 10
                        (fadeIn(tween(motion, easing = AppMotion.Easing)) +
                            slideInHorizontally(tween(AppMotion.Page, easing = AppMotion.Easing)) { it / enterDistance })
                            .togetherWith(
                                fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing)) +
                                    slideOutHorizontally(tween(motion, easing = AppMotion.Easing)) { -it / exitDistance }
                            )
                    },
                    label = "AuthModeTransition"
                ) { targetMode ->
                    when (targetMode) {
                        AuthMode.Login -> LoginForm(
                            repository = repository,
                            onAuthenticated = onAuthenticated,
                            onForgotPassword = { mode = AuthMode.ResetPassword }
                        )
                        AuthMode.Register -> RegisterForm(repository, onAuthenticated)
                        AuthMode.ResetPassword -> ResetPasswordForm(repository) {
                            mode = AuthMode.Login
                        }
                    }
                }
                Divider()
                AuthModeButtons(
                    mode = mode,
                    onModeChange = { mode = it }
                )
            }
        }
    }
}

@Composable
private fun LoginForm(
    repository: AuthRepository,
    onAuthenticated: (UserProfile) -> Unit,
    onForgotPassword: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var account by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var message by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(false) }

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        OutlinedTextField(
            value = account,
            onValueChange = { account = it },
            label = { Text("玩家 ID 或 QQ 号") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth()
        )
        OutlinedTextField(
            value = password,
            onValueChange = { password = it },
            label = { Text("密码") },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            modifier = Modifier.fillMaxWidth()
        )
        TextButton(
            onClick = onForgotPassword,
            modifier = Modifier.align(Alignment.End)
        ) {
            Text("忘记密码？", style = MaterialTheme.typography.labelLarge)
        }
        PrimaryActionButton(
            text = "登录",
            icon = Icons.Filled.Login,
            onClick = {
                if (loading) return@PrimaryActionButton
                scope.launch {
                    loading = true
                    message = null
                    when (val result = repository.login(account, password)) {
                        is RepoResult.Success -> onAuthenticated(result.value)
                        is RepoResult.Error -> message = result.message
                    }
                    loading = false
                }
            }
        )
        InlineMessage(message)
    }
}

@Composable
private fun RegisterForm(repository: AuthRepository, onAuthenticated: (UserProfile) -> Unit) {
    val scope = rememberCoroutineScope()
    var gameId by remember { mutableStateOf("") }
    var qq by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var verificationToken by remember { mutableStateOf("") }
    var message by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(false) }

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        OutlinedTextField(gameId, { gameId = it }, label = { Text("游戏内 ID") }, singleLine = true, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(qq, { qq = it }, label = { Text("QQ 号") }, singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number), modifier = Modifier.fillMaxWidth())
        OutlinedTextField(password, { password = it }, label = { Text("密码") }, singleLine = true, visualTransformation = PasswordVisualTransformation(), modifier = Modifier.fillMaxWidth())
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
            OutlinedTextField(
                value = code,
                onValueChange = { if (it.length <= 6) code = it },
                label = { Text("验证码") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                modifier = Modifier.weight(1f)
            )
            OutlinedButton(onClick = {
                if (loading) return@OutlinedButton
                scope.launch {
                    loading = true
                    message = null
                    when (val result = repository.requestRegisterCode(gameId, qq, password)) {
                        is RepoResult.Success -> {
                            verificationToken = result.value.verificationToken
                            message = "验证码已发送到服务器私聊，请在有效期内输入。"
                        }
                        is RepoResult.Error -> message = result.message
                    }
                    loading = false
                }
            }) {
                Icon(Icons.Filled.Send, contentDescription = null)
                Spacer(Modifier.width(6.dp))
                Text("发送")
            }
        }
        PrimaryActionButton(
            text = "完成注册",
            icon = Icons.Filled.PersonAdd,
            onClick = {
                if (loading) return@PrimaryActionButton
                scope.launch {
                    loading = true
                    message = null
                    when (val result = repository.register(verificationToken, code, password)) {
                        is RepoResult.Success -> onAuthenticated(result.value)
                        is RepoResult.Error -> message = result.message
                    }
                    loading = false
                }
            }
        )
        InlineMessage(message)
    }
}

@Composable
private fun ResetPasswordForm(
    repository: AuthRepository,
    initialAccount: String = "",
    accountReadOnly: Boolean = false,
    onDone: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var account by remember(initialAccount) { mutableStateOf(initialAccount) }
    var code by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var verificationToken by remember { mutableStateOf("") }
    var message by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(false) }

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        OutlinedTextField(
            value = account,
            onValueChange = { if (!accountReadOnly) account = it },
            label = { Text("玩家 ID 或 QQ 号") },
            singleLine = true,
            enabled = !accountReadOnly,
            modifier = Modifier.fillMaxWidth()
        )
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
            OutlinedTextField(
                value = code,
                onValueChange = { if (it.length <= 6) code = it },
                label = { Text("验证码") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                modifier = Modifier.weight(1f)
            )
            OutlinedButton(onClick = {
                if (loading) return@OutlinedButton
                scope.launch {
                    loading = true
                    message = null
                    when (val result = repository.requestResetCode(account)) {
                        is RepoResult.Success -> {
                            verificationToken = result.value.verificationToken
                            message = "改密验证码已发送到服务器私聊，请在有效期内输入。"
                        }
                        is RepoResult.Error -> message = result.message
                    }
                    loading = false
                }
            }) {
                Icon(Icons.Filled.Send, contentDescription = null)
                Spacer(Modifier.width(6.dp))
                Text("发送")
            }
        }
        OutlinedTextField(password, { password = it }, label = { Text("新密码") }, singleLine = true, visualTransformation = PasswordVisualTransformation(), modifier = Modifier.fillMaxWidth())
        PrimaryActionButton(
            text = "重设密码",
            icon = Icons.Filled.Lock,
            onClick = {
                if (loading) return@PrimaryActionButton
                scope.launch {
                    loading = true
                    message = null
                    when (val result = repository.resetPassword(verificationToken, code, password)) {
                        is RepoResult.Success -> {
                            message = result.value
                            onDone()
                        }
                        is RepoResult.Error -> message = result.message
                    }
                    loading = false
                }
            }
        )
        InlineMessage(message)
    }
}

@Composable
private fun AuthModeButtons(mode: AuthMode, onModeChange: (AuthMode) -> Unit) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        modifier = Modifier.fillMaxWidth()
    ) {
        TextButton(
            onClick = { onModeChange(AuthMode.Login) },
            enabled = mode != AuthMode.Login
        ) {
            Icon(Icons.Filled.Login, contentDescription = null)
            Spacer(Modifier.width(4.dp))
            Text("登录")
        }
        TextButton(
            onClick = { onModeChange(AuthMode.Register) },
            enabled = mode != AuthMode.Register
        ) {
            Icon(Icons.Filled.PersonAdd, contentDescription = null)
            Spacer(Modifier.width(4.dp))
            Text("注册")
        }
    }
}

@Composable
private fun MainShell(
    user: UserProfile,
    authRepository: AuthRepository,
    walletRepository: WalletRepository,
    chatRepository: ChatRepository,
    aiRepository: AiRepository,
    appUpdateRepository: AppUpdateRepository,
    appearance: AppAppearance,
    onThemeModeChange: (DeuteriumThemeMode) -> Unit,
    onColorPresetChange: (DeuteriumColorPreset) -> Unit,
    onDynamicColorChange: (Boolean) -> Unit,
    showServerEvents: Boolean,
    onShowServerEventsChange: (Boolean) -> Unit,
    walletNotifications: Boolean,
    onWalletNotificationsChange: (Boolean) -> Unit,
    onLogout: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var section by remember { mutableStateOf(MainSection.Wallet) }
    var transferRecipient by remember { mutableStateOf<ResolvedPlayerRef?>(null) }
    val snackbarHostState = remember { SnackbarHostState() }
    val density = LocalDensity.current
    val imeVisible = WindowInsets.ime.getBottom(density) > 0
    val chatListState = rememberLazyListState()
    var chatFollowLatest by rememberSaveable { mutableStateOf(true) }
    var chatUnseenMessages by rememberSaveable { mutableStateOf(0) }
    var chatObservedVisibleCount by rememberSaveable { mutableStateOf(0) }
    var chatObservedLastVisibleId by rememberSaveable { mutableStateOf<String?>(null) }
    val bottomBarEnter = if (BuildConfig.CANARY_UI) {
        fadeIn(tween(AppMotion.CanaryFast, easing = AppMotion.Easing))
    } else {
        fadeIn(tween(AppMotion.Fast, easing = AppMotion.Easing)) +
            expandVertically(tween(AppMotion.Fast, easing = AppMotion.Easing), expandFrom = Alignment.Top)
    }
    val bottomBarExit = if (BuildConfig.CANARY_UI) {
        fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing))
    } else {
        fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing)) +
            shrinkVertically(tween(AppMotion.Fast, easing = AppMotion.Easing), shrinkTowards = Alignment.Top)
    }

    Scaffold(
        containerColor = Color.Transparent,
        topBar = {
            AppSurface(
                modifier = Modifier
                    .fillMaxWidth()
                    .windowInsetsPadding(WindowInsets.statusBars)
                    .padding(horizontal = 12.dp, vertical = 8.dp),
                shape = AppShapes.extraLarge
            ) {
                Row(
                    modifier = Modifier.padding(horizontal = 18.dp, vertical = 14.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Text(
                            if (BuildConfig.CANARY_UI) "Deuterium Canary" else "Deuterium",
                            style = MaterialTheme.typography.titleLarge,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                        Text(
                            text = user.gameId,
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                    }
                    if (BuildConfig.CANARY_UI) {
                        Surface(
                            color = MaterialTheme.colorScheme.tertiaryContainer,
                            contentColor = MaterialTheme.colorScheme.onTertiaryContainer,
                            shape = AppShapes.pill,
                            tonalElevation = 0.dp,
                            shadowElevation = 0.dp
                        ) {
                            Text(
                                BuildConfig.VERSION_NAME,
                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 5.dp),
                                style = MaterialTheme.typography.labelSmall,
                                fontWeight = FontWeight.SemiBold,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis
                            )
                        }
                    }
                }
            }
        },
        bottomBar = {
            AnimatedVisibility(
                visible = !imeVisible,
                enter = bottomBarEnter,
                exit = bottomBarExit
            ) {
                AppSurface(
                    modifier = Modifier
                        .fillMaxWidth()
                        .windowInsetsPadding(WindowInsets.navigationBars)
                        .padding(
                            horizontal = if (BuildConfig.CANARY_UI) 10.dp else 12.dp,
                            vertical = if (BuildConfig.CANARY_UI) 6.dp else 8.dp
                        ),
                    shape = AppShapes.extraLarge
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(if (BuildConfig.CANARY_UI) 64.dp else 72.dp)
                            .padding(horizontal = 6.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        BottomItem("钱包", Icons.Filled.Home, section == MainSection.Wallet) { section = MainSection.Wallet }
                        BottomItem("聊天", Icons.Filled.Chat, section == MainSection.Chat) { section = MainSection.Chat }
                        BottomItem("Saki", Icons.Filled.Chat, section == MainSection.Ai) { section = MainSection.Ai }
                        BottomItem("我的", Icons.Filled.Person, section == MainSection.Profile) { section = MainSection.Profile }
                    }
                }
            }
        },
        snackbarHost = { SnackbarHost(snackbarHostState) }
    ) { padding ->
        Box(
            Modifier
                .padding(padding)
                .fillMaxSize()
        ) {
            AnimatedContent(
                targetState = section,
                transitionSpec = {
                    val forward = targetState.motionOrder() >= initialState.motionOrder()
                    val enterDistance = if (BuildConfig.CANARY_UI) 11 else 7
                    val exitDistance = if (BuildConfig.CANARY_UI) 13 else 9
                    val enterOffset: (Int) -> Int = { if (forward) it / enterDistance else -it / enterDistance }
                    val exitOffset: (Int) -> Int = { if (forward) -it / exitDistance else it / exitDistance }
                    (fadeIn(tween(if (BuildConfig.CANARY_UI) AppMotion.CanaryFast else AppMotion.Fast, easing = AppMotion.Easing)) +
                        slideInHorizontally(tween(AppMotion.Page, easing = AppMotion.Easing), enterOffset))
                        .togetherWith(
                            fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing)) +
                                slideOutHorizontally(
                                    tween(if (BuildConfig.CANARY_UI) AppMotion.CanaryFast else AppMotion.Fast, easing = AppMotion.Easing),
                                    exitOffset
                                )
                        )
                },
                modifier = Modifier.fillMaxSize(),
                label = "MainSectionTransition"
            ) { targetSection ->
                when (targetSection) {
                    MainSection.Wallet -> WalletScreen(
                        repository = walletRepository,
                        onTransfer = {
                            transferRecipient = null
                            section = MainSection.Transfer
                        }
                    )
                    MainSection.Transfer -> TransferScreen(
                        repository = walletRepository,
                        initialRecipient = transferRecipient,
                        onBack = { section = MainSection.Wallet }
                    )
                    MainSection.Chat -> ChatScreen(
                        repository = chatRepository,
                        showServerEvents = showServerEvents,
                        listState = chatListState,
                        followLatest = chatFollowLatest,
                        onFollowLatestChange = { chatFollowLatest = it },
                        unseenMessages = chatUnseenMessages,
                        onUnseenMessagesChange = { chatUnseenMessages = it },
                        observedVisibleCount = chatObservedVisibleCount,
                        onObservedVisibleCountChange = { chatObservedVisibleCount = it },
                        observedLastVisibleId = chatObservedLastVisibleId,
                        onObservedLastVisibleIdChange = { chatObservedLastVisibleId = it },
                        onTransferToPlayer = {
                            transferRecipient = it
                            section = MainSection.Transfer
                        }
                    )
                    MainSection.Ai -> AiScreen(repository = aiRepository)
                    MainSection.Profile -> ProfileScreen(
                        user = user,
                        repository = authRepository,
                        chatRepository = chatRepository,
                        appUpdateRepository = appUpdateRepository,
                        appearance = appearance,
                        onThemeModeChange = onThemeModeChange,
                        onColorPresetChange = onColorPresetChange,
                        onDynamicColorChange = onDynamicColorChange,
                        showServerEvents = showServerEvents,
                        onShowServerEventsChange = onShowServerEventsChange,
                        walletNotifications = walletNotifications,
                        onWalletNotificationsChange = onWalletNotificationsChange,
                        onLogout = {
                            scope.launch {
                                authRepository.logout()
                                onLogout()
                            }
                        }
                    )
                }
            }
        }
    }
}

@Composable
private fun WalletScreen(repository: WalletRepository, onTransfer: () -> Unit) {
    var message by remember { mutableStateOf<String?>(null) }
    var autoDismissMessage by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val refreshBalance: () -> Unit = {
        scope.launch {
            message = when (val result = repository.refreshBalance()) {
                is RepoResult.Success -> {
                    autoDismissMessage = true
                    "余额已刷新：${result.value.amount}"
                }
                is RepoResult.Error -> {
                    autoDismissMessage = false
                    result.message
                }
            }
        }
    }

    LaunchedEffect(Unit) {
        when (val result = repository.loadWallet()) {
            is RepoResult.Success -> {
                message = null
                autoDismissMessage = false
            }
            is RepoResult.Error -> {
                message = result.message
                autoDismissMessage = false
            }
        }
    }

    if (BuildConfig.CANARY_UI) {
        CanaryWalletScreen(
            repository = repository,
            message = message,
            autoDismissMessage = autoDismissMessage,
            onDismissMessage = {
                message = null
                autoDismissMessage = false
            },
            onRefresh = refreshBalance,
            onTransfer = onTransfer
        )
        return
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(18.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp)
    ) {
        item {
            AppSurface(
                color = MaterialTheme.colorScheme.primaryContainer,
                contentColor = MaterialTheme.colorScheme.onPrimaryContainer,
                shape = AppShapes.large,
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text("信用点余额", style = MaterialTheme.typography.titleMedium)
                    Text(
                        text = repository.balance?.amount ?: "正在加载",
                        style = MaterialTheme.typography.displaySmall,
                        fontWeight = FontWeight.Bold
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        Button(onClick = refreshBalance) {
                            Icon(Icons.Filled.Refresh, contentDescription = null)
                            Spacer(Modifier.width(6.dp))
                            Text("刷新")
                        }
                    }
                }
            }
        }
        item {
            PrimaryActionButton("发起转账", Icons.Filled.Add, onTransfer)
            InlineMessage(
                message = message,
                autoDismiss = autoDismissMessage,
                onDismiss = {
                    message = null
                    autoDismissMessage = false
                }
            )
        }
        item {
            Text("最近流水", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
        }
        items(
            items = repository.records,
            key = { it.recordId },
            contentType = { "wallet-record" }
        ) { record ->
            WalletRecordRow(record, Modifier.animateItem())
        }
    }
}

@Composable
private fun CanaryWalletScreen(
    repository: WalletRepository,
    message: String?,
    autoDismissMessage: Boolean,
    onDismissMessage: () -> Unit,
    onRefresh: () -> Unit,
    onTransfer: () -> Unit
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = 10.dp, top = 8.dp, end = 10.dp, bottom = 14.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp)
    ) {
        item {
            AppSurface(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(28.dp),
                color = MaterialTheme.colorScheme.surface
            ) {
                Column(
                    modifier = Modifier.padding(18.dp),
                    verticalArrangement = Arrangement.spacedBy(14.dp)
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        PlayerAvatar(
                            gameId = "D",
                            statusColor = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.size(48.dp)
                        )
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Text(
                                "Deuterium Wallet",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.SemiBold
                            )
                            Text(
                                "服务器信用点余额",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    }
                    Text(
                        text = repository.balance?.amount ?: "正在加载",
                        style = MaterialTheme.typography.displayMedium,
                        fontWeight = FontWeight.Bold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        CanaryWalletAction(
                            label = "刷新",
                            icon = Icons.Filled.Refresh,
                            modifier = Modifier.weight(1f),
                            onClick = onRefresh
                        )
                        CanaryWalletAction(
                            label = "转账",
                            icon = Icons.Filled.Send,
                            modifier = Modifier.weight(1f),
                            onClick = onTransfer
                        )
                    }
                }
            }
        }
        if (message != null) {
            item {
                InlineMessage(
                    message = message,
                    autoDismiss = autoDismissMessage,
                    onDismiss = onDismissMessage
                )
            }
        }
        item {
            Text(
                "最近流水",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(start = 6.dp, top = 2.dp, end = 6.dp)
            )
        }
        if (repository.records.isEmpty()) {
            item {
                AppSurface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = AppShapes.large,
                    color = MaterialTheme.colorScheme.surface
                ) {
                    Text(
                        "暂无流水",
                        modifier = Modifier.padding(18.dp),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        }
        items(
            items = repository.records,
            key = { it.recordId },
            contentType = { "canary-wallet-record" }
        ) { record ->
            CanaryWalletRecordRow(record, Modifier.animateItem())
        }
    }
}

@Composable
private fun CanaryWalletAction(
    label: String,
    icon: ImageVector,
    modifier: Modifier = Modifier,
    onClick: () -> Unit
) {
    AppSurface(
        modifier = modifier
            .height(48.dp)
            .clickable(onClick = onClick),
        shape = AppShapes.pill,
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.82f)
    ) {
        Row(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 14.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.Center
        ) {
            Icon(icon, contentDescription = null, modifier = Modifier.size(19.dp))
            Spacer(Modifier.width(7.dp))
            Text(label, fontWeight = FontWeight.SemiBold)
        }
    }
}

@Composable
private fun CanaryWalletRecordRow(record: WalletRecord, modifier: Modifier = Modifier) {
    val positive = record.direction == "income"
    val sign = if (positive) "+" else "-"
    val amountColor = if (positive) Color(0xFF2E7D32) else MaterialTheme.colorScheme.error
    AppSurface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(22.dp),
        color = MaterialTheme.colorScheme.surface
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 11.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(42.dp)
                    .clip(CircleShape)
                    .background(if (positive) Color(0xFFE8F5E9) else MaterialTheme.colorScheme.errorContainer),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    if (positive) Icons.Filled.Add else Icons.Filled.Send,
                    contentDescription = null,
                    tint = amountColor,
                    modifier = Modifier.size(21.dp)
                )
            }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(
                    record.otherPlayer.gameId,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    listOfNotNull(formatIsoDateTimeUtc8(record.occurredAt), record.status, record.note).joinToString(" · "),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }
            Text(
                text = "$sign${record.amount}",
                fontWeight = FontWeight.Bold,
                color = amountColor,
                maxLines = 1
            )
        }
    }
}

@Composable
private fun TransferScreen(
    repository: WalletRepository,
    initialRecipient: ResolvedPlayerRef?,
    onBack: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var query by remember(initialRecipient) { mutableStateOf(initialRecipient?.gameId ?: "") }
    var recipient by remember(initialRecipient) { mutableStateOf(initialRecipient) }
    var recipientSearchJob by remember { mutableStateOf<Job?>(null) }
    var recipientSearchGeneration by remember { mutableStateOf(0) }
    var recipientSearching by remember { mutableStateOf(false) }
    var amount by remember { mutableStateOf("") }
    var note by remember { mutableStateOf("") }
    var message by remember { mutableStateOf<String?>(null) }
    var showConfirm by remember { mutableStateOf(false) }
    var pendingTransfer by remember { mutableStateOf<PendingTransfer?>(null) }
    var feedbackPhase by remember { mutableStateOf(TransferFeedbackPhase.Idle) }
    var feedbackMessage by remember { mutableStateOf("") }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(if (BuildConfig.CANARY_UI) 10.dp else 18.dp),
        verticalArrangement = Arrangement.spacedBy(if (BuildConfig.CANARY_UI) 10.dp else 14.dp)
    ) {
        if (BuildConfig.CANARY_UI) {
            AppSurface(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(28.dp),
                color = MaterialTheme.colorScheme.surface
            ) {
                Row(
                    modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    AppIconButtonSurface(
                        modifier = Modifier.size(46.dp),
                        onClick = onBack
                    ) {
                        Icon(Icons.Filled.Close, contentDescription = "返回钱包", modifier = Modifier.size(22.dp))
                    }
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Text("转账", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                        Text(
                            "Deuterium Pay",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            }
        } else {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("转账", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f))
                TextButton(onClick = onBack) { Text("返回钱包") }
            }
        }
        AppSurface(
            modifier = Modifier.fillMaxWidth(),
            shape = if (BuildConfig.CANARY_UI) RoundedCornerShape(26.dp) else AppShapes.large,
            color = MaterialTheme.colorScheme.surface
        ) {
            Column(
                Modifier.padding(if (BuildConfig.CANARY_UI) 14.dp else 16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                Text(if (BuildConfig.CANARY_UI) "收款人" else "收款玩家", style = MaterialTheme.typography.titleMedium)
                Row(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
                    OutlinedTextField(
                        value = query,
                        onValueChange = {
                            query = it
                            recipient = null
                            message = null
                            recipientSearchGeneration += 1
                            recipientSearchJob?.cancel()
                            recipientSearching = false
                        },
                        label = { Text("游戏内 ID 或 QQ") },
                        singleLine = true,
                        modifier = Modifier.weight(1f)
                    )
                    OutlinedButton(
                        enabled = query.isNotBlank() && !recipientSearching,
                        onClick = {
                            val searchedQuery = query.trim()
                            recipientSearchGeneration += 1
                            val generation = recipientSearchGeneration
                            recipientSearchJob?.cancel()
                            recipientSearching = true
                            recipientSearchJob = scope.launch {
                                val result = repository.findRecipient(searchedQuery)
                                if (generation != recipientSearchGeneration || query.trim() != searchedQuery) {
                                    return@launch
                                }
                                recipientSearching = false
                                when (result) {
                                is RepoResult.Success -> {
                                    recipient = result.value
                                    message = "已确认收款方：${result.value.gameId}"
                                }
                                is RepoResult.Error -> {
                                    recipient = null
                                    message = result.message
                                }
                            }
                        }
                    }) {
                        Icon(Icons.Filled.Search, contentDescription = null)
                    }
                }
                AnimatedVisibility(
                    visible = recipient != null,
                    enter = if (BuildConfig.CANARY_UI) {
                        fadeIn(tween(AppMotion.CanaryFast, easing = AppMotion.Easing))
                    } else {
                        fadeIn(tween(AppMotion.Fast, easing = AppMotion.Easing)) +
                            expandVertically(tween(AppMotion.Fast, easing = AppMotion.Easing))
                    },
                    exit = if (BuildConfig.CANARY_UI) {
                        fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing))
                    } else {
                        fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing)) +
                            shrinkVertically(tween(AppMotion.Fast, easing = AppMotion.Easing))
                    }
                ) {
                    recipient?.let {
                        PlayerSummaryCard(player = playerSummary(it))
                    }
                }
            }
        }
        AppSurface(
            modifier = Modifier.fillMaxWidth(),
            shape = if (BuildConfig.CANARY_UI) RoundedCornerShape(26.dp) else AppShapes.large,
            color = MaterialTheme.colorScheme.surface
        ) {
            Column(Modifier.padding(if (BuildConfig.CANARY_UI) 14.dp else 16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("转账信息", style = MaterialTheme.typography.titleMedium)
                OutlinedTextField(
                    value = amount,
                    onValueChange = { amount = it },
                    label = { Text("金额") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                    modifier = Modifier.fillMaxWidth()
                )
                OutlinedTextField(
                    value = note,
                    onValueChange = { note = it },
                    label = { Text("备注，可选") },
                    modifier = Modifier.fillMaxWidth()
                )
                PrimaryActionButton("二次确认", Icons.Filled.Check) {
                    val amountError = validateAmount(amount)
                    message = when {
                        recipient == null -> "请先确认收款玩家。"
                        amountError != null -> amountError
                        else -> {
                            showConfirm = true
                            null
                        }
                    }
                }
            }
        }
        InlineMessage(message)
    }

    if (showConfirm) {
        val submitTransfer: () -> Unit = {
            showConfirm = false
            message = null
            pendingTransfer = PendingTransfer(recipient, amount, note)
            feedbackMessage = "正在确认转账"
            feedbackPhase = TransferFeedbackPhase.Loading
            scope.launch {
                val pending = PendingTransfer(recipient, amount, note)
                when (val result = repository.transfer(pending.recipient, pending.amount, pending.note)) {
                    is RepoResult.Success -> {
                        val transfer = result.value
                        feedbackMessage = when (transfer.status) {
                            "success" -> "已转出 ${transfer.amount} 信用点"
                            "processing" -> "转账正在处理中，请稍后查看余额和流水。"
                            "unknown" -> "转账结果暂未确认，请稍后查看余额和流水。"
                            else -> "转账未完成，请查看余额和流水确认结果。"
                        }
                        feedbackPhase = if (transfer.status == "success") {
                            TransferFeedbackPhase.Success
                        } else {
                            TransferFeedbackPhase.Notice
                        }
                    }
                    is RepoResult.Error -> {
                        feedbackMessage = result.message
                        feedbackPhase = TransferFeedbackPhase.Error
                    }
                }
            }
        }
        if (BuildConfig.CANARY_UI) {
            CanaryTransferConfirmSheet(
                recipient = recipient,
                amount = amount,
                note = note,
                onConfirm = submitTransfer,
                onDismiss = { showConfirm = false }
            )
        } else {
            AlertDialog(
                onDismissRequest = { showConfirm = false },
                title = { Text("确认转账") },
                text = {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("收款方：${recipient?.gameId.orEmpty()}")
                        Text("金额：${amount.ifBlank { "0" }} 信用点")
                        Text("备注：${note.ifBlank { "无" }}")
                    }
                },
                confirmButton = {
                    Button(onClick = submitTransfer) {
                        Text("确认提交")
                    }
                },
                dismissButton = {
                    TextButton(onClick = { showConfirm = false }) {
                        Text("继续修改")
                    }
                }
            )
        }
    }

    if (feedbackPhase != TransferFeedbackPhase.Idle) {
        if (BuildConfig.CANARY_UI) {
            CanaryTransferFeedbackSheet(
                phase = feedbackPhase,
                message = feedbackMessage,
                onDismiss = {
                    feedbackPhase = TransferFeedbackPhase.Idle
                    pendingTransfer = null
                }
            )
        } else {
            TransferFeedbackDialog(
                phase = feedbackPhase,
                message = feedbackMessage,
                onDismiss = {
                    feedbackPhase = TransferFeedbackPhase.Idle
                    pendingTransfer = null
                }
            )
        }
    }
}

@Composable
private fun CanaryTransferConfirmSheet(
    recipient: ResolvedPlayerRef?,
    amount: String,
    note: String,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit
) {
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 10.dp, vertical = 12.dp),
            contentAlignment = Alignment.BottomCenter
        ) {
            Surface(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(topStart = 30.dp, topEnd = 30.dp, bottomStart = 22.dp, bottomEnd = 22.dp),
                color = MaterialTheme.colorScheme.surface,
                border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.55f)),
                tonalElevation = 0.dp,
                shadowElevation = 0.dp
            ) {
                Column(
                    modifier = Modifier.padding(18.dp),
                    verticalArrangement = Arrangement.spacedBy(14.dp)
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        PlayerAvatar(
                            gameId = recipient?.gameId ?: "?",
                            statusColor = if (recipient?.online == true) Color(0xFF2E7D32) else MaterialTheme.colorScheme.outline,
                            modifier = Modifier.size(48.dp)
                        )
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                            Text("确认转账", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                            Text(
                                recipient?.gameId ?: "未选择收款人",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    }
                    CanaryTransferSummaryRow("金额", "${amount.ifBlank { "0" }} 信用点")
                    CanaryTransferSummaryRow("备注", note.ifBlank { "无" })
                    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        OutlinedButton(
                            onClick = onDismiss,
                            modifier = Modifier
                                .weight(1f)
                                .height(48.dp)
                        ) {
                            Text("继续修改")
                        }
                        Button(
                            onClick = onConfirm,
                            modifier = Modifier
                                .weight(1f)
                                .height(48.dp),
                            shape = AppShapes.pill
                        ) {
                            Text("确认提交")
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun CanaryTransferSummaryRow(label: String, value: String) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            label,
            modifier = Modifier.weight(1f),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Text(
            value,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.SemiBold,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis
        )
    }
}

@Composable
private fun CanaryTransferFeedbackSheet(
    phase: TransferFeedbackPhase,
    message: String,
    onDismiss: () -> Unit
) {
    val loading = phase == TransferFeedbackPhase.Loading
    Dialog(
        onDismissRequest = { if (!loading) onDismiss() },
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 10.dp, vertical = 12.dp),
            contentAlignment = Alignment.BottomCenter
        ) {
            Surface(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(topStart = 30.dp, topEnd = 30.dp, bottomStart = 22.dp, bottomEnd = 22.dp),
                color = MaterialTheme.colorScheme.surface,
                border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.55f)),
                tonalElevation = 0.dp,
                shadowElevation = 0.dp
            ) {
                Column(
                    modifier = Modifier.padding(18.dp),
                    verticalArrangement = Arrangement.spacedBy(14.dp),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    val iconColor = when (phase) {
                        TransferFeedbackPhase.Success -> Color(0xFF2E7D32)
                        TransferFeedbackPhase.Error -> MaterialTheme.colorScheme.error
                        TransferFeedbackPhase.Notice -> MaterialTheme.colorScheme.secondary
                        else -> MaterialTheme.colorScheme.primary
                    }
                    Box(
                        modifier = Modifier
                            .size(58.dp)
                            .clip(CircleShape)
                            .background(iconColor.copy(alpha = 0.14f)),
                        contentAlignment = Alignment.Center
                    ) {
                        Icon(
                            when (phase) {
                                TransferFeedbackPhase.Success -> Icons.Filled.Check
                                TransferFeedbackPhase.Error -> Icons.Filled.Close
                                else -> Icons.Filled.Refresh
                            },
                            contentDescription = null,
                            tint = iconColor,
                            modifier = Modifier.size(30.dp)
                        )
                    }
                    Text(
                        when (phase) {
                            TransferFeedbackPhase.Loading -> "正在转账"
                            TransferFeedbackPhase.Success -> "转账成功"
                            TransferFeedbackPhase.Notice -> "结果待确认"
                            TransferFeedbackPhase.Error -> "转账失败"
                            TransferFeedbackPhase.Idle -> ""
                        },
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.SemiBold
                    )
                    Text(
                        message,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    if (!loading) {
                        Button(
                            onClick = onDismiss,
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(48.dp),
                            shape = AppShapes.pill
                        ) {
                            Text("完成")
                        }
                    }
                }
            }
        }
    }
}

/*
 * Dev keeps the original transfer feedback dialog below. Canary uses the
 * bottom-sheet variant above to keep the experiment isolated.
 */
@Composable
private fun TransferFeedbackDialog(
    phase: TransferFeedbackPhase,
    message: String,
    onDismiss: () -> Unit
) {
    val loading = phase == TransferFeedbackPhase.Loading
    val success = phase == TransferFeedbackPhase.Success
    val loadingTransition = rememberInfiniteTransition(label = "TransferLoadingTransition")
    val loadingRotation by loadingTransition.animateFloat(
        initialValue = 0f,
        targetValue = 360f,
        animationSpec = infiniteRepeatable(
            animation = tween(720, easing = LinearEasing),
            repeatMode = RepeatMode.Restart
        ),
        label = "TransferLoadingRotation"
    )
    val iconScale by animateFloatAsState(
        targetValue = when (phase) {
            TransferFeedbackPhase.Success -> 1.12f
            TransferFeedbackPhase.Error -> 1.02f
            else -> 1f
        },
        animationSpec = spring(
            dampingRatio = Spring.DampingRatioMediumBouncy,
            stiffness = Spring.StiffnessMedium
        ),
        label = "TransferFeedbackIconScale"
    )
    val errorShake = remember { Animatable(0f) }

    LaunchedEffect(phase) {
        if (phase == TransferFeedbackPhase.Error) {
            errorShake.snapTo(0f)
            repeat(3) {
                errorShake.animateTo(7f, tween(42, easing = LinearEasing))
                errorShake.animateTo(-7f, tween(42, easing = LinearEasing))
            }
            errorShake.animateTo(0f, tween(42, easing = LinearEasing))
        } else {
            errorShake.snapTo(0f)
        }
    }

    val iconMotion = Modifier.graphicsLayer {
        rotationZ = if (loading) loadingRotation else 0f
        scaleX = iconScale
        scaleY = iconScale
        translationX = errorShake.value
    }

    AlertDialog(
        onDismissRequest = { if (!loading) onDismiss() },
        icon = {
            when (phase) {
                TransferFeedbackPhase.Loading -> Box(
                    modifier = iconMotion
                        .size(62.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.primaryContainer),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        Icons.Filled.Refresh,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.onPrimaryContainer,
                        modifier = Modifier.size(34.dp)
                    )
                }
                TransferFeedbackPhase.Success -> Box(
                    modifier = iconMotion
                        .size(62.dp)
                        .clip(CircleShape)
                        .background(Color(0xFF2E7D32)),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        Icons.Filled.Check,
                        contentDescription = null,
                        tint = Color.White,
                        modifier = Modifier.size(38.dp)
                    )
                }
                TransferFeedbackPhase.Notice -> Box(
                    modifier = iconMotion
                        .size(62.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.secondaryContainer),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        Icons.Filled.Refresh,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.onSecondaryContainer,
                        modifier = Modifier.size(34.dp)
                    )
                }
                TransferFeedbackPhase.Error -> Box(
                    modifier = iconMotion
                        .size(62.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.errorContainer),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        Icons.Filled.Close,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.onErrorContainer,
                        modifier = Modifier.size(34.dp)
                    )
                }
                TransferFeedbackPhase.Idle -> Spacer(Modifier.size(54.dp))
            }
        },
        title = {
            Text(
                when (phase) {
                    TransferFeedbackPhase.Loading -> "正在转账"
                    TransferFeedbackPhase.Success -> "转账成功"
                    TransferFeedbackPhase.Notice -> "结果待确认"
                    TransferFeedbackPhase.Error -> "转账失败"
                    TransferFeedbackPhase.Idle -> ""
                }
            )
        },
        text = {
            Text(
                message,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        },
        confirmButton = {
            if (!loading) {
                Button(onClick = onDismiss) {
                    Text(if (success) "完成" else "返回修改")
                }
            }
        }
    )
}

@Composable
private fun ChatScreen(
    repository: ChatRepository,
    showServerEvents: Boolean,
    listState: LazyListState,
    followLatest: Boolean,
    onFollowLatestChange: (Boolean) -> Unit,
    unseenMessages: Int,
    onUnseenMessagesChange: (Int) -> Unit,
    observedVisibleCount: Int,
    onObservedVisibleCountChange: (Int) -> Unit,
    observedLastVisibleId: String?,
    onObservedLastVisibleIdChange: (String?) -> Unit,
    onTransferToPlayer: (ResolvedPlayerRef) -> Unit
) {
    val clipboardManager = LocalClipboardManager.current
    val scope = rememberCoroutineScope()
    var input by remember { mutableStateOf(TextFieldValue("")) }
    var message by remember { mutableStateOf<String?>(null) }
    var autoDismissMessage by remember { mutableStateOf(false) }
    var selectedPlayer by remember { mutableStateOf<PlayerDirectoryItem?>(null) }
    var selectedMentions by remember { mutableStateOf<List<MentionSelection>>(emptyList()) }
    var showMentions by remember { mutableStateOf(false) }
    var showPlayerDirectory by remember { mutableStateOf(false) }
    var chatSearchExpanded by rememberSaveable { mutableStateOf(false) }
    var chatSearchQuery by rememberSaveable { mutableStateOf("") }
    var replyTarget by remember { mutableStateOf<ChatFeedItem?>(null) }
    var actionMessage by remember { mutableStateOf<ChatFeedItem?>(null) }
    val density = LocalDensity.current
    val keyboardPadding = with(density) {
        (
            WindowInsets.ime.getBottom(this) -
                WindowInsets.navigationBars.getBottom(this)
            )
            .coerceAtLeast(0)
            .toDp()
    }
    val mentionQuery by remember { derivedStateOf { activeMentionQuery(input) } }
    val mentionPlayers by remember { derivedStateOf { repository.mentionCandidates(mentionQuery) } }
    val visibleMessages by remember(showServerEvents) {
        derivedStateOf { repository.messages.filter { showServerEvents || !it.event } }
    }
    val searchActive by remember {
        derivedStateOf { BuildConfig.CANARY_UI && chatSearchQuery.isNotBlank() }
    }
    val displayedMessages by remember {
        derivedStateOf {
            if (!searchActive) {
                visibleMessages
            } else {
                val query = chatSearchQuery.trim()
                visibleMessages.filter { chatMatchesQuery(it, query) }
            }
        }
    }
    val displayedOnlineCount by remember {
        derivedStateOf { repository.onlineCount ?: repository.playerDirectory.count { it.serverOnline } }
    }
    val showMentionSuggestions by remember {
        derivedStateOf { (showMentions || mentionQuery != null) && mentionPlayers.isNotEmpty() }
    }

    LaunchedEffect(listState) {
        snapshotFlow {
            val layout = listState.layoutInfo
            val lastVisibleIndex = layout.visibleItemsInfo.lastOrNull()?.index ?: -1
            val atBottom = layout.totalItemsCount == 0 || lastVisibleIndex >= layout.totalItemsCount - 1
            listState.isScrollInProgress to atBottom
        }.collect { (scrolling, atBottom) ->
            if (scrolling) {
                onFollowLatestChange(atBottom)
            } else if (atBottom) {
                onFollowLatestChange(true)
                onUnseenMessagesChange(0)
            }
        }
    }

    LaunchedEffect(visibleMessages.size, visibleMessages.lastOrNull()?.id, searchActive) {
        val currentLastId = visibleMessages.lastOrNull()?.id
        val insertedCount = (visibleMessages.size - observedVisibleCount).coerceAtLeast(0)
        if (observedLastVisibleId != null && currentLastId != observedLastVisibleId && insertedCount > 0 && !followLatest) {
            onUnseenMessagesChange(unseenMessages + insertedCount)
        }
        if (!searchActive && followLatest && visibleMessages.isNotEmpty()) {
            val targetIndex = displayedMessages.lastIndex
            val lastVisibleIndex = listState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: -1
            if (lastVisibleIndex < targetIndex) {
                listState.scrollToItem(targetIndex)
            }
            onUnseenMessagesChange(0)
        }
        onObservedVisibleCountChange(visibleMessages.size)
        onObservedLastVisibleIdChange(currentLastId)
    }

    LaunchedEffect(repository) {
        if (repository.messages.isEmpty()) {
            repository.syncRecentMessages()
        }
        repository.refreshPresenceAndDirectory()
        repository.refreshFollows()
    }

    LaunchedEffect(repository) {
        while (isActive) {
            delay(15_000L)
            if (repository.appForeground) repository.syncRecentMessages()
        }
    }

    Box(Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(bottom = keyboardPadding)
        ) {
            AppSurface(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(
                        min = if (BuildConfig.CANARY_UI) 64.dp else 48.dp,
                        max = if (BuildConfig.CANARY_UI && chatSearchExpanded) 136.dp else 64.dp
                    )
                    .padding(
                        horizontal = if (BuildConfig.CANARY_UI) 10.dp else 12.dp,
                        vertical = 4.dp
                    ),
                shape = if (BuildConfig.CANARY_UI) {
                    RoundedCornerShape(28.dp)
                } else {
                    AppShapes.pill
                }
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(
                            horizontal = if (BuildConfig.CANARY_UI) 10.dp else 6.dp,
                            vertical = if (BuildConfig.CANARY_UI) 8.dp else 4.dp
                        ),
                    verticalArrangement = Arrangement.spacedBy(if (BuildConfig.CANARY_UI) 8.dp else 6.dp)
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(if (BuildConfig.CANARY_UI) 48.dp else 40.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        AppIconButtonSurface(
                            modifier = Modifier
                                .height(if (BuildConfig.CANARY_UI) 48.dp else 40.dp)
                                .width(if (BuildConfig.CANARY_UI) 92.dp else 76.dp),
                            onClick = { showPlayerDirectory = true }
                        ) {
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.spacedBy(5.dp)
                            ) {
                                Icon(Icons.Filled.People, contentDescription = null, modifier = Modifier.size(18.dp))
                                Text(
                                    displayedOnlineCount.toString(),
                                    style = MaterialTheme.typography.labelLarge,
                                    fontWeight = FontWeight.SemiBold
                                )
                            }
                        }
                        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(1.dp)) {
                            Text(
                                "公共聊天",
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = FontWeight.SemiBold,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis
                            )
                            if (BuildConfig.CANARY_UI && searchActive) {
                                Text(
                                    "${displayedMessages.size} 条匹配",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                        }
                        if (BuildConfig.CANARY_UI) {
                            AppIconButtonSurface(
                                modifier = Modifier.size(48.dp),
                                selected = chatSearchExpanded,
                                onClick = {
                                    chatSearchExpanded = !chatSearchExpanded
                                    if (!chatSearchExpanded) chatSearchQuery = ""
                                }
                            ) {
                                Icon(
                                    if (chatSearchExpanded) Icons.Filled.Close else Icons.Filled.Search,
                                    contentDescription = if (chatSearchExpanded) "关闭聊天搜索" else "搜索当前聊天",
                                    modifier = Modifier.size(20.dp)
                                )
                            }
                        }
                    }
                    if (BuildConfig.CANARY_UI) {
                        AnimatedVisibility(
                            visible = chatSearchExpanded,
                            enter = fadeIn(tween(AppMotion.CanaryFast, easing = AppMotion.Easing)),
                            exit = fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing))
                        ) {
                            BasicTextField(
                                value = chatSearchQuery,
                                onValueChange = { chatSearchQuery = it },
                                singleLine = true,
                                textStyle = MaterialTheme.typography.bodyMedium.copy(
                                    color = MaterialTheme.colorScheme.onSurface
                                ),
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .height(46.dp)
                                    .clip(AppShapes.pill)
                                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.78f))
                                    .padding(horizontal = 16.dp)
                            ) { innerTextField ->
                                Box(
                                    Modifier.fillMaxSize(),
                                    contentAlignment = Alignment.CenterStart
                                ) {
                                    if (chatSearchQuery.isBlank()) {
                                        Text(
                                            "搜索玩家、内容或时间",
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                            style = MaterialTheme.typography.bodyMedium,
                                            maxLines = 1,
                                            overflow = TextOverflow.Ellipsis
                                        )
                                    }
                                    innerTextField()
                                }
                            }
                        }
                    }
                }
            }

            LazyColumn(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth(),
                state = listState,
                contentPadding = PaddingValues(
                    start = if (BuildConfig.CANARY_UI) 10.dp else 14.dp,
                    top = if (BuildConfig.CANARY_UI) 8.dp else 10.dp,
                    end = if (BuildConfig.CANARY_UI) 10.dp else 14.dp,
                    bottom = 8.dp
                ),
                verticalArrangement = Arrangement.spacedBy(if (BuildConfig.CANARY_UI) 6.dp else 8.dp)
            ) {
                items(
                    items = displayedMessages,
                    key = { it.id },
                    contentType = { if (it.event) "chat-event" else if (it.mine) "chat-mine" else "chat-other" }
                ) { chat ->
                    val itemModifier = if (BuildConfig.CANARY_UI) {
                        Modifier.animateItem(
                            fadeInSpec = tween(AppMotion.CanaryFast, easing = AppMotion.Easing),
                            placementSpec = spring(
                                dampingRatio = Spring.DampingRatioNoBouncy,
                                stiffness = Spring.StiffnessMedium
                            ),
                            fadeOutSpec = tween(AppMotion.Micro, easing = AppMotion.Easing)
                        )
                    } else {
                        Modifier
                    }
                    ChatBubble(
                        message = chat,
                        modifier = itemModifier,
                        onLongPress = {
                            if (BuildConfig.CANARY_UI) {
                                actionMessage = chat
                            } else {
                                clipboardManager.setText(AnnotatedString(chat.content))
                                message = if (chat.event) "已复制服务器事件。" else "已复制 ${chat.sender} 的消息。"
                                autoDismissMessage = true
                            }
                        }
                    )
                }
                if (BuildConfig.CANARY_UI && searchActive && displayedMessages.isEmpty()) {
                    item(
                        key = "chat-search-empty",
                        contentType = "chat-search-empty"
                    ) {
                        AppSurface(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp, vertical = 18.dp),
                            shape = AppShapes.large,
                            color = MaterialTheme.colorScheme.surface
                        ) {
                            Column(
                                modifier = Modifier.padding(18.dp),
                                verticalArrangement = Arrangement.spacedBy(6.dp),
                                horizontalAlignment = Alignment.CenterHorizontally
                            ) {
                                Icon(Icons.Filled.Search, contentDescription = null)
                                Text("没有匹配的聊天", fontWeight = FontWeight.SemiBold)
                                Text(
                                    "换一个玩家名、关键词或时间再试。",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                        }
                    }
                }
            }

            AnimatedVisibility(
                visible = showMentionSuggestions,
                enter = fadeIn(tween(AppMotion.Fast, easing = AppMotion.Easing)),
                exit = fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing))
            ) {
                MentionSuggestions(
                    players = mentionPlayers,
                    onSelect = { player ->
                        input = insertMention(input, player.gameId)
                        selectedMentions = (selectedMentions + MentionSelection(player.playerRef, player.gameId))
                            .distinctBy { it.playerRef }
                        showMentions = false
                    }
                )
            }

            AnimatedVisibility(
                visible = unseenMessages > 0,
                enter = fadeIn(tween(AppMotion.Fast, easing = AppMotion.Easing)),
                exit = fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing))
            ) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 4.dp),
                    contentAlignment = Alignment.Center
                ) {
                    AssistChip(
                        onClick = {
                            scope.launch {
                                if (!searchActive && visibleMessages.isNotEmpty()) {
                                    listState.animateScrollToItem(visibleMessages.lastIndex)
                                }
                                onFollowLatestChange(true)
                                onUnseenMessagesChange(0)
                            }
                        },
                        leadingIcon = {
                            Icon(Icons.Filled.Chat, contentDescription = null, modifier = Modifier.size(18.dp))
                        },
                        label = { Text("$unseenMessages 条新消息") }
                    )
                }
            }

            InlineMessage(
                message = message ?: repository.connectionMessage,
                modifier = Modifier.padding(horizontal = 16.dp),
                autoDismiss = message != null && autoDismissMessage,
                onDismiss = {
                    message = null
                    autoDismissMessage = false
                }
            )
            ChatInputBar(
                input = input,
                replyTarget = replyTarget,
                onInputChange = {
                    input = it
                    selectedMentions = selectedMentions.filter { mention ->
                        it.text.contains("@${mention.gameId}")
                    }
                    showMentions = activeMentionQuery(it) != null
                },
                onMentionClick = {
                    if (showMentions) {
                        showMentions = false
                    } else {
                        if (activeMentionQuery(input) == null) {
                            input = appendMentionMarker(input)
                        }
                        showMentions = true
                    }
                },
                onClearReply = { replyTarget = null },
                onSend = {
                    val outgoingContent = if (BuildConfig.CANARY_UI) {
                        composeReplyContent(replyTarget, input.text)
                    } else {
                        input.text
                    }
                    when (val result = repository.send(outgoingContent, selectedMentions.map { it.playerRef })) {
                        is RepoResult.Success -> {
                            input = TextFieldValue("")
                            replyTarget = null
                            selectedMentions = emptyList()
                            showMentions = false
                            message = null
                            autoDismissMessage = false
                            onFollowLatestChange(true)
                            onUnseenMessagesChange(0)
                            scope.launch {
                                if (!searchActive && displayedMessages.isNotEmpty()) {
                                    listState.scrollToItem(displayedMessages.lastIndex)
                                }
                            }
                        }
                        is RepoResult.Error -> {
                            message = result.message
                            autoDismissMessage = false
                        }
                    }
                }
            )
        }

        if (showPlayerDirectory) {
            PlayerDirectoryPanel(
                players = repository.playerDirectory,
                onDismiss = { showPlayerDirectory = false },
                onInspect = { player ->
                    selectedPlayer = player
                    showPlayerDirectory = false
                }
            )
        }
    }

    actionMessage?.let { selected ->
        ChatMessageActionsDialog(
            message = selected,
            onDismiss = { actionMessage = null },
            onReply = {
                if (!selected.event) {
                    replyTarget = selected
                    message = "正在回复 ${selected.sender}。"
                    autoDismissMessage = true
                }
                actionMessage = null
            },
            onCopy = {
                clipboardManager.setText(AnnotatedString(selected.content))
                message = if (selected.event) "已复制服务器事件。" else "已复制 ${selected.sender} 的消息。"
                autoDismissMessage = true
                actionMessage = null
            }
        )
    }

    selectedPlayer?.let { player ->
        AlertDialog(
            onDismissRequest = { selectedPlayer = null },
            title = { Text(player.gameId) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("QQ：${player.qq ?: "未绑定或不可见"}")
                    Text(playerDirectoryPrimaryLabel(player))
                    Text(playerDirectoryStatusLabel(player))
                }
            },
            confirmButton = {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                    OutlinedButton(
                        enabled = !player.self,
                        onClick = {
                            scope.launch {
                                when (val result = repository.toggleFollow(player)) {
                                    is RepoResult.Success -> {
                                        val followed = result.value
                                        selectedPlayer = player.copy(followed = followed)
                                        message = if (followed) "已关心 ${player.gameId}。" else "已取消关心 ${player.gameId}。"
                                        autoDismissMessage = true
                                    }
                                    is RepoResult.Error -> {
                                        message = result.message
                                        autoDismissMessage = false
                                    }
                                }
                            }
                        }
                    ) {
                        Text(if (player.followed) "取消关心" else "关心")
                    }
                    Button(
                        enabled = !player.self,
                        onClick = {
                            selectedPlayer = null
                            onTransferToPlayer(
                                ResolvedPlayerRef(
                                    playerRef = player.playerRef,
                                    gameId = player.gameId,
                                    qq = player.qq,
                                    online = player.serverOnline,
                                    registered = player.registered,
                                    source = "player_directory"
                                )
                            )
                        }
                    ) {
                        Text("向他转账")
                    }
                }
            },
            dismissButton = {
                TextButton(onClick = { selectedPlayer = null }) {
                    Text("关闭")
                }
            }
        )
    }
}

@Composable
private fun AiScreen(repository: AiRepository) {
    val scope = rememberCoroutineScope()
    val clipboardManager = LocalClipboardManager.current
    val listState = rememberLazyListState()
    val density = LocalDensity.current
    val keyboardPadding = with(density) {
        (
            WindowInsets.ime.getBottom(this) -
                WindowInsets.navigationBars.getBottom(this)
            )
            .coerceAtLeast(0)
            .toDp()
    }
    var input by remember { mutableStateOf("") }
    var showPlans by remember { mutableStateOf(false) }
    var localMessage by remember { mutableStateOf<String?>(null) }
    var forceScrollToken by remember { mutableStateOf(0) }
    val quota = repository.quota
    val planName = repository.currentPlan?.name?.takeIf { it.isNotBlank() } ?: "免费额度"
    val restoreTime = quota?.restoresAt ?: quota?.restoreAt ?: quota?.resetsAt ?: quota?.resetAt
    val assistantName = repository.assistantName

    LaunchedEffect(Unit) {
        repository.loadInitial()
    }

    val nearBottom by remember {
        derivedStateOf {
            val layout = listState.layoutInfo
            val total = layout.totalItemsCount
            if (total == 0) {
                true
            } else {
                (layout.visibleItemsInfo.lastOrNull()?.index ?: 0) >= total - 2
            }
        }
    }

    LaunchedEffect(repository.messages.size, repository.messages.lastOrNull()?.id) {
        val last = repository.messages.lastOrNull() ?: return@LaunchedEffect
        if (last.role == "user" || nearBottom) {
            listState.scrollToItem(repository.messages.lastIndex)
        }
    }

    LaunchedEffect(forceScrollToken, repository.messages.size) {
        if (forceScrollToken > 0 && repository.messages.isNotEmpty()) {
            listState.scrollToItem(repository.messages.lastIndex)
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(bottom = keyboardPadding)
    ) {
        AppSurface(
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = 64.dp)
                .padding(horizontal = 10.dp, vertical = 4.dp)
                .clickable(enabled = !repository.loading) {
                    scope.launch {
                        repository.loadPlans()
                        showPlans = true
                    }
                },
            shape = AppShapes.extraLarge
        ) {
            Row(
                modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Surface(
                    modifier = Modifier.size(42.dp),
                    shape = CircleShape,
                    color = MaterialTheme.colorScheme.surfaceVariant,
                    tonalElevation = 0.dp,
                    shadowElevation = 0.dp
                ) {
                    XiaoxiangAvatar(Modifier.fillMaxSize())
                }
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
                    Text(
                        assistantName,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                    Text(
                        text = aiQuotaLabel(planName, quota?.remaining, quota?.limit, restoreTime),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
                Text(
                    if (repository.loading) "加载中" else "升级",
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.primary,
                    maxLines = 1
                )
            }
        }

        LazyColumn(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth(),
            state = listState,
            contentPadding = PaddingValues(horizontal = 12.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            if (repository.messages.isEmpty()) {
                item(key = "ai-empty", contentType = "ai-empty") {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 8.dp, vertical = 18.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Surface(
                            modifier = Modifier.size(54.dp),
                            shape = CircleShape,
                            color = MaterialTheme.colorScheme.surfaceVariant,
                            tonalElevation = 0.dp,
                            shadowElevation = 0.dp
                        ) {
                            XiaoxiangAvatar(Modifier.fillMaxSize())
                        }
                        Text(assistantName, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
                        Text(
                            "发送 /new 开启新对话",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            }
            items(
                items = repository.messages,
                key = { it.id },
                contentType = { "ai-${it.role}" }
            ) { message ->
                AiMessageBubble(
                    message = message,
                    assistantName = assistantName,
                    onCopy = { text ->
                        clipboardManager.setText(AnnotatedString(text))
                        localMessage = "已复制 $assistantName 的消息。"
                    }
                )
            }
        }

        InlineMessage(
            message = localMessage ?: repository.message,
            modifier = Modifier.padding(horizontal = 12.dp),
            onDismiss = {
                if (localMessage != null) {
                    localMessage = null
                } else {
                    repository.clearMessage()
                }
            }
        )
        AiInputBar(
            input = input,
            sending = repository.sending || repository.loading,
            assistantName = assistantName,
            onInputChange = { input = it },
            onSend = {
                if (repository.sending || repository.loading) return@AiInputBar
                val outgoing = input
                val trimmed = outgoing.trim()
                val localError = validateAiMessage(trimmed)
                if (localError != null && trimmed != "/new") {
                    localMessage = localError
                    return@AiInputBar
                }
                input = ""
                forceScrollToken += 1
                scope.launch {
                    repository.sendMessage(outgoing)
                }
            }
        )
    }

    if (showPlans) {
        AiPlansDialog(
            plans = repository.plans,
            purchasing = repository.purchasing,
            onDismiss = { showPlans = false },
            onRefresh = { scope.launch { repository.loadPlans() } },
            onPurchase = { plan ->
                scope.launch { repository.purchase(plan) }
            }
        )
    }
}

@Composable
private fun AiInputBar(
    input: String,
    sending: Boolean,
    assistantName: String,
    onInputChange: (String) -> Unit,
    onSend: () -> Unit
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
        color = MaterialTheme.colorScheme.surface,
        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.55f)),
        tonalElevation = 0.dp,
        shadowElevation = 0.dp
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Surface(
                modifier = Modifier.weight(1f),
                shape = AppShapes.pill,
                color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.62f),
                contentColor = MaterialTheme.colorScheme.onSurface,
                tonalElevation = 0.dp,
                shadowElevation = 0.dp
            ) {
                BasicTextField(
                    value = input,
                    onValueChange = onInputChange,
                    enabled = true,
                    textStyle = MaterialTheme.typography.bodyLarge.copy(color = MaterialTheme.colorScheme.onSurface),
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = 44.dp, max = 120.dp)
                        .padding(horizontal = 16.dp, vertical = 12.dp),
                    maxLines = 5,
                    decorationBox = { innerTextField ->
                        Box(contentAlignment = Alignment.CenterStart) {
                            if (input.isBlank()) {
                                Text(
                                    "给 $assistantName 发消息",
                                    style = MaterialTheme.typography.bodyLarge,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                            innerTextField()
                        }
                    }
                )
            }
            val canSend = !sending && input.isNotBlank()
            Surface(
                modifier = Modifier
                    .size(44.dp)
                    .clickable(enabled = canSend) { onSend() },
                shape = CircleShape,
                color = if (canSend) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant,
                contentColor = if (canSend) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurfaceVariant,
                tonalElevation = 0.dp,
                shadowElevation = 0.dp
            ) {
                Box(contentAlignment = Alignment.Center) {
                    Icon(Icons.Filled.Send, contentDescription = if (sending) "发送中" else "发送", modifier = Modifier.size(20.dp))
                }
            }
        }
    }
}

@Composable
private fun AiMessageBubble(
    message: AiUiMessage,
    assistantName: String,
    onCopy: (String) -> Unit
) {
    val mine = message.role == "user"
    val content = message.content.ifBlank { if (message.streaming) "正在生成..." else "" }
    if (mine) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.End
        ) {
            Surface(
                modifier = Modifier.widthIn(max = 340.dp),
                color = MaterialTheme.colorScheme.primary,
                contentColor = MaterialTheme.colorScheme.onPrimary,
                shape = RoundedCornerShape(topStart = 22.dp, topEnd = 22.dp, bottomStart = 22.dp, bottomEnd = 7.dp),
                tonalElevation = 0.dp,
                shadowElevation = 0.dp
            ) {
                Text(
                    text = content,
                    modifier = Modifier.padding(horizontal = 15.dp, vertical = 10.dp),
                    style = MaterialTheme.typography.bodyLarge
                )
            }
        }
        return
    }
    val canCopy = !message.streaming && content.isNotBlank()
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.Top
    ) {
        Surface(
            modifier = Modifier.size(30.dp),
            shape = CircleShape,
            color = MaterialTheme.colorScheme.surfaceVariant,
            tonalElevation = 0.dp,
            shadowElevation = 0.dp
        ) {
            XiaoxiangAvatar(Modifier.fillMaxSize())
        }
        Column(
            modifier = Modifier
                .weight(1f)
                .combinedClickable(
                    enabled = canCopy,
                    onClick = {},
                    onLongClick = { onCopy(content) }
                ),
            verticalArrangement = Arrangement.spacedBy(5.dp)
        ) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    assistantName,
                    style = MaterialTheme.typography.labelLarge,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    if (message.streaming) message.statusText ?: "生成中" else message.time,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                if (canCopy) {
                    IconButton(
                        modifier = Modifier.size(30.dp),
                        onClick = { onCopy(content) }
                    ) {
                        Icon(Icons.Filled.ContentCopy, contentDescription = "复制 AI 消息", modifier = Modifier.size(15.dp))
                    }
                }
            }
            if (message.renderMarkdown) {
                AiMarkdownText(content)
            } else {
                Text(
                    text = content,
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurface
                )
            }
            if (message.sources.isNotEmpty()) {
                Text(
                    text = "参考：" + message.sources.take(3).joinToString(" / ") { source ->
                        source.title?.takeIf { it.isNotBlank() } ?: "知识库"
                    },
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
            }
        }
    }
}

@Composable
private fun AiMarkdownText(content: String, modifier: Modifier = Modifier) {
    val blocks = remember(content) { parseAiMarkdown(content) }
    val bodyColor = MaterialTheme.colorScheme.onSurface
    val variantColor = MaterialTheme.colorScheme.onSurfaceVariant
    val codeBackground = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.70f)
    val codeColor = MaterialTheme.colorScheme.onSurface
    Column(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        blocks.forEach { block ->
            when (block) {
                is AiMarkdownBlock.Heading -> Text(
                    text = markdownInline(block.text, codeBackground, codeColor),
                    style = if (block.level <= 1) MaterialTheme.typography.titleMedium else MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = bodyColor
                )
                is AiMarkdownBlock.Paragraph -> Text(
                    text = markdownInline(block.text, codeBackground, codeColor),
                    style = MaterialTheme.typography.bodyLarge,
                    color = bodyColor
                )
                is AiMarkdownBlock.Bullet -> Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("•", style = MaterialTheme.typography.bodyLarge, color = variantColor)
                    Text(
                        text = markdownInline(block.text, codeBackground, codeColor),
                        style = MaterialTheme.typography.bodyLarge,
                        color = bodyColor,
                        modifier = Modifier.weight(1f)
                    )
                }
                is AiMarkdownBlock.Quote -> Row(horizontalArrangement = Arrangement.spacedBy(9.dp)) {
                    Box(
                        modifier = Modifier
                            .width(3.dp)
                            .heightIn(min = 24.dp)
                            .clip(AppShapes.pill)
                            .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.55f))
                    )
                    Text(
                        text = markdownInline(block.text, codeBackground, codeColor),
                        style = MaterialTheme.typography.bodyMedium,
                        color = variantColor,
                        modifier = Modifier.weight(1f)
                    )
                }
                is AiMarkdownBlock.Code -> Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(12.dp),
                    color = codeBackground,
                    tonalElevation = 0.dp,
                    shadowElevation = 0.dp
                ) {
                    Text(
                        text = block.text,
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace,
                        color = codeColor,
                        modifier = Modifier.padding(10.dp)
                    )
                }
            }
        }
    }
}

private sealed interface AiMarkdownBlock {
    data class Heading(val level: Int, val text: String) : AiMarkdownBlock
    data class Paragraph(val text: String) : AiMarkdownBlock
    data class Bullet(val text: String) : AiMarkdownBlock
    data class Quote(val text: String) : AiMarkdownBlock
    data class Code(val text: String) : AiMarkdownBlock
}

private fun parseAiMarkdown(content: String): List<AiMarkdownBlock> {
    if (content.isBlank()) return listOf(AiMarkdownBlock.Paragraph(""))
    val blocks = mutableListOf<AiMarkdownBlock>()
    val paragraph = mutableListOf<String>()
    val code = StringBuilder()
    var inCode = false

    fun flushParagraph() {
        if (paragraph.isEmpty()) return
        blocks += AiMarkdownBlock.Paragraph(paragraph.joinToString("\n").trim())
        paragraph.clear()
    }

    content.lines().forEach { raw ->
        val line = raw.trimEnd()
        val trimmed = line.trim()
        if (trimmed.startsWith("```")) {
            if (inCode) {
                blocks += AiMarkdownBlock.Code(code.toString().trimEnd())
                code.clear()
                inCode = false
            } else {
                flushParagraph()
                inCode = true
            }
            return@forEach
        }
        if (inCode) {
            code.appendLine(raw)
            return@forEach
        }
        if (trimmed.isBlank()) {
            flushParagraph()
            return@forEach
        }
        val heading = Regex("""^(#{1,3})\s+(.+)$""").matchEntire(trimmed)
        if (heading != null) {
            flushParagraph()
            blocks += AiMarkdownBlock.Heading(heading.groupValues[1].length, heading.groupValues[2].trim())
            return@forEach
        }
        val bullet = Regex("""^[-*+]\s+(.+)$""").matchEntire(trimmed)
        if (bullet != null) {
            flushParagraph()
            blocks += AiMarkdownBlock.Bullet(bullet.groupValues[1].trim())
            return@forEach
        }
        if (trimmed.startsWith(">")) {
            flushParagraph()
            blocks += AiMarkdownBlock.Quote(trimmed.removePrefix(">").trim())
            return@forEach
        }
        paragraph += line
    }
    if (inCode && code.isNotBlank()) {
        blocks += AiMarkdownBlock.Code(code.toString().trimEnd())
    }
    flushParagraph()
    return blocks.ifEmpty { listOf(AiMarkdownBlock.Paragraph(content)) }
}

private fun markdownInline(text: String, codeBackground: Color, codeColor: Color): AnnotatedString =
    buildAnnotatedString {
        var index = 0
        while (index < text.length) {
            when {
                text.startsWith("`", index) -> {
                    val end = text.indexOf('`', startIndex = index + 1)
                    if (end > index + 1) {
                        withStyle(SpanStyle(fontFamily = FontFamily.Monospace, background = codeBackground, color = codeColor)) {
                            append(text.substring(index + 1, end))
                        }
                        index = end + 1
                    } else {
                        append(text[index])
                        index += 1
                    }
                }
                text.startsWith("**", index) -> {
                    val end = text.indexOf("**", startIndex = index + 2)
                    if (end > index + 2) {
                        withStyle(SpanStyle(fontWeight = FontWeight.Bold)) {
                            append(text.substring(index + 2, end))
                        }
                        index = end + 2
                    } else {
                        append(text[index])
                        index += 1
                    }
                }
                else -> {
                    append(text[index])
                    index += 1
                }
            }
        }
    }

@Composable
private fun AiPlansDialog(
    plans: List<AiPlan>,
    purchasing: Boolean,
    onDismiss: () -> Unit,
    onRefresh: () -> Unit,
    onPurchase: (AiPlan) -> Unit
) {
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 14.dp, vertical = 18.dp),
            contentAlignment = Alignment.BottomCenter
        ) {
            Surface(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(24.dp),
                color = MaterialTheme.colorScheme.surface,
                tonalElevation = 0.dp,
                shadowElevation = 0.dp,
                border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.55f))
            ) {
                Column(
                    modifier = Modifier
                        .heightIn(max = 560.dp)
                        .verticalScroll(rememberScrollState())
                        .padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("AI 套餐", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold, modifier = Modifier.weight(1f))
                        IconButton(onClick = onRefresh) {
                            Icon(Icons.Filled.Refresh, contentDescription = "刷新套餐")
                        }
                        IconButton(onClick = onDismiss) {
                            Icon(Icons.Filled.Close, contentDescription = "关闭")
                        }
                    }
                    if (plans.isEmpty()) {
                        Text(
                            "暂无可购买套餐。",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    plans.forEach { plan ->
                        AppSurface(modifier = Modifier.fillMaxWidth(), shape = AppShapes.medium) {
                            Column(
                                modifier = Modifier.padding(14.dp),
                                verticalArrangement = Arrangement.spacedBy(8.dp)
                            ) {
                                Text(plan.name.ifBlank { "AI 套餐" }, fontWeight = FontWeight.SemiBold)
                                plan.description?.takeIf { it.isNotBlank() }?.let {
                                    Text(
                                        it,
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant
                                    )
                                }
                                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                                    FilterChip(selected = false, onClick = {}, label = { Text(plan.priceLabel()) })
                                    FilterChip(selected = false, onClick = {}, label = { Text(plan.durationLabel()) })
                                    FilterChip(selected = false, onClick = {}, label = { Text(plan.quotaLabel()) })
                                }
                                Button(
                                    enabled = !purchasing,
                                    onClick = { onPurchase(plan) },
                                    modifier = Modifier.align(Alignment.End)
                                ) {
                                    Text(if (purchasing) "处理中" else "购买")
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

private fun aiQuotaLabel(planName: String, remaining: Int?, limit: Int?, restoreTime: String?): String {
    val quotaText = if (remaining != null && limit != null && limit > 0) {
        "$remaining/$limit"
    } else {
        "额度加载中"
    }
    val restoreText = restoreTime?.takeIf { it.isNotBlank() }?.let {
        "，恢复 ${formatIsoDateTimeUtc8(it)}"
    }.orEmpty()
    return "$planName · 剩余额度 $quotaText$restoreText"
}

@Composable
private fun ChatMessageActionsDialog(
    message: ChatFeedItem,
    onDismiss: () -> Unit,
    onReply: () -> Unit,
    onCopy: () -> Unit
) {
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 14.dp, vertical = 18.dp),
            contentAlignment = Alignment.BottomCenter
        ) {
            Surface(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(24.dp),
                color = MaterialTheme.colorScheme.surface,
                tonalElevation = 0.dp,
                shadowElevation = 0.dp,
                border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.55f))
            ) {
                Column(
                    modifier = Modifier.padding(14.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    Text(
                        if (message.event) "服务器事件" else message.sender,
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.primary,
                        fontWeight = FontWeight.SemiBold
                    )
                    Text(
                        compactChatSnippet(message.content, 120),
                        maxLines = 3,
                        overflow = TextOverflow.Ellipsis,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        ChatActionButton(
                            label = "回复",
                            icon = Icons.Filled.Reply,
                            enabled = !message.event,
                            modifier = Modifier.weight(1f),
                            onClick = onReply
                        )
                        ChatActionButton(
                            label = "复制",
                            icon = Icons.Filled.ContentCopy,
                            modifier = Modifier.weight(1f),
                            onClick = onCopy
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ChatActionButton(
    label: String,
    icon: ImageVector,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    onClick: () -> Unit
) {
    OutlinedButton(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier.height(44.dp)
    ) {
        Icon(icon, contentDescription = null, modifier = Modifier.size(18.dp))
        Spacer(Modifier.width(7.dp))
        Text(label)
    }
}

@Composable
private fun ReplyComposerPreview(
    replyTarget: ChatFeedItem,
    onClear: () -> Unit
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = AppShapes.large,
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.72f),
        tonalElevation = 0.dp,
        shadowElevation = 0.dp
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp)
        ) {
            Box(
                modifier = Modifier
                    .width(3.dp)
                    .height(34.dp)
                    .clip(AppShapes.pill)
                    .background(MaterialTheme.colorScheme.primary)
            )
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(
                    "回复 ${replyTarget.sender}",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.primary,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    compactChatSnippet(replyTarget.content, 84),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }
            IconButton(onClick = onClear, modifier = Modifier.size(34.dp)) {
                Icon(Icons.Filled.Close, contentDescription = "取消回复", modifier = Modifier.size(18.dp))
            }
        }
    }
}

@Composable
private fun PlayerDirectoryPanel(
    players: List<PlayerDirectoryItem>,
    onDismiss: () -> Unit,
    onInspect: (PlayerDirectoryItem) -> Unit
) {
    val serverOnlineCount by remember(players) {
        derivedStateOf { players.count { it.serverOnline } }
    }

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        Surface(
            modifier = Modifier.fillMaxSize(),
            color = MaterialTheme.colorScheme.background
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 18.dp, vertical = 16.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp)
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Text("玩家列表", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                        Text(
                            "$serverOnlineCount 位服务器在线，${players.size} 位可查看玩家",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    IconButton(onClick = onDismiss) {
                        Icon(Icons.Filled.Close, contentDescription = "关闭玩家列表")
                    }
                }

                LazyColumn(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                    contentPadding = PaddingValues(bottom = 12.dp)
                ) {
                    items(
                        items = players,
                        key = { it.playerRef },
                        contentType = { "player-directory-item" }
                    ) { player ->
                        AppSurface(
                            modifier = Modifier
                                .fillMaxWidth()
                                .animateItem(),
                            shape = AppShapes.list,
                            color = MaterialTheme.colorScheme.surface
                        ) {
                            Row(
                                modifier = Modifier.padding(horizontal = 14.dp, vertical = 12.dp),
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.spacedBy(12.dp)
                            ) {
                                Box(
                                    modifier = Modifier
                                        .size(10.dp)
                                        .clip(CircleShape)
                                        .background(playerDirectoryDotColor(player))
                                )
                                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                        Text(
                                            player.gameId,
                                            style = MaterialTheme.typography.titleMedium,
                                            fontWeight = FontWeight.SemiBold
                                        )
                                        if (player.self) {
                                            Text(
                                                "我",
                                                style = MaterialTheme.typography.labelSmall,
                                                color = MaterialTheme.colorScheme.primary
                                            )
                                        }
                                        if (player.followed) {
                                            Text(
                                                "已关心",
                                                style = MaterialTheme.typography.labelSmall,
                                                color = MaterialTheme.colorScheme.tertiary
                                            )
                                        }
                                    }
                                    Text(
                                        playerDirectoryPrimaryLabel(player),
                                        maxLines = 1,
                                        overflow = TextOverflow.Ellipsis,
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant
                                    )
                                    Text(
                                        playerDirectoryStatusLabel(player),
                                        maxLines = 1,
                                        overflow = TextOverflow.Ellipsis,
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant
                                    )
                                }
                                TextButton(onClick = { onInspect(player) }) {
                                    Text("查看")
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

private fun playerDirectoryPrimaryLabel(player: PlayerDirectoryItem): String =
    when {
        player.serverOnline && player.registered -> "服务器在线，已注册 App"
        player.serverOnline -> "服务器在线玩家"
        player.registered -> "已注册 App"
        else -> "玩家"
    }

private fun playerDirectoryStatusLabel(player: PlayerDirectoryItem): String =
    when {
        player.serverOnline -> "当前服务器在线"
        player.appForeground -> "App 前台在线"
        player.appConnected -> "App 后台连接在线"
        player.appStatus == "online" -> "App 前台在线"
        player.appStatus == "just_online" -> "App 5 分钟内在线"
        player.appStatus == "recent_online" -> "App 30 分钟内在线"
        player.appStatus == "recently_online" -> "App 1 天内在线"
        player.appLastSeenAt != null -> "App 离线较久"
        else -> "暂无 App 在线记录"
    }

private fun playerDirectoryDotColor(player: PlayerDirectoryItem): Color =
    when {
        player.serverOnline -> Color(0xFF2E7D32)
        player.appForeground || player.appConnected -> Color(0xFF1565C0)
        player.appStatus == "just_online" || player.appStatus == "recent_online" -> Color(0xFFF9A825)
        else -> Color(0xFF9E9E9E)
    }

@Composable
private fun PlayerAvatar(
    gameId: String,
    statusColor: Color,
    modifier: Modifier = Modifier.size(38.dp)
) {
    Box(modifier = modifier, contentAlignment = Alignment.BottomEnd) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.primaryContainer),
            contentAlignment = Alignment.Center
        ) {
            Text(
                playerInitial(gameId),
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onPrimaryContainer,
                fontWeight = FontWeight.SemiBold
            )
        }
        Box(
            modifier = Modifier
                .size(10.dp)
                .clip(CircleShape)
                .background(statusColor)
        )
    }
}

@Composable
private fun MentionSuggestions(
    players: List<PlayerDirectoryItem>,
    onSelect: (PlayerDirectoryItem) -> Unit
) {
    AppSurface(
        modifier = Modifier
            .padding(
                horizontal = if (BuildConfig.CANARY_UI) 10.dp else 16.dp,
                vertical = if (BuildConfig.CANARY_UI) 4.dp else 6.dp
            )
            .fillMaxWidth(),
        shape = if (BuildConfig.CANARY_UI) {
            RoundedCornerShape(topStart = 22.dp, topEnd = 22.dp, bottomStart = 14.dp, bottomEnd = 14.dp)
        } else {
            AppShapes.list
        }
    ) {
        Column(
            Modifier.padding(if (BuildConfig.CANARY_UI) 10.dp else 12.dp),
            verticalArrangement = Arrangement.spacedBy(if (BuildConfig.CANARY_UI) 6.dp else 8.dp)
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    if (BuildConfig.CANARY_UI) "提及玩家" else "选择要提及的玩家",
                    modifier = Modifier.weight(1f),
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Text(
                    "${players.size}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            LazyColumn(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(max = if (BuildConfig.CANARY_UI) 284.dp else 260.dp),
                verticalArrangement = Arrangement.spacedBy(if (BuildConfig.CANARY_UI) 4.dp else 6.dp)
            ) {
                items(
                    items = players,
                    key = { it.playerRef },
                    contentType = { "mention-player" }
                ) { player ->
                    AppSurface(
                        modifier = Modifier
                            .fillMaxWidth()
                            .animateItem(),
                        shape = if (BuildConfig.CANARY_UI) AppShapes.large else AppShapes.medium,
                        color = if (BuildConfig.CANARY_UI) {
                            MaterialTheme.colorScheme.surface
                        } else {
                            MaterialTheme.colorScheme.surfaceVariant
                        }
                    ) {
                        TextButton(
                            onClick = { onSelect(player) },
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.spacedBy(10.dp)
                            ) {
                                if (BuildConfig.CANARY_UI) {
                                    PlayerAvatar(player.gameId, playerDirectoryDotColor(player), Modifier.size(38.dp))
                                } else {
                                    Box(
                                        modifier = Modifier
                                            .size(9.dp)
                                            .clip(CircleShape)
                                            .background(playerDirectoryDotColor(player))
                                    )
                                }
                                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(1.dp)) {
                                    Text(
                                        "@${player.gameId}",
                                        style = MaterialTheme.typography.bodyMedium,
                                        fontWeight = FontWeight.SemiBold,
                                        color = MaterialTheme.colorScheme.onSurface
                                    )
                                    Text(
                                        when {
                                            player.followed -> "已关心"
                                            player.serverOnline -> "服务器在线玩家"
                                            player.appForeground -> "App 前台在线"
                                            player.appConnected -> "App 后台连接在线"
                                            else -> "可提及玩家"
                                        },
                                        maxLines = 1,
                                        overflow = TextOverflow.Ellipsis,
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant
                                    )
                                }
                                if (BuildConfig.CANARY_UI) {
                                    Text(
                                        when {
                                            player.followed -> "★"
                                            player.serverOnline -> "在线"
                                            player.appConnected -> "App"
                                            else -> ""
                                        },
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.primary,
                                        maxLines = 1
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ChatInputBar(
    input: TextFieldValue,
    replyTarget: ChatFeedItem?,
    onInputChange: (TextFieldValue) -> Unit,
    onMentionClick: () -> Unit,
    onClearReply: () -> Unit,
    onSend: () -> Unit
) {
    val focusRequester = remember { FocusRequester() }
    val keyboardController = LocalSoftwareKeyboardController.current
    val controlSize = if (BuildConfig.CANARY_UI) 44.dp else 46.dp
    val inputMaxHeight = if (BuildConfig.CANARY_UI) 96.dp else 108.dp
    val inputShape = if (BuildConfig.CANARY_UI) AppShapes.large else AppShapes.extraLarge
    fun focusInput() {
        focusRequester.requestFocus()
        keyboardController?.show()
    }

    AppSurface(
        modifier = Modifier.fillMaxWidth(),
        shape = if (BuildConfig.CANARY_UI) {
            RoundedCornerShape(topStart = 20.dp, topEnd = 20.dp)
        } else {
            RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp)
        }
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(
                    horizontal = if (BuildConfig.CANARY_UI) 10.dp else 12.dp,
                    vertical = if (BuildConfig.CANARY_UI) 6.dp else 7.dp
                ),
            verticalArrangement = Arrangement.spacedBy(if (BuildConfig.CANARY_UI) 6.dp else 7.dp)
        ) {
            if (BuildConfig.CANARY_UI) {
                AnimatedVisibility(
                    visible = replyTarget != null,
                    enter = fadeIn(tween(AppMotion.CanaryFast, easing = AppMotion.Easing)),
                    exit = fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing))
                ) {
                    if (replyTarget != null) {
                        ReplyComposerPreview(
                            replyTarget = replyTarget,
                            onClear = onClearReply
                        )
                    }
                }
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(if (BuildConfig.CANARY_UI) 8.dp else 10.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                AppIconButtonSurface(
                    modifier = Modifier.size(controlSize),
                    onClick = onMentionClick
                ) {
                    Icon(
                        Icons.Filled.AlternateEmail,
                        contentDescription = "提及玩家",
                        modifier = Modifier.size(24.dp)
                    )
                }
                AppSurface(
                    modifier = Modifier
                        .weight(1f)
                        .heightIn(min = controlSize, max = inputMaxHeight)
                        .animateContentSize(
                            animationSpec = tween(
                                if (BuildConfig.CANARY_UI) AppMotion.CanaryFast else AppMotion.Fast,
                                easing = AppMotion.Easing
                            )
                        ),
                    shape = inputShape,
                    color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.78f)
                ) {
                    BasicTextField(
                        value = input,
                        onValueChange = onInputChange,
                        maxLines = 4,
                        textStyle = MaterialTheme.typography.bodyLarge.copy(
                            color = MaterialTheme.colorScheme.onSurface
                        ),
                        modifier = Modifier
                            .fillMaxWidth()
                            .focusRequester(focusRequester)
                            .padding(horizontal = 14.dp, vertical = 11.dp)
                    ) { innerTextField ->
                        Box(
                            contentAlignment = Alignment.CenterStart,
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            if (input.text.isBlank()) {
                                Text(
                                    if (BuildConfig.CANARY_UI && replyTarget != null) "输入回复" else "公共聊天",
                                    maxLines = 1,
                                    softWrap = false,
                                    overflow = TextOverflow.Clip,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                            innerTextField()
                        }
                    }
                }
                AppIconButtonSurface(
                    modifier = Modifier.size(controlSize),
                    selected = true,
                    onClick = onSend
                ) {
                    Icon(
                        Icons.Filled.Send,
                        contentDescription = "发送",
                        modifier = Modifier.size(24.dp)
                    )
                }
            }
        }
    }
}

@Composable
private fun ProfileScreen(
    user: UserProfile,
    repository: AuthRepository,
    chatRepository: ChatRepository,
    appUpdateRepository: AppUpdateRepository,
    appearance: AppAppearance,
    onThemeModeChange: (DeuteriumThemeMode) -> Unit,
    onColorPresetChange: (DeuteriumColorPreset) -> Unit,
    onDynamicColorChange: (Boolean) -> Unit,
    showServerEvents: Boolean,
    onShowServerEventsChange: (Boolean) -> Unit,
    walletNotifications: Boolean,
    onWalletNotificationsChange: (Boolean) -> Unit,
    onLogout: () -> Unit
) {
    var showPasswordReset by remember { mutableStateOf(false) }
    var showChatHistory by remember { mutableStateOf(false) }
    var updateMessage by remember { mutableStateOf<String?>(null) }
    var updateMessageAutoDismiss by remember { mutableStateOf(false) }
    var checkingUpdate by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(18.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp)
    ) {
        Text("我的资料", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
        AppSurface(modifier = Modifier.fillMaxWidth(), shape = AppShapes.large) {
            Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text(user.gameId, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                InfoRow("QQ", user.qq)
                InfoRow("玩家身份", "已绑定服务器身份")
            }
        }
        AppSurface(modifier = Modifier.fillMaxWidth(), shape = AppShapes.large) {
            Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                        Text("账号安全", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                        Text(
                            "修改密码会重新发送游戏内验证码。",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    TextButton(onClick = { showPasswordReset = !showPasswordReset }) {
                        Text(if (showPasswordReset) "收起" else "修改密码")
                    }
                }
                AnimatedVisibility(
                    visible = showPasswordReset,
                    enter = fadeIn(tween(AppMotion.Fast, easing = AppMotion.Easing)) +
                        expandVertically(tween(AppMotion.Fast, easing = AppMotion.Easing)),
                    exit = fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing)) +
                        shrinkVertically(tween(AppMotion.Fast, easing = AppMotion.Easing))
                ) {
                    ResetPasswordForm(
                        repository = repository,
                        initialAccount = user.gameId,
                        accountReadOnly = true
                    ) {
                        showPasswordReset = false
                        onLogout()
                    }
                }
            }
        }
        AppearanceSettingsSection(
            appearance = appearance,
            onThemeModeChange = onThemeModeChange,
            onColorPresetChange = onColorPresetChange,
            onDynamicColorChange = onDynamicColorChange
        )
        ChatSettingsSection(
            showServerEvents = showServerEvents,
            onShowServerEventsChange = onShowServerEventsChange,
            walletNotifications = walletNotifications,
            onWalletNotificationsChange = onWalletNotificationsChange
        )
        ChatHistoryEntry(onClick = { showChatHistory = true })
        UpdateCheckEntry(
            loading = checkingUpdate,
            message = updateMessage,
            autoDismiss = updateMessageAutoDismiss,
            onDismissMessage = {
                updateMessage = null
                updateMessageAutoDismiss = false
            },
            onClick = {
                if (!checkingUpdate) {
                    checkingUpdate = true
                    updateMessage = null
                    updateMessageAutoDismiss = false
                    scope.launch {
                        when (val result = appUpdateRepository.checkUpdate()) {
                            is RepoResult.Success -> {
                                updateMessage = result.value.message
                                updateMessageAutoDismiss = true
                            }
                            is RepoResult.Error -> {
                                updateMessage = result.message
                                updateMessageAutoDismiss = false
                            }
                        }
                        checkingUpdate = false
                    }
                }
            }
        )
        OutlinedButton(onClick = onLogout, modifier = Modifier.fillMaxWidth()) {
            Icon(Icons.Filled.ExitToApp, contentDescription = null)
            Spacer(Modifier.width(8.dp))
            Text("退出登录")
        }
    }

    if (showChatHistory) {
        ChatHistoryDialog(
            repository = chatRepository,
            onDismiss = { showChatHistory = false }
        )
    }
}

@Composable
private fun UpdateCheckEntry(
    loading: Boolean,
    message: String?,
    autoDismiss: Boolean,
    onDismissMessage: () -> Unit,
    onClick: () -> Unit
) {
    AppSurface(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(enabled = !loading, onClick = onClick),
        shape = AppShapes.large
    ) {
        Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                Icon(Icons.Filled.Refresh, contentDescription = null)
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Text("检查更新", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                    Text(
                        "当前版本 ${BuildConfig.VERSION_NAME} (${BuildConfig.VERSION_CODE})",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                OutlinedButton(
                    onClick = onClick,
                    enabled = !loading
                ) {
                    Text(if (loading) "检查中" else "检查")
                }
            }
            InlineMessage(
                message = message,
                autoDismiss = autoDismiss,
                onDismiss = onDismissMessage
            )
        }
    }
}

@Composable
private fun ChatHistoryEntry(onClick: () -> Unit) {
    AppSurface(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
        shape = AppShapes.large
    ) {
        Row(
            modifier = Modifier.padding(18.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Icon(Icons.Filled.Chat, contentDescription = null)
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text("聊天记录", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                Text(
                    "查找本机保存的聊天历史，或删除当前账号的本地聊天记录。",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

@Composable
private fun ChatHistoryDialog(
    repository: ChatRepository,
    onDismiss: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var query by rememberSaveable { mutableStateOf("") }
    var history by remember { mutableStateOf<List<ChatHistoryItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var message by remember { mutableStateOf<String?>(null) }
    var confirmDelete by remember { mutableStateOf(false) }

    LaunchedEffect(query) {
        loading = true
        if (query.isNotBlank()) delay(250)
        history = repository.searchHistory(query)
        loading = false
    }

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        Surface(
            modifier = Modifier.fillMaxSize(),
            color = MaterialTheme.colorScheme.background
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .windowInsetsPadding(WindowInsets.statusBars)
                    .padding(horizontal = 18.dp, vertical = 16.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp)
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                        Text("聊天记录", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
                        Text(
                            "仅保存并搜索当前账号在本机的聊天历史",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    IconButton(onClick = onDismiss) {
                        Icon(Icons.Filled.Close, contentDescription = "关闭聊天记录")
                    }
                }

                OutlinedTextField(
                    value = query,
                    onValueChange = { query = it },
                    label = { Text("搜索玩家或内容") },
                    leadingIcon = { Icon(Icons.Filled.Search, contentDescription = null) },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth()
                )

                InlineMessage(message)

                when {
                    loading -> {
                        AppSurface(modifier = Modifier.fillMaxWidth(), shape = AppShapes.medium) {
                            Text(
                                "正在读取本地聊天记录...",
                                modifier = Modifier.padding(14.dp),
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    }
                    history.isEmpty() -> {
                        AppSurface(modifier = Modifier.fillMaxWidth(), shape = AppShapes.medium) {
                            Text(
                                if (query.isBlank()) "本机还没有保存聊天记录。" else "没有找到匹配的聊天记录。",
                                modifier = Modifier.padding(14.dp),
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    }
                    else -> {
                        LazyColumn(
                            modifier = Modifier
                                .fillMaxWidth()
                                .weight(1f),
                            verticalArrangement = Arrangement.spacedBy(8.dp),
                            contentPadding = PaddingValues(bottom = 12.dp)
                        ) {
                            items(
                                items = history,
                                key = { it.messageId },
                                contentType = { if (it.event) "history-event" else "history-message" }
                            ) { item ->
                                ChatHistoryRow(item, Modifier.animateItem())
                            }
                        }
                    }
                }

                OutlinedButton(
                    onClick = { confirmDelete = true },
                    enabled = history.isNotEmpty() || query.isNotBlank(),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Icon(Icons.Filled.Delete, contentDescription = null)
                    Spacer(Modifier.width(8.dp))
                    Text("删除当前账号的所有聊天记录")
                }
            }
        }
    }

    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text("删除聊天记录") },
            text = { Text("这只会删除当前账号保存在本机的聊天记录，不会影响后端聊天消息。") },
            confirmButton = {
                Button(
                    onClick = {
                        scope.launch {
                            repository.deleteHistory()
                            history = emptyList()
                            message = "已删除当前账号的本地聊天记录。"
                            confirmDelete = false
                        }
                    }
                ) {
                    Text("删除")
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmDelete = false }) {
                    Text("取消")
                }
            }
        )
    }
}

@Composable
private fun ChatHistoryRow(item: ChatHistoryItem, modifier: Modifier = Modifier) {
    AppSurface(
        modifier = modifier.fillMaxWidth(),
        shape = AppShapes.list,
        color = if (item.event) MaterialTheme.colorScheme.tertiaryContainer else MaterialTheme.colorScheme.surface
    ) {
        Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    if (item.event) "服务器事件" else item.sender,
                    modifier = Modifier.weight(1f),
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold
                )
                Text(
                    item.displayTime.ifBlank { item.sentAt },
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            Text(item.content, style = MaterialTheme.typography.bodyMedium)
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun AppearanceSettingsSection(
    appearance: AppAppearance,
    onThemeModeChange: (DeuteriumThemeMode) -> Unit,
    onColorPresetChange: (DeuteriumColorPreset) -> Unit,
    onDynamicColorChange: (Boolean) -> Unit
) {
    val dynamicAvailable = Build.VERSION.SDK_INT >= Build.VERSION_CODES.S
    AppSurface(modifier = Modifier.fillMaxWidth(), shape = AppShapes.large) {
        Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text("应用外观", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                Text(
                    "外观设置会保存在当前设备。",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            Text("主题模式", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                DeuteriumThemeMode.entries.forEach { mode ->
                    FilterChip(
                        selected = appearance.themeMode == mode,
                        onClick = { onThemeModeChange(mode) },
                        label = { Text(mode.label) }
                    )
                }
            }

            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text("莫奈动态取色", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold)
                    Text(
                        if (dynamicAvailable) "使用系统壁纸色生成 Material 3 配色。" else "需要 Android 12 或更高版本。",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                Switch(
                    checked = appearance.dynamicColor && dynamicAvailable,
                    enabled = dynamicAvailable,
                    onCheckedChange = onDynamicColorChange
                )
            }

            Text("应用颜色", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                DeuteriumColorPreset.entries.forEach { preset ->
                    FilterChip(
                        selected = appearance.colorPreset == preset && !appearance.dynamicColor,
                        enabled = !appearance.dynamicColor,
                        onClick = { onColorPresetChange(preset) },
                        label = { Text(preset.label) },
                        leadingIcon = {
                            Box(
                                modifier = Modifier
                                    .size(12.dp)
                                    .clip(CircleShape)
                                    .background(preset.swatch)
                            )
                        }
                    )
                }
            }

        }
    }
}

@Composable
private fun ChatSettingsSection(
    showServerEvents: Boolean,
    onShowServerEventsChange: (Boolean) -> Unit,
    walletNotifications: Boolean,
    onWalletNotificationsChange: (Boolean) -> Unit
) {
    AppSurface(modifier = Modifier.fillMaxWidth(), shape = AppShapes.large) {
        Column(Modifier.padding(18.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
            SettingSwitchRow(
                title = "服务器事件",
                description = "显示服务器事件，例如玩家进出、签到和世界保存提示。",
                checked = showServerEvents,
                onCheckedChange = onShowServerEventsChange
            )
            Divider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.45f))
            SettingSwitchRow(
                title = "钱包变动通知",
                description = "App 在后台时，成功收入或支出会通过系统通知提醒。",
                checked = walletNotifications,
                onCheckedChange = onWalletNotificationsChange
            )
        }
    }
}

@Composable
private fun SettingSwitchRow(
    title: String,
    description: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
            Text(
                description,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        Switch(
            checked = checked,
            onCheckedChange = onCheckedChange
        )
    }
}
@Composable
private fun WalletRecordRow(record: WalletRecord, modifier: Modifier = Modifier) {
    val sign = if (record.direction == "income") "+" else "-"
    AppSurface(
        modifier = modifier.fillMaxWidth(),
        shape = AppShapes.list
    ) {
        Row(
            modifier = Modifier.padding(14.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(Modifier.weight(1f)) {
                Text(record.otherPlayer.gameId, fontWeight = FontWeight.SemiBold)
                Text(
                    text = listOfNotNull(formatIsoDateTimeUtc8(record.occurredAt), record.status, record.note).joinToString(" · "),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            Text(
                text = "$sign${record.amount}",
                fontWeight = FontWeight.Bold,
                color = if (record.direction == "income") Color(0xFF2E7D32) else MaterialTheme.colorScheme.error
            )
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun PlayerSummaryCard(player: PlayerSummary) {
    AppSurface(
        color = MaterialTheme.colorScheme.secondaryContainer,
        contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
        shape = AppShapes.medium
    ) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(player.gameId, fontWeight = FontWeight.SemiBold)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(selected = player.online, onClick = {}, label = { Text(if (player.online) "在线" else "离线") })
                player.qq?.let { FilterChip(selected = false, onClick = {}, label = { Text("QQ $it") }) }
            }
            Text(if (player.registered) "已绑定 App 账号" else "服务器可确认玩家", style = MaterialTheme.typography.bodySmall)
        }
    }
}

@Composable
private fun ChatBubble(
    message: ChatFeedItem,
    modifier: Modifier = Modifier,
    onLongPress: () -> Unit
) {
    if (message.event) {
        Box(modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
            val eventColor = if (BuildConfig.CANARY_UI) {
                MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.92f)
            } else {
                MaterialTheme.colorScheme.tertiaryContainer
            }
            val eventContentColor = if (BuildConfig.CANARY_UI) {
                MaterialTheme.colorScheme.onSurfaceVariant
            } else {
                MaterialTheme.colorScheme.onTertiaryContainer
            }
            Surface(
                color = eventColor,
                contentColor = eventContentColor,
                shape = AppShapes.pill,
                tonalElevation = 0.dp,
                shadowElevation = 0.dp,
                modifier = Modifier.combinedClickable(
                    onClick = {},
                    onLongClickLabel = if (BuildConfig.CANARY_UI) "操作" else "复制",
                    onLongClick = onLongPress
                )
            ) {
                Text(
                    text = message.content,
                    modifier = Modifier.padding(
                        horizontal = if (BuildConfig.CANARY_UI) 12.dp else 14.dp,
                        vertical = if (BuildConfig.CANARY_UI) 6.dp else 7.dp
                    ),
                    style = if (BuildConfig.CANARY_UI) {
                        MaterialTheme.typography.labelMedium
                    } else {
                        MaterialTheme.typography.bodySmall
                    },
                    color = eventContentColor
                )
            }
        }
        return
    }

    val bubbleShape = if (BuildConfig.CANARY_UI) {
        if (message.mine) {
            RoundedCornerShape(topStart = 22.dp, topEnd = 22.dp, bottomStart = 22.dp, bottomEnd = 7.dp)
        } else {
            RoundedCornerShape(topStart = 22.dp, topEnd = 22.dp, bottomStart = 7.dp, bottomEnd = 22.dp)
        }
    } else {
        RoundedCornerShape(22.dp)
    }
    val bubbleColor = if (message.mine) {
        MaterialTheme.colorScheme.primaryContainer
    } else if (BuildConfig.CANARY_UI) {
        MaterialTheme.colorScheme.surface
    } else {
        MaterialTheme.colorScheme.surfaceVariant
    }
    val contentColor = if (message.mine) {
        MaterialTheme.colorScheme.onPrimaryContainer
    } else {
        MaterialTheme.colorScheme.onSurfaceVariant
    }

    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = if (message.mine) Arrangement.End else Arrangement.Start
    ) {
        Surface(
            color = bubbleColor,
            contentColor = contentColor,
            shape = bubbleShape,
            tonalElevation = 0.dp,
            shadowElevation = 0.dp,
            modifier = Modifier
                .fillMaxWidth(if (BuildConfig.CANARY_UI) 0.86f else 0.82f)
                .widthIn(max = if (BuildConfig.CANARY_UI) 380.dp else 340.dp)
                .combinedClickable(
                    onClick = {},
                    onLongClickLabel = if (BuildConfig.CANARY_UI) "操作" else "复制",
                    onLongClick = onLongPress
                )
        ) {
            Column(
                Modifier.padding(
                    horizontal = if (BuildConfig.CANARY_UI) 13.dp else 14.dp,
                    vertical = if (BuildConfig.CANARY_UI) 9.dp else 10.dp
                ),
                verticalArrangement = Arrangement.spacedBy(if (BuildConfig.CANARY_UI) 4.dp else 5.dp)
            ) {
                Row {
                    Text(
                        message.sender,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier.weight(1f),
                        color = contentColor
                    )
                    Text(
                        message.time,
                        style = MaterialTheme.typography.labelSmall,
                        color = contentColor.copy(alpha = 0.72f)
                    )
                }
                Text(
                    text = message.content,
                    style = MaterialTheme.typography.bodyLarge,
                    color = contentColor
                )
                if (message.mine && message.deliveryState != ChatDeliveryState.Confirmed) {
                    Text(
                        text = if (message.deliveryState == ChatDeliveryState.Pending) "发送中…" else "发送结果待确认",
                        style = MaterialTheme.typography.labelSmall,
                        color = contentColor.copy(alpha = 0.72f)
                    )
                }
            }
        }
    }
}

@Composable
private fun RowScope.BottomItem(label: String, icon: ImageVector, selected: Boolean, onClick: () -> Unit) {
    val iconScale by animateFloatAsState(
        targetValue = if (selected) {
            if (BuildConfig.CANARY_UI) 1.08f else 1.14f
        } else {
            1f
        },
        animationSpec = spring(
            dampingRatio = if (BuildConfig.CANARY_UI) {
                Spring.DampingRatioNoBouncy
            } else {
                Spring.DampingRatioMediumBouncy
            },
            stiffness = Spring.StiffnessMedium
        ),
        label = "BottomItemIconScale"
    )
    val contentAlpha by animateFloatAsState(
        targetValue = if (selected) 1f else 0.74f,
        animationSpec = tween(AppMotion.Fast, easing = AppMotion.Easing),
        label = "BottomItemAlpha"
    )
    NavigationBarItem(
        selected = selected,
        onClick = onClick,
        icon = {
            Icon(
                icon,
                contentDescription = label,
                modifier = Modifier.graphicsLayer {
                    scaleX = iconScale
                    scaleY = iconScale
                    alpha = contentAlpha
                }
            )
        },
        label = { Text(label) }
    )
}

@Composable
private fun PrimaryActionButton(text: String, icon: ImageVector, onClick: () -> Unit) {
    val interactionSource = remember { MutableInteractionSource() }
    val pressed by interactionSource.collectIsPressedAsState()
    val scale by animateFloatAsState(
        targetValue = if (pressed) {
            if (BuildConfig.CANARY_UI) AppMotion.CanaryPressedScale else AppMotion.PressedScale
        } else {
            1f
        },
        animationSpec = spring(
            dampingRatio = if (BuildConfig.CANARY_UI) {
                Spring.DampingRatioNoBouncy
            } else {
                Spring.DampingRatioMediumBouncy
            },
            stiffness = Spring.StiffnessHigh
        ),
        label = "PrimaryActionPressScale"
    )
    Button(
        onClick = onClick,
        modifier = Modifier
            .fillMaxWidth()
            .graphicsLayer {
                scaleX = scale
                scaleY = scale
            },
        shape = AppShapes.pill,
        interactionSource = interactionSource
    ) {
        Icon(icon, contentDescription = null)
        Spacer(Modifier.width(8.dp))
        Text(text)
    }
}

@Composable
private fun InlineMessage(
    message: String?,
    modifier: Modifier = Modifier,
    autoDismiss: Boolean = false,
    onDismiss: () -> Unit = {}
) {
    LaunchedEffect(message, autoDismiss) {
        if (message != null && autoDismiss) {
            delay(3_000)
            onDismiss()
        }
    }
    AnimatedVisibility(
        visible = message != null,
        enter = if (BuildConfig.CANARY_UI) {
            fadeIn(tween(AppMotion.CanaryFast, easing = AppMotion.Easing))
        } else {
            fadeIn(tween(AppMotion.Fast, easing = AppMotion.Easing)) +
                expandVertically(tween(AppMotion.Fast, easing = AppMotion.Easing))
        },
        exit = if (BuildConfig.CANARY_UI) {
            fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing))
        } else {
            fadeOut(tween(AppMotion.Micro, easing = AppMotion.Easing)) +
                shrinkVertically(tween(AppMotion.Fast, easing = AppMotion.Easing))
        },
        modifier = modifier.fillMaxWidth()
    ) {
        if (message != null) {
            AppSurface(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = AppShapes.small,
                modifier = Modifier.fillMaxWidth()
            ) {
                Text(
                    text = message,
                    modifier = Modifier.padding(10.dp),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

@Composable
private fun InfoRow(label: String, value: String) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(label, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, fontWeight = FontWeight.SemiBold)
    }
}

private fun activeMentionQuery(input: TextFieldValue): String? {
    val cursor = input.selection.end.coerceIn(0, input.text.length)
    val beforeCursor = input.text.substring(0, cursor)
    val atIndex = beforeCursor.lastIndexOf('@')
    if (atIndex == -1) return null
    val query = beforeCursor.substring(atIndex + 1)
    return if (query.any { it.isWhitespace() }) null else query
}

private fun appendMentionMarker(input: TextFieldValue): TextFieldValue {
    val start = minOf(input.selection.start, input.selection.end).coerceIn(0, input.text.length)
    val end = maxOf(input.selection.start, input.selection.end).coerceIn(0, input.text.length)
    val before = input.text.substring(0, start)
    val after = input.text.substring(end)
    val prefix = when {
        before.isBlank() -> ""
        before.endsWith(" ") -> before
        else -> "$before "
    }
    val text = "${prefix}@${after}"
    val cursor = prefix.length + 1
    return TextFieldValue(text, TextRange(cursor))
}

private fun insertMention(input: TextFieldValue, gameId: String): TextFieldValue {
    val query = activeMentionQuery(input)
    val selectionStart = minOf(input.selection.start, input.selection.end).coerceIn(0, input.text.length)
    val selectionEnd = maxOf(input.selection.start, input.selection.end).coerceIn(0, input.text.length)
    val beforeSelection = input.text.substring(0, selectionStart)
    val afterSelection = input.text.substring(selectionEnd)
    val atIndex = beforeSelection.lastIndexOf('@')
    val mention = "@$gameId "
    val text: String
    val cursor: Int
    if (query != null && atIndex != -1) {
        val prefix = beforeSelection.substring(0, atIndex)
        text = prefix + mention + afterSelection
        cursor = prefix.length + mention.length
    } else if (beforeSelection.isBlank()) {
        text = mention + afterSelection
        cursor = mention.length
    } else {
        val prefix = if (beforeSelection.endsWith(" ")) beforeSelection else "$beforeSelection "
        text = prefix + mention + afterSelection
        cursor = prefix.length + mention.length
    }
    return TextFieldValue(text, TextRange(cursor))
}
