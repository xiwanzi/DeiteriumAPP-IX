package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.*
import java.time.*
import java.time.format.DateTimeFormatter

data class ServerEvent(val id:String,val title:String,val summary:String,val location:String,val start:LocalDateTime,val end:LocalDateTime,val art:String,val participants:Int) {
    val status:String get()=when{LocalDateTime.now()<start->"即将开始";LocalDateTime.now()>end->"已结束";else->"进行中"}
}
@Composable
fun EventsPage(state:LabState,topInset:Dp,onEvent:(String)->Unit) {
    var filter by rememberSaveable{mutableStateOf("全部")}
    val events=state.events.filter{filter=="全部"||it.status==filter}
    LazyColumn(contentPadding=PaddingValues(start=18.dp,end=18.dp,top=topInset,bottom=40.dp),verticalArrangement=Arrangement.spacedBy(16.dp)) {
        item{Text("一起参与",style=MaterialTheme.typography.headlineLarge);Text("发现正在发生和即将开始的服务器活动",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)}
        item{Row(Modifier.horizontalScroll(rememberScrollState()),horizontalArrangement=Arrangement.spacedBy(8.dp)){listOf("全部","进行中","即将开始","已结束").forEach{ChoiceChip(filter==it,{filter=it},{Text(it)})}}}
        items(events,key={it.id}){event->Surface(onClick={onEvent(event.id)},shape=RoundedCornerShape(24.dp),color=MaterialTheme.colorScheme.surface){Column(Modifier.padding(18.dp)){
            Row(verticalAlignment=Alignment.CenterVertically){ProductArt(event.art,Modifier.size(90.dp).background(MaterialTheme.colorScheme.primaryContainer.copy(alpha=.3f),RoundedCornerShape(19.dp)));Column(Modifier.weight(1f).padding(start=15.dp)){Text(event.status,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.primary);Text(event.title,Modifier.padding(top=6.dp),style=MaterialTheme.typography.titleMedium);Text(event.summary,Modifier.padding(top=5.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
            Row(Modifier.fillMaxWidth().padding(top=16.dp),horizontalArrangement=Arrangement.SpaceBetween){Text(event.start.format(DateTimeFormatter.ofPattern("MM月dd日 HH:mm")),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant);Text(if(event.id in state.joinedEvents)"已报名" else "${event.participants} 人参与",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.primary)}
        }}}
    }
}
@Composable
fun EventDetailPage(state:LabState,id:String,topInset:Dp) {
    val event=state.events.find{it.id==id} ?: return;val joined=id in state.joinedEvents
    LazyColumn(contentPadding=PaddingValues(start=22.dp,end=22.dp,top=topInset,bottom=45.dp),verticalArrangement=Arrangement.spacedBy(24.dp)) {
        item{ProductArt(event.art,Modifier.fillMaxWidth().height(210.dp).background(MaterialTheme.colorScheme.primaryContainer.copy(alpha=.35f),RoundedCornerShape(28.dp)))}
        item{Text(event.status,color=MaterialTheme.colorScheme.primary,style=MaterialTheme.typography.bodyMedium);Text(event.title,Modifier.padding(top=8.dp),style=MaterialTheme.typography.headlineLarge);Text(event.summary,Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodyLarge,color=MaterialTheme.colorScheme.onSurfaceVariant)}
        item{LabCard{DetailRow("开始时间",event.start.format(DateTimeFormatter.ofPattern("MM-dd HH:mm")));DetailRow("结束时间",event.end.format(DateTimeFormatter.ofPattern("MM-dd HH:mm")));DetailRow("集合地点",event.location);DetailRow("参与人数","${event.participants+if(joined)1 else 0} 人")}}
        item{Text("参与说明",style=MaterialTheme.typography.titleLarge);Text(when(event.id){"garden"->"基础建材将在现场提供，欢迎带上自己的装饰。活动开始前到出生点集合，按现场安排分组。";"explore"->"请准备护甲、食物与常用工具。远征中保持队伍联系，贵重物品建议提前存放。";"market"->"欢迎携带商品或展示项目。交易通过市场下单，由平台担保，交付后再确认。";"photo"->"在公共频道分享截图与地点，也可以邀请其他玩家一起参观。请尊重他人的建筑和领地。";else->"本次活动已经结束，可以关注活动中心的后续安排。"},Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodyLarge,color=MaterialTheme.colorScheme.onSurfaceVariant)}
        item{MotionButton({if(joined)state.joinedEvents.remove(id) else state.joinedEvents.add(id)},Modifier.fillMaxWidth().height(52.dp),enabled=event.status!="已结束"){Text(if(event.status=="已结束")"活动已结束" else if(joined)"已报名 · 取消报名" else "报名参加")}}
    }
}
