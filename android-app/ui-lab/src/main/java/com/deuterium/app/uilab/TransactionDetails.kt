package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.*
import kotlinx.coroutines.launch

@Composable
fun TimeRemainingCard(title:String,remainingMillis:Long,totalMillis:Long,caption:String,paused:Boolean=false,modifier:Modifier=Modifier) {
    val seconds=(remainingMillis.coerceAtLeast(0)+999)/1000
    val parts=buildList{if(seconds>=86400)add(seconds/86400 to "天");add(seconds/3600%24 to "时");add(seconds/60%60 to "分");add(seconds%60 to "秒")}
    val ink=if(remainingMillis<=0&&!paused)Color(0xFFFF9500) else MaterialTheme.colorScheme.primary
    Column(modifier.fillMaxWidth().padding(top=14.dp).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.30f),RoundedCornerShape(19.dp)).padding(16.dp)) {
        Row(verticalAlignment=Alignment.CenterVertically){Icon(if(paused)Icons.Outlined.PauseCircleOutline else Icons.Outlined.Schedule,null,Modifier.size(17.dp),tint=ink);Text(title,Modifier.padding(start=7.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
        Row(Modifier.fillMaxWidth().padding(top=13.dp,bottom=12.dp),verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.spacedBy(7.dp)){
            parts.forEachIndexed{index,(number,label)->
                Column(Modifier.weight(1f),horizontalAlignment=Alignment.CenterHorizontally){Text(number.toString().padStart(2,'0'),style=MaterialTheme.typography.headlineSmall.copy(fontSize=26.sp,fontWeight=FontWeight.Medium,fontFeatureSettings="tnum"));Text(label,style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
                if(index<parts.lastIndex)Text(":",Modifier.padding(bottom=15.dp),color=MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.45f),fontSize=20.sp)
            }
        }
        Box(Modifier.fillMaxWidth().height(3.dp).background(MaterialTheme.colorScheme.outlineVariant.copy(alpha=.5f),CircleShape)){
            Box(Modifier.fillMaxWidth((remainingMillis.toFloat()/totalMillis.coerceAtLeast(1)).coerceIn(0f,1f)).fillMaxHeight().background(if(paused)MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.4f) else ink.copy(alpha=.7f),CircleShape))
        }
        Text(caption,Modifier.padding(top=11.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
fun RefundDecisionDialog(approve:Boolean,amount:Long,onClose:()->Unit,onSubmit:suspend (String)->Boolean) {
    var reason by rememberSaveable{mutableStateOf("")};var error by remember{mutableStateOf<String?>(null)}
    val scope=rememberCoroutineScope();var busy by remember{mutableStateOf(false)}
    if(approve)IosDialog(onClose,{Text("同意退款？")},{Column{Text("确认后将 ${credit(amount)} 信用点原路退回付款方。");error?.let{Text(it,color=MaterialTheme.colorScheme.error)}}},{PlainButton({if(!busy)scope.launch{busy=true;if(!onSubmit(""))error="操作未完成或结果待确认，请查看交易状态";busy=false}},enabled=!busy){Text(if(busy)"正在处理…" else "确认退款")}}, {PlainButton(onClose){Text("取消")}})
    else IosSheet(onClose){Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(24.dp)){
        Text("拒绝退款",style=MaterialTheme.typography.headlineSmall)
        Text("请说明拒绝的具体原因，帮助对方了解处理结果。拒绝后恢复剩余确认时间，本订单不能再次申请退款。",Modifier.padding(top=12.dp,bottom=22.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        RefinedField(reason,{reason=it.take(500);error=null},Modifier.fillMaxWidth(),label={Text("拒绝理由（必填）")},minLines=4,maxLines=7)
        Text("${reason.length}/500",Modifier.align(Alignment.End).padding(top=7.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
        error?.let{Text(it,color=MaterialTheme.colorScheme.error,style=MaterialTheme.typography.bodySmall)}
        MotionButton({if(!busy)scope.launch{busy=true;if(!onSubmit(reason.trim()))error="操作未完成或状态已变化，请查看交易状态";busy=false}},Modifier.fillMaxWidth().padding(top=22.dp).height(52.dp),enabled=reason.trim().length>=2&&!busy){Text(if(busy)"正在提交…" else "提交拒绝理由")}
    }}
}
