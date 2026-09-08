package com.deuterium.app.uilab

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Send
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.text.*
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.contentDescription
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun PlayerAvatar(name: String, modifier: Modifier = Modifier) {
    val avatarModifier=modifier.size(42.dp)
    val account=LocalAccountAvatar.current
    val source=if(account?.name==name)account.avatar else Players.find{it.name==name}?.avatar
    source?.let{LocalAvatar(it,name,avatarModifier);return}
    val dark = MaterialTheme.colorScheme.background.red < .5f
    val color = if(dark) MaterialTheme.colorScheme.secondaryContainer else when(name) {
        "Luna" -> Color(0xFFF2E5D8); "Aster" -> Color(0xFFE7E4F1); else -> MaterialTheme.colorScheme.primaryContainer
    }
    Box(avatarModifier.clip(CircleShape).background(color), contentAlignment = Alignment.Center) {
        Text(name.take(1).uppercase(), color = MaterialTheme.colorScheme.onSecondaryContainer, fontSize = 18.sp, lineHeight = 24.sp, fontWeight = FontWeight.SemiBold)
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
fun ChatPage(state: LabState, topInset: Dp, onProfile:(String)->Unit, onTransfer: (String) -> Unit) {
    val motion = LocalMotion.current
    val list = rememberLazyListState()
    val clipboard = LocalClipboardManager.current
    val haptics = LocalHapticFeedback.current
    val keyboard = LocalSoftwareKeyboardController.current
    val focus = remember { FocusRequester() }
    val scope = rememberCoroutineScope()
    var selectedMessage by remember{mutableStateOf<ChatLine?>(null)}
    var forwarding by remember{mutableStateOf<ChatLine?>(null)}
    var copied by remember { mutableStateOf(false) }
    var player by remember { mutableStateOf<PlayerProfile?>(null) }
    val nearBottom by remember { derivedStateOf { list.firstVisibleItemIndex == 0 && list.firstVisibleItemScrollOffset < 80 } }
    var previousCount by remember { mutableIntStateOf(state.chat.size) }
    var previousNewest by remember { mutableStateOf(state.chat.lastOrNull()?.id) }
    var unread by remember { mutableIntStateOf(0) }
    val activeMention = mentionRange(state.chatDraft)
    val query = activeMention?.let { state.chatDraft.text.substring(it.first + 1, it.last + 1) }
    val liveCandidates = if(query == null) emptyList() else searchPlayers(query).sortedByDescending { it.name in state.followed }.take(5)
    var lastCandidates by remember { mutableStateOf<List<PlayerProfile>>(emptyList()) }
    SideEffect { if(query != null) lastCandidates = liveCandidates }
    val candidates = if(query == null) lastCandidates else liveCandidates
    fun mention(name: String) {
        state.mentionPlayer(name); player = null
        scope.launch { withFrameNanos { }; focus.requestFocus(); keyboard?.show() }
    }
    LaunchedEffect(state.chat.size) {
        val added = state.chat.size > previousCount && state.chat.lastOrNull()?.id != previousNewest; previousCount = state.chat.size;previousNewest=state.chat.lastOrNull()?.id
        if(nearBottom || (added && state.chat.lastOrNull()?.mine == true)) {
            if(motion) list.animateScrollToItem(0) else list.scrollToItem(0)
            unread = 0
        } else if(added) unread++
    }
    LaunchedEffect(list.firstVisibleItemIndex,state.chat.size){if(state.chat.size>=90&&list.firstVisibleItemIndex>=state.chat.size-12)state.loadOlderPublic()}
    Column(Modifier.fillMaxSize()) {
        Box(Modifier.weight(1f)) {
            LazyColumn(state = list, modifier = Modifier.fillMaxSize(), reverseLayout = true,
                contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = topInset + 12.dp, bottom = 8.dp), verticalArrangement = Arrangement.spacedBy(10.dp, Alignment.Bottom)) {
                if(state.chatReplyPending) item(key = "typing") { TypingStatus("正在发送…") }
                items(state.chat.asReversed(), key = { it.id }, contentType = { "message" }) { line ->
                    Row(Modifier.fillMaxWidth().then(if(motion) Modifier.animateItem() else Modifier), horizontalArrangement = if(line.mine) Arrangement.End else Arrangement.Start) {
                        if(!line.mine) {
                            PlayerAvatar(line.name, Modifier.clip(RoundedCornerShape(15.dp)).semantics { contentDescription = "玩家${line.name}头像" }.combinedClickable(
                                onClickLabel = "查看${line.name}资料", onLongClickLabel = "提及${line.name}",
                                onClick = { onProfile(line.name) },
                                onLongClick = { haptics.performHapticFeedback(HapticFeedbackType.LongPress); mention(line.name) }))
                            Spacer(Modifier.width(9.dp))
                        }
                        Column(Modifier.widthIn(max = 268.dp), horizontalAlignment = if(line.mine) Alignment.End else Alignment.Start) {
                            if(!line.mine) Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(line.name, Modifier.padding(start = 3.dp, bottom = 5.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                if(line.name in state.followed) Icon(Icons.Outlined.Star, "特别关心", Modifier.padding(start = 4.dp, bottom = 5.dp).size(13.dp), tint = MaterialTheme.colorScheme.primary)
                            }
                            val shape = RoundedCornerShape(topStart = 19.dp, topEnd = 19.dp, bottomStart = if(line.mine)19.dp else 6.dp, bottomEnd = if(line.mine)6.dp else 19.dp)
                            Surface(shape = shape, color = if(line.mine) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surface,
                                contentColor = if(line.mine) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurface,
                                modifier = Modifier.clip(shape).combinedClickable(onClick = {}, onLongClick = {
                                    haptics.performHapticFeedback(HapticFeedbackType.LongPress);selectedMessage=line
                                })) {
                                val annotated = buildAnnotatedString {
                                    append(line.text)
                                    Regex("@[A-Za-z0-9_]+").findAll(line.text).forEach { addStyle(SpanStyle(fontWeight = FontWeight.Bold, color = if(line.mine) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.primary), it.range.first, it.range.last+1) }
                                }
                                Column(Modifier.padding(horizontal=13.dp,vertical=9.dp)){
                                    line.reply?.let{QuotedMessage(it,line.mine,Modifier.padding(bottom=7.dp))}
                                    Text(annotated,style=MaterialTheme.typography.bodyLarge.copy(fontSize=16.sp,lineHeight=23.sp))
                                }
                            }
                            if(line.mine&&line.id==state.chat.lastOrNull{it.mine}?.id)Text("已发送", Modifier.padding(horizontal = 3.dp, vertical = 4.dp), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                }
                item(key = "day") { Text(state.chatStatus, Modifier.fillMaxWidth().padding(vertical = 6.dp), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = androidx.compose.ui.text.style.TextAlign.Center) }
                if(state.loadingChatHistory)item(key="history-loading"){Text("正在读取历史消息…",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
            }
            if(unread > 0 && !nearBottom) FilledTonalButton(onClick = { scope.launch { list.animateScrollToItem(0); unread = 0 } }, modifier = Modifier.align(Alignment.BottomCenter)) { Text("↓ $unread 条新消息") }
            androidx.compose.animation.AnimatedVisibility(query != null, modifier = Modifier.align(Alignment.BottomCenter).padding(horizontal = 16.dp, vertical = 8.dp),
                enter = fadeIn(tween(if(motion) 140 else 0)) + slideInVertically(tween(if(motion) 180 else 0)) { it / 3 },
                exit = fadeOut(tween(if(motion) 100 else 0)) + slideOutVertically { it / 4 }) {
                LiquidGlass(Modifier.fillMaxWidth(), radius = 22.dp) {
                    Column(Modifier.padding(8.dp)) {
                        Text("提及玩家", Modifier.padding(horizontal = 12.dp, vertical = 7.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        if(candidates.isEmpty()) Text("没有找到匹配的玩家", Modifier.padding(14.dp), style = MaterialTheme.typography.bodyMedium)
                        LazyColumn(Modifier.heightIn(max = 220.dp)) {
                            items(candidates, key = { it.name }) { candidate ->
                                Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(16.dp)).semantics { contentDescription = "提及${candidate.name}" }.clickable { mention(candidate.name) }.padding(horizontal = 10.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                                    PlayerAvatar(candidate.name, Modifier.size(32.dp))
                                    Text(candidate.name, Modifier.weight(1f).padding(start = 10.dp), style = MaterialTheme.typography.titleMedium)
                                    if(candidate.name in state.followed) Icon(Icons.Outlined.Star, null, Modifier.size(16.dp), tint = MaterialTheme.colorScheme.primary)
                                }
                            }
                        }
                    }
                }
            }
        }
        if(copied) Text("已复制消息", Modifier.padding(horizontal = 24.dp, vertical = 4.dp), color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.bodySmall)
        MessageComposer(state.chatDraft,{if(it.text.length<=256)state.chatDraft=it},state::sendChat,"发消息，输入 @ 提及玩家",focus,state.chatReplyTo,{state.chatReplyTo=null})
    }
    selectedMessage?.let{line->MessageActionSheet(line,{selectedMessage=null},onForward={forwarding=line}){state.chatReplyTo=ChatReply(line.id,if(line.mine)state.userName else line.name,line.text,line.remoteId);scope.launch{delay(220);focus.requestFocus();withFrameNanos{};keyboard?.show()}}}
    forwarding?.let{ForwardMessageSheet(state,it,null){forwarding=null}}
    player?.let { selected -> PlayerDetails(selected, state, { mention(selected.name) }, { player = null; onTransfer(selected.name) }, { player = null }) }
}

@Composable
private fun TypingStatus(label: String) {
    val motion = LocalMotion.current
    Row(Modifier.padding(start = 4.dp, top = 4.dp, bottom = 6.dp), verticalAlignment = Alignment.CenterVertically) {
        if(motion) {
            val transition = rememberInfiniteTransition(label = "typing")
            repeat(3) { index ->
                val alpha by transition.animateFloat(.25f, 1f, infiniteRepeatable(tween(450, delayMillis = index * 110), RepeatMode.Reverse), label = "dot-$index")
                Box(Modifier.padding(end = 4.dp).size(5.dp).graphicsLayer { this.alpha = alpha }.background(MaterialTheme.colorScheme.primary, CircleShape))
            }
        }
        Text(label, Modifier.padding(start = 5.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}
