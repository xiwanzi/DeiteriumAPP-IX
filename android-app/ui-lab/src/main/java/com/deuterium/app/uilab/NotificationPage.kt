package com.deuterium.app.uilab

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

@Composable
fun NotificationPage(topInset:androidx.compose.ui.unit.Dp,allowed:Boolean,userName:String,onEnable:()->Unit,onSettings:()->Unit) {
    val context=LocalContext.current
    val preferences=remember(userName){NotificationPreferences(context,userName)}
    val scope=rememberCoroutineScope()
    LaunchedEffect(userName){preferences.sync()}
    LazyColumn(contentPadding=PaddingValues(start=16.dp,end=16.dp,top=topInset,bottom=45.dp),verticalArrangement=Arrangement.spacedBy(20.dp)) {
        item{SettingsGroup{SettingsRow("系统通知",Icons.Outlined.NotificationsNone,detail=if(allowed)"已允许" else "未开启",onClick=onSettings)
            if(!allowed)PlainButton(onEnable,Modifier.fillMaxWidth()){Text("允许系统通知")}}}
        item{SettingsGroup{ToggleRow("接收推送通知","在系统通知栏接收消息与业务提醒",preferences.get("enabled"),{value->scope.launch{preferences.update("enabled",value)}})}}
        val groups=listOf("消息" to listOf(NotificationTopic.Direct,NotificationTopic.Mentions,NotificationTopic.Followed),"交易" to listOf(NotificationTopic.Wallet,NotificationTopic.Market,NotificationTopic.Commissions),"服务器与应用" to listOf(NotificationTopic.Announcements,NotificationTopic.Updates))
        groups.forEach{(title,topics)->
            item{Text(title,Modifier.padding(start=18.dp,bottom=5.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant);SettingsGroup{topics.forEachIndexed{index,topic->if(index>0)SettingsDivider();ToggleRow(topic.title,topic.detail,preferences.get(topic.key),{value->scope.launch{preferences.update(topic.key,value)}},enabled=preferences.get("enabled"))}}}
        }
        item{SettingsGroup{ToggleRow("通知内容预览","在提醒中显示消息与订单摘要",preferences.get("showPreviews"),{value->scope.launch{preferences.update("showPreviews",value)}},enabled=preferences.get("enabled"))}}
        preferences.error?.let{error->item{Text(error,Modifier.padding(horizontal=18.dp),color=MaterialTheme.colorScheme.error)}}
        item{Text("关闭推送不会删除消息、订单或钱包记录。提示音、振动与锁屏显示由系统通知设置管理。",Modifier.padding(horizontal=18.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
    }
}
