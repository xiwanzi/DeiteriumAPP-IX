package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.unit.*
import kotlinx.coroutines.launch

@Composable
fun PlayerProfilePage(name:String,state:LabState,avatar:String?,topInset:Dp,onMessage:()->Unit,onTransfer:()->Unit) {
    if(state.isUnavailableAccount(name)){
        Box(Modifier.fillMaxSize().padding(top=topInset),contentAlignment=Alignment.Center){Text("该账号已注销",color=MaterialTheme.colorScheme.onSurfaceVariant)}
        return
    }
    LaunchedEffect(name){state.loadProfile(name)}
    val own=name==state.userName
    val player=Players.find{it.name==name}
    if(!own&&player==null&&name!="AI 助手"){
        Box(Modifier.fillMaxSize().padding(top=topInset),contentAlignment=Alignment.Center){Text("用户资料不可用",color=MaterialTheme.colorScheme.onSurfaceVariant)}
        return
    }
    val qq=if(own)state.api?.user?.optString("qq")?.takeIf{it.isNotBlank()} ?: "尚未公开" else player?.qq?.takeIf{it.isNotBlank()} ?: "尚未公开"
    val bio=if(own)state.profileBio.ifBlank{"还没有填写个人简介"} else player?.bio ?: "Deuterium 服务与帮助"
    val clipboard=LocalClipboardManager.current
    LazyColumn(contentPadding=PaddingValues(start=20.dp,end=20.dp,top=topInset,bottom=50.dp),verticalArrangement=Arrangement.spacedBy(22.dp)) {
        item{Column(Modifier.fillMaxWidth().padding(top=15.dp,bottom=5.dp),horizontalAlignment=Alignment.CenterHorizontally){if(own)LocalAvatar(avatar,name,Modifier.size(98.dp)) else PlayerAvatar(name,Modifier.size(98.dp));Text(name,Modifier.padding(top=17.dp),style=MaterialTheme.typography.headlineSmall);Text(if(own||player?.online==true)"当前在线" else "上次在线：${player?.lastSeen ?: "暂无记录"}",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
        item{Row(horizontalArrangement=Arrangement.spacedBy(10.dp)){
            ProfileAction("消息",Icons.Outlined.ChatBubbleOutline,Modifier.weight(1f),onMessage)
            ProfileAction("特别关心",if(name in state.followed)Icons.Outlined.Star else Icons.Outlined.StarOutline,Modifier.weight(1f),{state.toggleFollow(name)},active=name in state.followed,enabled=!own)
            ProfileAction("转账",Icons.Outlined.AccountBalanceWallet,Modifier.weight(1f),onTransfer,enabled=!own&&player!=null)
        }}
        item{LabCard{Text("个人简介",style=MaterialTheme.typography.titleMedium);Text(bio,Modifier.padding(top=11.dp),style=MaterialTheme.typography.bodyLarge,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
        item{LabCard{DetailRow("玩家 ID",name);Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){Text("QQ",Modifier.weight(1f),color=MaterialTheme.colorScheme.onSurfaceVariant);Text(qq);IconButton({clipboard.setText(AnnotatedString(qq))},enabled=qq!="尚未填写"){Icon(Icons.Outlined.ContentCopy,"复制QQ",Modifier.size(18.dp),tint=MaterialTheme.colorScheme.primary)}}}}
    }
}
@Composable
private fun ProfileAction(title:String,icon:ImageVector,modifier:Modifier,onClick:()->Unit,active:Boolean=false,enabled:Boolean=true) {
    Surface(onClick=onClick,modifier=modifier,enabled=enabled,shape=RoundedCornerShape(19.dp),color=if(active)MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surface){Column(Modifier.fillMaxWidth().padding(vertical=16.dp),horizontalAlignment=Alignment.CenterHorizontally){Icon(icon,null,Modifier.size(24.dp),tint=MaterialTheme.colorScheme.primary.copy(alpha=if(enabled)1f else .35f));Text(title,Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurface.copy(alpha=if(enabled)1f else .4f))}}
}
@Composable
fun BioEditorPage(state:LabState,topInset:Dp,onDone:()->Unit) {
    var draft by rememberSaveable{mutableStateOf(state.profileBio)}
    val scope=rememberCoroutineScope();var busy by remember{mutableStateOf(false)}
    Column(Modifier.fillMaxSize().imePadding().padding(horizontal=20.dp).padding(top=topInset)) {
        Text("让大家认识你",style=MaterialTheme.typography.headlineSmall)
        Text("这段简介会显示在你的玩家资料页。",Modifier.padding(top=9.dp,bottom=22.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        RefinedField(draft,{draft=it.take(200)},Modifier.fillMaxWidth(),placeholder={Text("介绍一下你的兴趣、擅长的事情或游戏计划")},minLines=5,maxLines=8)
        Text("${draft.length}/200",Modifier.align(Alignment.End).padding(top=10.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
        MotionButton({if(!busy)scope.launch{busy=true;if(state.saveBio(draft))onDone();busy=false}},Modifier.fillMaxWidth().padding(top=24.dp).height(52.dp),enabled=!busy){Text(if(busy)"正在保存…" else "保存简介")}
    }
}
