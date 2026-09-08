package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.shape.*
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Send
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.*
import androidx.compose.ui.platform.LocalUriHandler

@Composable
fun InfoPage(state:LabState,topInset:Dp,query:String,onOpen:(String)->Unit) {
    LaunchedEffect(state){while(true){state.refreshContacts();delay(15000)}}
    LazyColumn(contentPadding=PaddingValues(start=20.dp,end=20.dp,top=topInset,bottom=120.dp),verticalArrangement=Arrangement.spacedBy(18.dp)) {
        if(query.isBlank()) {
            item { Surface(shape=RoundedCornerShape(24.dp),color=MaterialTheme.colorScheme.surface) { Column {
                HubRow("官方公告","服务器维护安排与近期更新",DeuteriumIcons.Announcement,Color(0xFFE99C38),"置顶"){onOpen("announcement")}
                HorizontalDivider(Modifier.padding(start=76.dp),color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.45f))
                HubRow("委托大厅","发布需求，完成委托获得报酬",DeuteriumIcons.Commission,Color(0xFF8072CF),"置顶"){onOpen("commissions") }
                HorizontalDivider(Modifier.padding(start=76.dp),color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.45f))
                HubRow("交易通知","订单、退款与结算提醒",Icons.Outlined.NotificationsNone,MaterialTheme.colorScheme.primary,state.commerce.notices.count{it.recipient==state.userName&&!it.read}.takeIf{it>0}?.toString().orEmpty()){onOpen("trade-notices")}
            } } }
            item { Surface(shape=RoundedCornerShape(24.dp),color=MaterialTheme.colorScheme.surface) { Column {
                HubRow("公共聊天",state.chat.lastOrNull()?.let{"${it.name}：${it.text}"} ?: "和服务器玩家聊一聊",DeuteriumIcons.PublicChat,MaterialTheme.colorScheme.primary,state.onlineCount?.let{"$it 在线"}.orEmpty()){onOpen("public")}
                HorizontalDivider(Modifier.padding(start=76.dp),color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.45f))
                HubRow("小祥 AI",state.directChats["AI 助手"]?.lastOrNull()?.text ?: "建筑灵感、玩法建议，随时聊聊",Icons.Outlined.AutoAwesome,Color(0xFF5778BC),""){onOpen("ai")}
            } } }
        }
        item { Text("联系人",Modifier.padding(start=4.dp),style=MaterialTheme.typography.titleMedium,color=MaterialTheme.colorScheme.onSurfaceVariant) }
        val people=recentContacts(searchPlayers(query),state.followed.toSet(),state.directChats,searchMode=query.isNotBlank())
        if(people.isEmpty())item{Text(if(query.isNotBlank())"没有找到联系人" else state.contactsError ?: if(!state.contactsKnown)"正在同步联系人…" else "还没有私聊记录，搜索玩家开始聊天",Modifier.padding(20.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)}
        item { Surface(shape=RoundedCornerShape(24.dp),color=MaterialTheme.colorScheme.surface) { Column {
            people.forEachIndexed { index,entry ->key(entry.person.playerRef.ifBlank{entry.person.name}){
                val person=entry.person
                Row(Modifier.fillMaxWidth().clickable{onOpen("dm:${person.name}")}.padding(16.dp),verticalAlignment=Alignment.CenterVertically) {
                    Box { PlayerAvatar(person.name,Modifier.size(46.dp).clickable{onOpen("player:${person.name}")});if(person.online)Box(Modifier.align(Alignment.BottomEnd).size(10.dp).background(Color(0xFF42AF72),CircleShape).border(2.dp,MaterialTheme.colorScheme.surface,CircleShape)) }
                    Column(Modifier.weight(1f).padding(horizontal=13.dp)) {
                        Row(verticalAlignment=Alignment.CenterVertically){Text(person.name,style=MaterialTheme.typography.titleMedium);if(person.name in state.followed)Icon(Icons.Outlined.Star,null,Modifier.padding(start=6.dp).size(13.dp),tint=MaterialTheme.colorScheme.primary)}
                        Text(entry.latest?.text ?: person.bio,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=1,overflow=TextOverflow.Ellipsis)
                    }
                    Text(entry.latest?.time ?: if(person.online)"在线" else if(person.lastSeen=="暂无记录")"—" else "离线",style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                }
                if(index<people.lastIndex)HorizontalDivider(Modifier.padding(start=76.dp),color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.4f))
            }}
        } } }
    }
}

@Composable
private fun HubRow(title:String,subtitle:String,icon:ImageVector,color:Color,aside:String,onClick:()->Unit) {
    Row(Modifier.fillMaxWidth().clickable(onClick=onClick).padding(16.dp),verticalAlignment=Alignment.CenterVertically) {
        Box(Modifier.size(46.dp).background(color.copy(alpha=.13f),if(title=="小祥 AI")CircleShape else RoundedCornerShape(16.dp)),contentAlignment=Alignment.Center){
            if(title=="小祥 AI")Image(painterResource(R.drawable.xiaoxiang_avatar),null,Modifier.size(46.dp).clip(CircleShape)) else Icon(icon,null,tint=color,modifier=Modifier.size(23.dp))
        }
        Column(Modifier.weight(1f).padding(horizontal=13.dp)){Text(title,style=MaterialTheme.typography.titleMedium);Text(subtitle,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=1,overflow=TextOverflow.Ellipsis)}
        if(aside.isNotBlank())Text(aside,style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
fun DirectChatPage(state:LabState,name:String,topInset:Dp,onProfile:(String)->Unit) {
    val messages=remember(name){state.conversation(name)}
    var draft by rememberSaveable(name,stateSaver=TextFieldValue.Saver){mutableStateOf(TextFieldValue(state.pendingDirectDraft(name)))}
    val list=rememberLazyListState()
    var selectedMessage by remember(name){mutableStateOf<ChatLine?>(null)}
    var forwarding by remember(name){mutableStateOf<ChatLine?>(null)}
    var aiOptions by remember{mutableStateOf(false)}
    val uriHandler=LocalUriHandler.current
    var replyId by rememberSaveable(name){mutableStateOf<Long?>(null)}
    val replying=messages.find{it.id==replyId}?.let{ChatReply(it.id,if(it.mine)state.userName else it.name,it.text,it.remoteId)}
    val focus=remember{FocusRequester()};val keyboard=LocalSoftwareKeyboardController.current;val scope=rememberCoroutineScope()
    val motion=LocalMotion.current
    fun send(){val text=draft.text;if(text.isNotBlank())scope.launch{if(state.sendDirect(name,text,replying)&&draft.text==text){draft=TextFieldValue("");replyId=null}}}
    LaunchedEffect(name){while(true){state.refreshDirect(name);delay(4000)}}
    LaunchedEffect(state.ai?.recoveredDraft){if(name=="AI 助手"&&draft.text.isBlank()&&!state.ai?.recoveredDraft.isNullOrBlank())draft=TextFieldValue(state.ai!!.recoveredDraft)}
    LaunchedEffect(messages.lastOrNull()?.id){if(list.firstVisibleItemIndex<2||messages.lastOrNull()?.mine==true){if(motion)list.animateScrollToItem(0) else list.scrollToItem(0)}}
    LaunchedEffect(list.firstVisibleItemIndex,messages.size){if(messages.size>=90&&list.firstVisibleItemIndex>=messages.size-12)state.loadOlderDirect(name)}
    Column(Modifier.fillMaxSize()) {
        LazyColumn(Modifier.weight(1f).fillMaxWidth(),state=list,reverseLayout=true,contentPadding=PaddingValues(start=20.dp,end=20.dp,top=topInset+18.dp,bottom=16.dp),verticalArrangement=Arrangement.spacedBy(10.dp,Alignment.Bottom)) {
            if(state.directPending[name]==true)item{Text(if(name=="AI 助手")state.ai?.statusText ?: "正在生成回复…" else "正在发送…",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
            state.directErrors[name]?.let{error->item{Text(error,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.error)}}
            items(messages.asReversed(),key={it.id}) { line -> Row(Modifier.fillMaxWidth(),horizontalArrangement=if(line.mine)Arrangement.End else Arrangement.Start) {
                if(!line.mine){if(name=="AI 助手")Image(painterResource(R.drawable.xiaoxiang_avatar),"小祥菜单",Modifier.size(42.dp).clip(CircleShape).clickable{aiOptions=true}) else PlayerAvatar(name,Modifier.clickable{onProfile(name)});Spacer(Modifier.width(9.dp))}
                Column(Modifier.widthIn(max=270.dp),horizontalAlignment=if(line.mine)Alignment.End else Alignment.Start) {
                    Surface(modifier=Modifier.combinedClickable(onClick={},onLongClick={selectedMessage=line}),shape=RoundedCornerShape(topStart=19.dp,topEnd=19.dp,bottomStart=if(line.mine)19.dp else 6.dp,bottomEnd=if(line.mine)6.dp else 19.dp),color=if(line.mine)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surface,
                        contentColor=if(line.mine)MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurface){Column(Modifier.padding(horizontal=13.dp,vertical=9.dp)){line.forwarded?.let{Text("转发自 ${it.name}",style=MaterialTheme.typography.labelSmall,color=if(line.mine)MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.primary);Spacer(Modifier.height(5.dp))};line.reply?.let{QuotedMessage(it,line.mine,Modifier.padding(bottom=7.dp))};Text(line.text,style=MaterialTheme.typography.bodyLarge.copy(fontSize=16.sp,lineHeight=23.sp))}}
                    if(line.mine&&line.id==messages.lastOrNull{it.mine}?.id)Text("已发送",Modifier.padding(4.dp),style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    if(name=="AI 助手"&&!line.mine){
                        if(line.aiStatus in setOf("unknown","incomplete","failed"))Text("回复未完整完成",Modifier.padding(top=5.dp),style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.error)
                        line.sources.forEach{source->PlainButton({uriHandler.openUri(source.url)},Modifier.fillMaxWidth()){Text((if(source.origin=="annotation")"引用来源 · " else "模型返回链接 · ")+source.title,style=MaterialTheme.typography.labelSmall,maxLines=2,overflow=TextOverflow.Ellipsis)}}
                    }
                }
            } }
            item { Text(if(name=="AI 助手")state.ai?.quotaText ?: "DEUTERIUM ASSISTANT" else "今天",Modifier.fillMaxWidth().padding(vertical=12.dp),style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant,textAlign=androidx.compose.ui.text.style.TextAlign.Center) }
            if(state.directHistoryLoading[name]==true)item{Text("正在读取历史消息…",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
        }
        if(name=="AI 助手"&&messages.isEmpty())Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal=20.dp),horizontalArrangement=Arrangement.spacedBy(8.dp)) {
            listOf("给我一些建筑建议","怎样使用市场").forEach{AssistChoice({scope.launch{state.sendDirect(name,it)}},label={Text(it)})}
        }
        MessageComposer(draft,{draft=it.copy(text=it.text.take(if(name=="AI 助手")2000 else 256))},{send()},if(name=="AI 助手")"向 AI 提问" else "发送消息",focus,replying,{replyId=null})
    }
    selectedMessage?.let{line->MessageActionSheet(line,{selectedMessage=null},onForward=if(name!="AI 助手")({forwarding=line})else null){replyId=line.id;scope.launch{delay(220);focus.requestFocus();withFrameNanos{};keyboard?.show()}}}
    forwarding?.let{ForwardMessageSheet(state,it,name){forwarding=null}}
    if(aiOptions)state.ai?.let{AiOptionsSheet(it,{aiOptions=false}){draft=TextFieldValue("");replyId=null;aiOptions=false}}
}

@Composable
private fun AiOptionsSheet(ai:BackendAI,onClose:()->Unit,onReset:()->Unit){
    val scope=rememberCoroutineScope();var resetting by remember{mutableStateOf(false)}
    LaunchedEffect(Unit){ai.loadPlans()}
    IosSheet(onClose){Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(24.dp),verticalArrangement=Arrangement.spacedBy(14.dp)){
        Text("小祥 AI",style=MaterialTheme.typography.headlineSmall);Text(ai.quotaText,color=MaterialTheme.colorScheme.onSurfaceVariant)
        MotionButton({if(!resetting)scope.launch{resetting=true;if(ai.reset())onReset();resetting=false}},Modifier.fillMaxWidth(),enabled=!ai.busy&&!resetting){Text(if(resetting)"正在开启…" else "新对话")}
        ai.plans.forEach{plan->LabCard{Text(plan.getString("name"),style=MaterialTheme.typography.titleMedium);Text(if(plan.optString("code")=="free")"免费 · ${plan.getInt("quotaPerWindow")} 次 / ${plan.getInt("windowHours")} 小时" else "${plan.getString("price")} 信用点 · 暂未开放",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
    }}
}

@Composable
fun CommunityDetail(event:Boolean,state:LabState,topInset:Dp) {
    LaunchedEffect(Unit){state.loadAnnouncements()}
    LazyColumn(contentPadding=PaddingValues(start=24.dp,end=24.dp,top=topInset,bottom=48.dp),verticalArrangement=Arrangement.spacedBy(24.dp)) {
        if(state.announcements.isEmpty())item{Text(state.announcementError ?: "暂无公告",Modifier.padding(vertical=32.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)}
        items(state.announcements,key={it.id}){entry->LabCard{Text(entry.title,style=MaterialTheme.typography.headlineSmall);Text(entry.publishedAt,Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant);Text(entry.content,Modifier.padding(top=18.dp),style=MaterialTheme.typography.bodyLarge)}}
    }
}
