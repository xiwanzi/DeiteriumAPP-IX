package com.deuterium.app.uilab

import androidx.compose.runtime.*
import androidx.compose.runtime.snapshots.SnapshotStateList
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.input.TextFieldValue
import kotlinx.coroutines.*
import org.json.JSONArray
import org.json.JSONObject
import java.net.URLEncoder
import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale
import java.util.UUID

data class LedgerEntry(val id: Long, val name: String, val detail: String, val amount: Long, val time: String, val at: LocalDateTime = LocalDateTime.now())
data class ChatLine(val id: Long, val name: String, val text: String, val mine: Boolean, val time: String = "刚刚", val reply: ChatReply? = null, val remoteId: String = "", val serverAt: Long = 0, val forwarded:ChatReply?=null,val aiStatus:String?=null,val sources:List<AiSource> = emptyList(),val searchUsed:Boolean=false)
data class Announcement(val id: String, val title: String, val content: String, val publishedAt: String)
fun credit(cents: Long): String = String.format(Locale.US, "%,.2f", cents / 100.0)
fun apiCents(value: String): Long = value.toBigDecimal().movePointRight(2).longValueExact()
private fun JSONArray.objects(): List<JSONObject> = (0 until length()).mapNotNull { optJSONObject(it) }

class LabState(private val scope: CoroutineScope, initialFollowed: Set<String> = emptySet(),
    private val saveFollowed: (Set<String>) -> Unit = {}, val userName: String = "", private val postNotification: (DemoNotice) -> Unit = {}, val api: BackendApi? = null,private val notificationPreferences:NotificationPreferences?=null,private val launcherIcons:LauncherIcons?=null) {
    var restoring by mutableStateOf(false)
    var storageMessage by mutableStateOf<String?>(null)
    var profileBio by mutableStateOf("")
    var balance by mutableLongStateOf(0)
    var balanceKnown by mutableStateOf(false)
    var walletError by mutableStateOf<String?>(null)
    var ledgerKnown by mutableStateOf(false);private set
    var ledgerError by mutableStateOf<String?>(null);private set
    var todayIncome by mutableStateOf<Long?>(null);private set
    var todayExpense by mutableStateOf<Long?>(null);private set
    var remoteHeld by mutableLongStateOf(0)
    var refreshed by mutableStateOf("等待同步")
    var refreshing by mutableStateOf(false)
    private var refreshAgain=false
    var transferPending by mutableStateOf(api?.pendingTransfer()!=null);private set
    var transferSucceeded by mutableStateOf(false);private set
    var activeTransferKey by mutableStateOf<String?>(null);private set
    var completedTransferKey by mutableStateOf<String?>(null);private set
    private var transferRecovering=false
    var chatReplyTo by mutableStateOf<ChatReply?>(null)
    var chatDraft by mutableStateOf(TextFieldValue(""))
    var chatReplyPending by mutableStateOf(false)
    var chatStatus by mutableStateOf("正在连接…")
    var onlineCount by mutableStateOf<Int?>(null);private set
    var loadingChatHistory by mutableStateOf(false);private set
    var publicHistoryCursor by mutableStateOf<String?>(null);private set
    private var publicHistoryLoaded=false
    var privacy by mutableStateOf(false)
    val ledger = mutableStateListOf<LedgerEntry>()
    val chat = mutableStateListOf<ChatLine>()
    val followed = mutableStateListOf<String>().apply { addAll(initialFollowed) }
    val recentTransfers = mutableStateListOf<String>()
    var notice by mutableStateOf<DemoNotice?>(null)
    val cart = mutableStateMapOf<String, Int>()
    val commerce = CommerceBook(userName, emptyList(), { balance }, { _, _, _ -> }, remoteOnly = true)
    val commissions = CommissionBook(userName, { balance }, { _, _, _ -> }, remoteOnly = true)
    val heldBalance: Long get() = remoteHeld
    val directChats = mutableStateMapOf<String, SnapshotStateList<ChatLine>>()
    val directPending = mutableStateMapOf<String, Boolean>()
    val conversationUnread = mutableStateMapOf<String, Int>()
    var activeConversation: String? = null
    var foreground = true
    val hasUnreadMessages: Boolean get() = conversationUnread.values.any{it>0}
    private val conversationIds=mutableMapOf<String,String>()
    private val contactRefreshLock=kotlinx.coroutines.sync.Mutex()
    var contactsKnown by mutableStateOf(api==null);private set
    var contactsRefreshing by mutableStateOf(false);private set
    var contactsError by mutableStateOf<String?>(null);private set
    private val readCursors=mutableMapOf<String,String>()
    private val directHistoryCursors=mutableMapOf<String,String?>()
    val directHistoryLoading=mutableStateMapOf<String,Boolean>()
    val directErrors=mutableStateMapOf<String,String>()
    val events = emptyList<ServerEvent>()
    val joinedEvents = mutableStateListOf<String>()
    var joinedEvent by mutableStateOf(false)
    val announcements = mutableStateListOf<Announcement>()
    var announcementError by mutableStateOf<String?>(null)
    private var serial = 10000L
    val nextEventSequence: Long get() = serial
    private var chatSocket: BackendChat? = null
    private var reconnect: Job? = null
    private var closed = false
    private var preferencesSyncedAt=0L
    private var uncertainMessage: Triple<String, String, String?>? = null
    val interventions=api?.let{BackendInterventions(it,this)}
    val ai=api?.let{BackendAI(it,this)}
    init { commerce.interventions=interventions;commissions.interventions=interventions;commerce.network=api?.let{BackendCatalog(it,this)};commissions.network=api?.let{BackendCommissions(it,this)} }
    fun restoreLedger(records: List<LedgerEntry>, sequence: Long) { ledger.clear(); ledger.addAll(records); serial = sequence.coerceAtLeast(10000) }

    fun connect() {
        val service = api ?: return
        scope.launch {
            runCatching { service.verifySession() }.onSuccess { profileBio = service.user?.optString("bio").orEmpty() }
                .onFailure { report(it) }
            if(!service.signedIn || closed) return@launch
            openChat()
            scope.launch{notificationPreferences?.sync();preferencesSyncedAt=System.nanoTime()/1_000_000}
            service.pendingChat()?.let { previous ->
                uncertainMessage = Triple(previous.getString("clientId"), previous.getString("content"), previous.optString("reply").takeIf { it.isNotBlank() })
                chatDraft = TextFieldValue(previous.getString("content"))
                storageMessage = "上一条消息结果待确认，重试会使用原消息标识"
            }
            refresh()
            scope.launch{loadAnnouncements()}
            scope.launch { while(!closed){if(foreground)loadConversations();delay(4000)} }
            scope.launch { loadDirectory();refreshPresence() }
            scope.launch { commerce.network?.refreshStore() }
            scope.launch { commerce.network?.refreshMarket() }
            scope.launch { commissions.network?.refresh();commissions.network?.recover() }
            scope.launch { commerce.network?.refreshOrders();commerce.network?.recoverOrder("STORE_PURCHASE");commerce.network?.recoverOrder("MARKET_PURCHASE");ai?.recoverPurchase() }
            scope.launch { while(!closed){syncNotices();delay(8000)} }
            if(service.pendingTransfer() != null) scope.launch { recoverTransfer() }
            scope.launch { while(!closed){delay(2500);if(!service.pendingTransfer()?.optString("transferId").isNullOrBlank())recoverTransfer()} }
        }
    }
    private fun openChat() {
        val service = api ?: return
        val token = service.token ?: return
        chatSocket?.close()
        chatSocket = BackendChat(service.http, service.baseUrl, token, { message -> scope.launch {
            if(message.optString("kind")=="private_chat") {
                val id=message.optString("conversationId")
                if(conversationIds.values.none{it==id})loadConversations()
                conversationIds.entries.find{it.value==id}?.key?.let{addMessage(message,false,it)}
            }else if(message.optString("kind")=="public_chat")addMessage(message,true)
        } },
            { connected, error -> scope.launch {
                chatStatus = if(connected) "已连接服务器" else error ?: "连接暂不可用"
                if(!connected && !closed) {
                    reconnect?.cancel(); reconnect = scope.launch { delay(4000); runCatching { service.verifySession() }; if(service.signedIn && !closed) openChat() }
                }
            } }, { scope.launch { loadChat() };launcherIcons?.requestSync() },launcherIcons?.let{icons->{icons.requestSync()}}).also { it.connect() }
    }
    private suspend fun loadChat() {
        runCatching { api!!.request("GET", "/chat/messages?limit=100") }
            .onSuccess { response->response.getJSONArray("messages").objects().asReversed().forEach { value -> addMessage(value, false) };if(!publicHistoryLoaded){publicHistoryCursor=cursor(response);publicHistoryLoaded=true} }.onFailure { chatStatus = it.message ?: "消息读取失败" }
    }
    private fun cursor(response:JSONObject)=response.optJSONObject("_page")?.optString("nextCursor")?.takeUnless{it.isBlank()||it=="null"}
    suspend fun loadOlderPublic(){
        val before=publicHistoryCursor ?: return;if(loadingChatHistory)return
        loadingChatHistory=true
        runCatching{api!!.request("GET","/chat/messages?limit=100&before=${URLEncoder.encode(before,"UTF-8")}")}.onSuccess{response->response.getJSONArray("messages").objects().asReversed().forEach{addMessage(it,false)};publicHistoryCursor=cursor(response)}.onFailure{report(it)}
        loadingChatHistory=false
    }
    private fun addMessage(value: JSONObject, notify: Boolean, thread:String?=null) {
        val target=if(thread==null)chat else conversation(thread)
        val id = value.optString("messageId"); if(id.isBlank() || target.any { it.remoteId == id }) return
        val sender = value.optJSONObject("sender") ?: return
        val name = sender.optString("gameId"); val mine = sender.optString("playerRef") == api?.playerRef
        if(name.isNotBlank() && name != userName && Players.none { it.name == name }) Players.add(PlayerProfile(name, online = sender.optBoolean("online"), playerRef = sender.optString("playerRef")))
        val sentAt=runCatching{Instant.parse(value.optString("sentAt"))}.getOrDefault(Instant.now())
        val at=sentAt.atZone(ZoneId.systemDefault()).toLocalDateTime()
        val reply = value.optJSONObject("reply")?.let { ChatReply(it.optString("messageId").hashCode().toLong(), it.optJSONObject("sender")?.optString("gameId").orEmpty(), if(it.optString("availability")=="UNAVAILABLE")"原消息不可见" else it.optString("content"), it.optString("messageId")) }
        val forwarded=value.optJSONObject("forwarded")?.let{ChatReply(it.optString("messageId").hashCode().toLong(),it.optJSONObject("sender")?.optString("gameId").orEmpty(),it.optString("content"),it.optString("messageId"))}
        target.add(ChatLine(++serial, if(mine) "你" else name, value.optString("content"), mine, at.format(DateTimeFormatter.ofPattern("HH:mm")), reply, id, sentAt.toEpochMilli(),forwarded))
        target.sortBy { it.serverAt }
        val mentions=value.optJSONArray("mentionedPlayerRefs")
        val mentioned=mentions!=null&&(0 until mentions.length()).any{mentions.optString(it)==api?.playerRef}
        if(notify && !mine && (name in followed || mentioned)) {
            val event = DemoNotice(++serial, if(name in followed) "特别关心 · $name" else "$name 提及了你", value.optString("content"))
            // The server inbox owns system notifications; this is the in-app visual hint only.
            notice = event
        }
    }
    fun sendChat() {
        val text = chatDraft.text.trim().take(256); if(text.isBlank() || chatReplyPending) return
        val replyId = chatReplyTo?.remoteId?.takeIf { it.isNotBlank() } ?: uncertainMessage?.takeIf{it.second==text}?.third
        val previous = uncertainMessage
        if(previous != null && (previous.second != text || previous.third != replyId)) { storageMessage = "上一条消息结果待确认，请先重试原消息"; return }
        val clientId = previous?.first ?: UUID.randomUUID().toString()
        uncertainMessage = Triple(clientId, text, replyId)
        val refs = api?.pendingChat()?.takeIf{it.optString("clientId")==clientId}?.optJSONArray("mentionedPlayerRefs")
            ?: JSONArray(Regex("@([A-Za-z0-9_]+)").findAll(text).mapNotNull { match -> Players.find { it.name == match.groupValues[1] }?.playerRef?.takeIf { it.isNotBlank() } }.toList())
        api?.savePendingChat(JSONObject().put("clientId", clientId).put("content", text).put("reply", replyId.orEmpty()).put("mentionedPlayerRefs",refs))
        chatReplyPending = true
        scope.launch {
            runCatching {
                val request=JSONObject().put("clientMessageId",clientId).put("content",text).put("mentionedPlayerRefs",refs)
                replyId?.let{request.put("replyToMessageId",it)}
                api?.request("POST","/chat/messages",request) ?: throw ApiFailure("CHAT_DISCONNECTED", "消息连接暂不可用")
            }.onSuccess { response ->
                when(response.optString("status")) {
                    "accepted" -> { response.optJSONObject("message")?.let{addMessage(it,false)};if(chatDraft.text.trim()==text)chatDraft = TextFieldValue(""); chatReplyTo = null; uncertainMessage = null; api?.savePendingChat(null);scope.launch{loadChat()} }
                    "unknown", "" -> { storageMessage = "消息发送结果待确认，请重试原消息" }
                    else -> { if(response.optJSONObject("error")?.optString("code") != "RESULT_UNKNOWN"){uncertainMessage = null;api?.savePendingChat(null)}; storageMessage = response.optJSONObject("error")?.optString("message") ?: "消息发送失败" }
                }
            }.onFailure { if(it is ApiFailure && it.code == "CHAT_DISCONNECTED"){uncertainMessage = null;api?.savePendingChat(null)}; report(it) }
            chatReplyPending = false
        }
    }
    fun refresh() {
        if(api == null) return
        if(refreshing){refreshAgain=true;return}
        refreshing=true
        scope.launch {
          try { do {
            refreshAgain=false
            runCatching { api.request("GET", "/wallet/balance").getJSONObject("balance") }.onSuccess { value ->
                balance = apiCents(value.optString("availableAmount", value.getString("amount"))); remoteHeld = apiCents(value.optString("heldAmount", "0"))
                balanceKnown = true; walletError = null; refreshed = if(value.optBoolean("fresh")) "刚刚更新 · 已同步" else "已读取缓存余额"
                todayIncome=value.optJSONObject("today")?.optString("income")?.takeIf{it.isNotBlank()}?.let(::apiCents)
                todayExpense=value.optJSONObject("today")?.optString("expense")?.takeIf{it.isNotBlank()}?.let(::apiCents)
            }.onFailure { todayIncome=null;todayExpense=null;walletError = it.message ?: "余额读取失败"; refreshed = "余额尚未同步" }
            runCatching { val values=api.request("GET","/wallet/records?limit=25").getJSONArray("records").objects();values to ledgerEntries(values) }.onSuccess { (values,records) ->
                ledger.clear(); ledger.addAll(records)
                ledgerKnown=true;ledgerError=null
                recentTransfers.clear();recentTransfers.addAll(values.filter{it.optString("direction")=="expense"&&it.optString("businessType","TRANSFER")=="TRANSFER"}.mapNotNull{it.optJSONObject("otherPlayer")?.optString("gameId")?.takeUnless{name->name.isBlank()||name=="null"}}.distinct().take(5))
            }.onFailure { ledgerKnown=false;ledgerError="账单暂时无法完整同步，请稍后重试";walletError=walletError ?: ledgerError }
          } while(refreshAgain) } finally { refreshing=false }
        }
    }
    suspend fun searchRecipients(query: String): List<PlayerProfile> {
        val service = api ?: throw ApiFailure("UNAVAILABLE", "账号服务暂不可用")
        val values = service.request("GET", "/wallet/recipients/search?query=${URLEncoder.encode(query.trim(), "UTF-8")}&type=auto").optJSONArray("candidates")?.objects().orEmpty()
        return values.map { value -> PlayerProfile(value.getString("gameId"), value.optString("qq", ""), value.optString("bio", ""), value.optBoolean("online"), value.optString("lastSeen", "暂无记录"), value.getString("playerRef"),Players.find{it.playerRef==value.getString("playerRef")}?.avatar) }.filter { it.name != userName }.also { results ->
            results.forEach { player -> Players.removeAll { it.name == player.name }; Players.add(player) }
        }
    }
    suspend fun transfer(name: String, cents: Long, note: String,requestId:String?=null,confirmedRecipientRef:String?=null): Boolean {
        val service = api ?: return false
        val requestScope=service.financialScope()
        val recipientRef=confirmedRecipientRef?.takeIf{it.isNotBlank()} ?: Players.find { it.name == name && it.playerRef.isNotBlank() }?.playerRef ?: return false
        val request = JSONObject().put("clientRequestId", requestId ?: UUID.randomUUID().toString()).put("recipientPlayerRef", recipientRef)
            .put("amount", java.math.BigDecimal(cents).movePointLeft(2).toPlainString()).put("note", note)
        val existing = service.pendingTransfer()
        if(existing != null) {
            val previous=existing.optJSONObject("request")
            val sameRequest=requestId!=null&&previous!=null&&listOf("clientRequestId","recipientPlayerRef","amount","note").all{previous.optString(it)==request.optString(it)}
            activeTransferKey=if(sameRequest)requestId else null
            storageMessage="转账结果待确认，正在核对，请勿重复付款"
            val recovered=recoverTransfer()
            if(sameRequest)return recovered&&completedTransferKey==requestId&&transferSucceeded
            if(recovered)storageMessage="上一笔转账已确认，请重新确认本次转账"
            return false
        }
        activeTransferKey=request.getString("clientRequestId")
        transferSucceeded=false;transferPending=true
        service.saveTransfer(JSONObject().put("request", request),requestScope)
        return submitTransfer(request,requestScope)
    }
    suspend fun recoverTransfer(): Boolean {
        if(transferRecovering)return false
        val pending = api?.pendingTransfer() ?: return false
        transferRecovering=true
        val requestScope=api!!.financialScope()
        val transferId = pending.optString("transferId")
        return runCatching {
            requestScope.verify(pending)
            if(transferId.isNotBlank())handleTransfer(api!!.request("GET", "/wallet/transfers/$transferId").getJSONObject("transfer"),pending.getJSONObject("request"),requestScope)
            else submitTransfer(pending.getJSONObject("request"),requestScope)
        }.getOrElse { report(it); false }.also { transferRecovering=false }
    }
    private suspend fun submitTransfer(request: JSONObject,requestScope:FinancialScope): Boolean = runCatching {
        requestScope.verifyCurrent(api!!.financialScope())
        handleTransfer(api!!.request("POST", "/wallet/transfers", request).getJSONObject("transfer"), request,requestScope)
    }.getOrElse {
        if(it is ApiFailure && it.status in listOf(400, 401, 403, 404, 409, 422, 501)) {
            api?.saveTransfer(null,requestScope)
            if(api?.financialScope()==requestScope)transferPending=false
        }
        report(it); false
    }
    private fun handleTransfer(value: JSONObject, request: JSONObject,requestScope:FinancialScope): Boolean {
        val pending=if(value.optString("status") in setOf("success","failed"))null else JSONObject().put("request",request).put("transferId",value.optString("transferId"))
        api?.saveTransfer(pending,requestScope)
        requestScope.verifyCurrent(api!!.financialScope())
        transferPending=pending!=null;transferSucceeded=value.optString("status")=="success"
        completedTransferKey=if(transferSucceeded)request.getString("clientRequestId") else null
        when(value.optString("status")) {
            "success" -> { refresh(); storageMessage = null; return true }
            "failed" -> { storageMessage = value.optJSONObject("error")?.optString("message") ?: "转账未完成" }
            else -> { storageMessage = "转账结果待确认，请勿重复付款；重新打开应用会继续核对" }
        }
        return false
    }
    fun loadAnnouncements() { scope.launch {
        runCatching { api!!.listAll("/announcements") }.onSuccess { values ->
            announcements.clear(); announcements.addAll(values.map { value -> Announcement(value.getString("announcementId"), value.getString("title"), value.optJSONArray("contentBlocks")?.objects()?.joinToString("\n\n") { it.optString("text") }?.takeIf { it.isNotBlank() } ?: value.optString("summary"), value.optString("publishedAt")) }); announcementError = null
        }.onFailure { announcementError = it.message ?: "公告读取失败" }
    } }
    fun toggleFollow(name: String) {
        val player = Players.find { it.name==name && it.playerRef.isNotBlank() } ?: return
        val remove = name in followed
        scope.launch { runCatching { if(remove)api!!.request("DELETE", "/chat/follows/${player.playerRef}") else api!!.request("POST", "/chat/follows", JSONObject().put("playerRef",player.playerRef)) }
            .onSuccess { value -> if(value.getBoolean("followed")){if(name !in followed)followed.add(name)}else followed.remove(name);saveFollowed(followed.toSet()) }.onFailure { report(it) } }
    }
    private suspend fun syncNotices(){
        val service=api ?: return
        if(System.nanoTime()/1_000_000-preferencesSyncedAt>60_000){notificationPreferences?.sync();refreshPresence();loadDirectory();preferencesSyncedAt=System.nanoTime()/1_000_000}
        runCatching{service.request("GET","/notifications?limit=100").getJSONArray("items").objects()}.onSuccess { values->
            val shown=service.shownNotices();var delivered=0
            val all=values.map{value->
                val target=value.getJSONObject("target");val ref=target.optString("referenceId");val kind=target.getString("kind")
                val route=when(kind){"WALLET"->"wallet";"PUBLIC_CHAT"->"public";"CONVERSATION"->conversationIds.entries.find{it.value==ref}?.key?.let{"dm:$it"} ?: "Info";"COMMISSION"->"commission:$ref";"ORDER"->"order:$ref";"REFUND"->"refund:$ref";"ANNOUNCEMENT"->"announcement";"APP_UPDATE"->"about";else->"Info"}
                val id=value.getString("notificationId");val at=Instant.parse(value.getString("createdAt")).atZone(ZoneId.systemDefault()).toLocalDateTime()
                val record=TradeNotice(id,userName,value.getString("title"),value.getString("body"),ref,kind,at,!value.isNull("readAt"),route)
                if(id !in shown && !record.read && value.optBoolean("systemPush",true) && delivered<5){
                    val topic=when(value.getString("topic")){"DIRECT_MESSAGES"->NotificationTopic.Direct;"MENTIONS"->NotificationTopic.Mentions;"FOLLOWED_PLAYERS"->NotificationTopic.Followed;"WALLET"->NotificationTopic.Wallet;"COMMISSIONS"->NotificationTopic.Commissions;"ANNOUNCEMENTS"->NotificationTopic.Announcements;"APP_UPDATES"->NotificationTopic.Updates;else->NotificationTopic.Market}
                    postNotification(DemoNotice(id.hashCode().toLong(),record.title,record.body,route,topic))
                    delivered++
                }
                record
            }
            commerce.notices.clear();commerce.notices.addAll(all.filter{it.kind in setOf("ORDER","REFUND","COMMISSION","WALLET")})
            service.saveShownNotices(values.map{it.getString("notificationId")}.toSet())
        }
    }
    fun readNotice(id:String){scope.launch{runCatching{api!!.request("POST","/notifications/read",JSONObject().put("clientRequestId",UUID.randomUUID().toString()).put("notificationIds",JSONArray(listOf(id))))}.onSuccess{commerce.markNoticeRead(id)}.onFailure{report(it)}}}
    suspend fun saveBio(value:String):Boolean = runCatching {
        val service=api ?: return false
        val original=service.request("GET", "/players/${service.playerRef}")
        val profile=service.request("PATCH", "/account/me/profile", JSONObject().put("clientRequestId",UUID.randomUUID().toString()).put("expectedVersion",original.getLong("version")).put("bio",value.trim()))
        profileBio=profile.getString("bio");true
    }.getOrElse { report(it);false }
    fun mentionPlayer(name: String) {
        val value = chatDraft; val range = mentionRange(value); val start = range?.first ?: value.selection.min; val end = range?.last?.plus(1) ?: value.selection.max
        val prefix = if(range == null && start > 0 && !value.text[start-1].isWhitespace()) " " else ""
        val mention = "$prefix@$name "; val updated = value.text.replaceRange(start, end, mention).take(256)
        chatDraft = TextFieldValue(updated, TextRange((start+mention.length).coerceAtMost(updated.length)))
    }
    fun conversation(name: String): SnapshotStateList<ChatLine> = directChats.getOrPut(name) { mutableStateListOf() }
    fun pendingDirectDraft(name:String):String = if(name=="AI 助手")ai?.recoveredDraft.orEmpty() else Players.find{it.name==name}?.let { api?.pendingDirect(it.playerRef)?.optString("content") }.orEmpty()
    private fun rememberPlayer(value:JSONObject) {
        val name=value.optString("gameId");if(name.isBlank()||name==userName)return
        val player=PlayerProfile(name,value.optString("qq").takeUnless{it=="null"}.orEmpty(),value.optString("bio"),value.optBoolean("online",value.optBoolean("serverOnline")),value.optString("lastSeenAt").takeUnless{it.isBlank()||it=="null"} ?: "暂无记录",value.optString("playerRef"),if(value.has("avatar"))value.optJSONObject("avatar")?.optString("assetId")?.takeIf{it.isNotBlank()}?.let{"asset:$it"} else Players.find{it.name==name}?.avatar)
        Players.removeAll{it.name==name};Players.add(player)
    }
    private suspend fun loadDirectory() {
        runCatching { api!!.request("GET","/chat/player-directory").optJSONArray("players")?.objects().orEmpty() }.onSuccess { values->values.forEach(::rememberPlayer) }
        runCatching { api!!.request("GET","/chat/follows").optJSONArray("players")?.objects().orEmpty() }.onSuccess { values->followed.clear();values.forEach{rememberPlayer(it);followed.add(it.getString("gameId"))};saveFollowed(followed.toSet()) }
    }
    suspend fun loadProfile(name:String) {
        val ref=if(name==userName)api?.playerRef else Players.find{it.name==name}?.playerRef
        if(ref.isNullOrBlank())return
        runCatching { api!!.request("GET","/players/$ref") }.onSuccess { if(name==userName)profileBio=it.optString("bio") else rememberPlayer(it) }
    }
    private suspend fun loadConversations() {
        if(closed)return
        val service=api ?: return
        contactRefreshLock.lock()
        try{
            if(closed)return
            contactsRefreshing=true
            val account=service.financialScope()
            val values=service.listAll("/chat/conversations")
            account.verifyCurrent(service.financialScope())
            values.forEach{value->val other=value.getJSONObject("otherPlayer");rememberPlayer(other);val name=other.getString("gameId");conversationIds[name]=value.getString("conversationId");value.optJSONObject("lastMessage")?.let{addMessage(it,false,name)};conversationUnread[name]=if(value.optJSONObject("lastMessage")?.optString("messageId")==readCursors[conversationIds[name]])0 else value.optInt("unreadCount").coerceAtLeast(0)}
            contactsKnown=true;contactsError=null
        }catch(cancelled:kotlinx.coroutines.CancellationException){throw cancelled}
        catch(error:Exception){contactsError=error.message ?: "联系人暂时无法同步"}
        finally{contactsRefreshing=false;contactRefreshLock.unlock()}
    }
    suspend fun refreshContacts()=loadConversations()
    suspend fun searchDirectory(query:String):List<PlayerProfile>{
        if(closed)return emptyList()
        val service=api ?: return searchPlayers(query)
        val account=service.financialScope()
        val response=service.request("GET","/chat/player-directory?query=${URLEncoder.encode(query.trim().removePrefix("@"),"UTF-8")}&limit=100")
        account.verifyCurrent(service.financialScope())
        val values=response.optJSONArray("players")?.objects().orEmpty()
        values.forEach(::rememberPlayer)
        return values.mapNotNull{value->Players.find{it.name==value.optString("gameId")}}
    }
    suspend fun refreshDirect(name:String) {
        if(name=="AI 助手"){ai?.refresh();return}
        if(name !in conversationIds)loadConversations()
        val id=conversationIds[name] ?: return
        runCatching { api!!.request("GET","/chat/conversations/$id/messages?limit=100") }
            .onSuccess { response->response.getJSONArray("items").objects().asReversed().forEach{addMessage(it,false,name)};directErrors.remove(name);if(name !in directHistoryCursors)directHistoryCursors[name]=cursor(response) }
            .onFailure { directErrors[name]=it.message ?: "消息读取失败" }
        conversation(name).lastOrNull()?.remoteId?.takeIf{foreground&&activeConversation==name&&it.isNotBlank()&&it!=readCursors[id]}?.let{last->
            runCatching { api!!.request("POST","/chat/conversations/$id/read",JSONObject().put("clientRequestId",UUID.randomUUID().toString()).put("lastReadMessageId",last)) }.onSuccess{readCursors[id]=last;conversationUnread[name]=it.optInt("unreadCount").coerceAtLeast(0)}
        }
    }
    private suspend fun refreshPresence(){runCatching{api!!.request("GET","/chat/presence")}.onSuccess{onlineCount=if(it.optBoolean("available"))it.getInt("onlineCount")else null}.onFailure{onlineCount=null}}
    suspend fun loadOlderDirect(name:String){
        val before=directHistoryCursors[name] ?: return;val id=conversationIds[name] ?: return;if(directHistoryLoading[name]==true)return
        directHistoryLoading[name]=true
        runCatching{api!!.request("GET","/chat/conversations/$id/messages?limit=100&cursor=${URLEncoder.encode(before,"UTF-8")}")}.onSuccess{response->response.getJSONArray("items").objects().asReversed().forEach{addMessage(it,false,name)};directHistoryCursors[name]=cursor(response)}.onFailure{directErrors[name]=it.message ?: "历史消息读取失败"}
        directHistoryLoading[name]=false
    }
    suspend fun sendDirect(name: String, text: String, replyTo: ChatReply? = null):Boolean {
        if(name=="AI 助手")return ai?.send(text,replyTo) ?: false
        val service=api ?: return false
        val other=Players.find{it.name==name&&it.playerRef.isNotBlank()} ?: run { storageMessage="请先查询并确认玩家身份";return false }
        if(directPending[name]==true)return false
        val previous=service.pendingDirect(other.playerRef)
        val request=previous ?: JSONObject().put("clientMessageId",UUID.randomUUID().toString()).put("content",text.trim()).apply { replyTo?.remoteId?.takeIf{it.isNotBlank()}?.let{put("replyToMessageId",it)} }
        if(request.getString("content")!=text.trim()||(replyTo!=null&&request.optString("replyToMessageId")!=replyTo.remoteId)){storageMessage="上一条私聊结果待确认，请先重试原消息";return false}
        service.savePendingDirect(other.playerRef,request);directPending[name]=true
        return try { runCatching {
            val id=conversationIds[name] ?: service.request("POST","/chat/conversations",JSONObject()
                .put("clientRequestId",UUID.nameUUIDFromBytes("${service.playerRef}:${other.playerRef}".toByteArray()).toString()).put("otherPlayerRef",other.playerRef)).getString("conversationId").also{conversationIds[name]=it}
            val value=service.request("POST","/chat/conversations/$id/messages",request)
            require(value.optString("messageId").isNotBlank()){"消息发送结果待确认，请重试原消息"}
            addMessage(value,false,name);service.savePendingDirect(other.playerRef,null);directErrors.remove(name);true
        }.getOrElse { error->if(error is ApiFailure&&error.status in listOf(400,401,403,404,422,501))service.savePendingDirect(other.playerRef,null);report(error);false }
        }finally{directPending[name]=false}
    }
    suspend fun forwardMessage(line:ChatLine,sourceName:String?,targetName:String):Boolean {
        val service=api ?: return false
        val other=Players.find{it.name==targetName&&it.playerRef.isNotBlank()} ?: return false
        val sourceId=sourceName?.let{conversationIds[it]}
        if(line.remoteId.isBlank()||(sourceName!=null&&sourceId==null)){storageMessage="原消息尚未同步，暂时无法转发";return false}
        val previous=service.pendingForward(other.playerRef)
        if(previous!=null&&previous.optString("sourceMessageId")!=line.remoteId){storageMessage="上一条转发结果待确认，请先重试原转发";return false}
        val request=previous ?: JSONObject().put("clientMessageId",UUID.randomUUID().toString()).put("sourceMessageId",line.remoteId).apply{sourceId?.let{put("sourceConversationId",it)}}
        service.savePendingForward(other.playerRef,request)
        return runCatching {
            val id=conversationIds[targetName] ?: service.request("POST","/chat/conversations",JSONObject()
                .put("clientRequestId",UUID.nameUUIDFromBytes("${service.playerRef}:${other.playerRef}".toByteArray()).toString()).put("otherPlayerRef",other.playerRef)).getString("conversationId").also{conversationIds[targetName]=it}
            val result=service.request("POST","/chat/conversations/$id/forwards",request)
            require(result.optString("messageId").isNotBlank()){"转发结果待确认，请使用原消息重试"}
            addMessage(result,false,targetName);service.savePendingForward(other.playerRef,null);true
        }.getOrElse{error->if(error is ApiFailure&&error.status in listOf(400,401,403,404,422,501))service.savePendingForward(other.playerRef,null);report(error);false}
    }
    fun reset() { cart.clear(); refresh(); loadAnnouncements() }
    fun close() { closed = true; reconnect?.cancel(); chatSocket?.close() }
    private fun report(error: Throwable) { storageMessage = error.message ?: "服务连接失败，请稍后重试" }
}
