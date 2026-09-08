package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.draw.clip
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

data class ChatReply(val messageId:Long,val name:String,val text:String,val remoteId:String="")

@Composable
fun QuotedMessage(reply:ChatReply,mine:Boolean=false,modifier:Modifier=Modifier) {
    val ink=if(mine)MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.primary
    Row(modifier.fillMaxWidth().background(ink.copy(alpha=.09f),RoundedCornerShape(9.dp)).padding(9.dp),verticalAlignment=Alignment.CenterVertically){
        Box(Modifier.width(2.dp).height(31.dp).background(ink.copy(alpha=.65f),CircleShape))
        Column(Modifier.padding(start=9.dp)){
            Text(reply.name,style=MaterialTheme.typography.labelSmall,color=ink)
            Text(reply.text,style=MaterialTheme.typography.bodySmall,color=if(mine)ink.copy(alpha=.82f) else MaterialTheme.colorScheme.onSurfaceVariant,maxLines=2,overflow=TextOverflow.Ellipsis)
        }
    }
}

@Composable
fun ReplyPreview(reply:ChatReply,onCancel:()->Unit) {
    Row(Modifier.fillMaxWidth().padding(horizontal=15.dp).clip(RoundedCornerShape(15.dp)).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.5f)).padding(start=12.dp),verticalAlignment=Alignment.CenterVertically){
        Icon(Icons.Outlined.Reply,null,Modifier.size(20.dp),tint=MaterialTheme.colorScheme.primary)
        Column(Modifier.weight(1f).padding(horizontal=9.dp,vertical=9.dp)){Text("回复 ${reply.name}",style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.primary);Text(reply.text,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=1,overflow=TextOverflow.Ellipsis)}
        IconButton(onCancel){Icon(Icons.Outlined.Close,"取消回复",Modifier.size(19.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant)}
    }
}

@Composable
fun MessageActionSheet(line:ChatLine,onClose:()->Unit,onForward:(()->Unit)?=null,onReply:()->Unit) {
    val clipboard=LocalClipboardManager.current;val context=LocalContext.current
    IosSheet(onClose){Column(Modifier.fillMaxWidth().padding(22.dp)){
        Text(line.name,style=MaterialTheme.typography.titleMedium)
        Text(line.text,Modifier.padding(top=9.dp,bottom=20.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=4,overflow=TextOverflow.Ellipsis)
        SettingsGroup{
            SettingsRow("复制",Icons.Outlined.ContentCopy,onClick={clipboard.setText(AnnotatedString(line.text));onClose();android.widget.Toast.makeText(context,"已复制消息",android.widget.Toast.LENGTH_SHORT).show()})
            SettingsDivider();SettingsRow("回复",Icons.Outlined.Reply,onClick={onClose();onReply()})
            if(onForward!=null){SettingsDivider();SettingsRow("转发",Icons.Outlined.Forward,onClick={onClose();onForward()})}
        }
        PlainButton(onClose,Modifier.fillMaxWidth().padding(top=8.dp)){Text("取消")}
    }}
}

@Composable
fun ForwardMessageSheet(state:LabState,line:ChatLine,sourceName:String?,onClose:()->Unit) {
    var query by remember{mutableStateOf("")};var busy by remember{mutableStateOf(false)}
    val scope=rememberCoroutineScope();val context=LocalContext.current
    LaunchedEffect(query){if(query.isNotBlank()){delay(300);runCatching{state.searchRecipients(query)}}}
    IosSheet({if(!busy)onClose()}){Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(24.dp)){
        Text("转发给",style=MaterialTheme.typography.headlineSmall)
        Text(line.text,Modifier.padding(vertical=15.dp),maxLines=3,overflow=TextOverflow.Ellipsis,style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        RefinedField(query,{query=it.take(32)},Modifier.fillMaxWidth(),label={Text("搜索玩家 ID 或 QQ")},singleLine=true)
        val people=searchPlayers(query).filter{it.name!=state.userName&&it.playerRef.isNotBlank()}
        if(people.isEmpty())Text("没有找到联系人",Modifier.padding(vertical=20.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)
        people.forEach{person->PlainButton({if(!busy)scope.launch{busy=true;if(state.forwardMessage(line,sourceName,person.name)){android.widget.Toast.makeText(context,"已转发给 ${person.name}",android.widget.Toast.LENGTH_SHORT).show();onClose()};busy=false}},Modifier.fillMaxWidth().padding(top=8.dp),enabled=!busy){Text(person.name)}}
        if(busy)Text("正在转发…",Modifier.padding(top=12.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)
    }}
}
