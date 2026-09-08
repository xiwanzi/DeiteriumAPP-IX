package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.shape.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.*
import androidx.compose.ui.window.*
import java.time.*
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun ActionFeedback(title:String,detail:String,onDone:()->Unit) {
    var done by remember{mutableStateOf(false)};val latest by rememberUpdatedState(onDone)
    LaunchedEffect(done){if(done){delay(600);latest()}}
    IosOverlayHost(onDismissRequest={if(done)onDone()}){
        SoftGlassSurface(Modifier.fillMaxWidth(.88f),radius=30.dp){Column(Modifier.padding(26.dp),horizontalAlignment=Alignment.CenterHorizontally){PaymentConfirmation(Modifier.size(130.dp)){done=true};Text(title,style=MaterialTheme.typography.headlineSmall);Text(detail,Modifier.padding(top=10.dp,bottom=14.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant,textAlign=androidx.compose.ui.text.style.TextAlign.Center)}}
    }
}

fun countdownText(millis:Long):String {
    val seconds=(millis.coerceAtLeast(0)+999)/1000;val days=seconds/86400;val hours=seconds/3600%24;val minutes=seconds/60%60;val s=seconds%60
    return (if(days>0)"${days}天 " else "")+"%02d:%02d:%02d".format(hours,minutes,s)
}
@Composable
fun OrderCountdown(order:CommerceOrder,nowMillis:Long) {
    if(order.stage !in listOf(OrderStage.Shipped,OrderStage.Completed)||order.refund==RefundState.Approved)return
    val paused=order.platformPending||order.refund==RefundState.Requested
    val remaining=(if(paused)order.pausedMillis else order.deadlineMillis?.minus(nowMillis)) ?: return
    TimeRemainingCard(if(paused)"自动确认已暂停" else if(order.construction)"距离自动验收" else "距离自动确认收货",remaining,order.confirmationHours*3600000L,
        if(order.platformPending)"等待平台裁决，双方暂不能自行确认结算。" else if(paused)"退款处理后继续剩余时间。" else if(order.construction)if(order.stage==OrderStage.Completed)"工程已完成，请在剩余时间内验收。" else "总工期包含验收预留，完成提交不重置计时。" else "请及时检查物品，有疑问先联系卖家。",paused)
}

@Composable
fun RefundStatusPage(book:CommerceBook,id:String,topInset:Dp,onChat:(String)->Unit,onOrder:(String)->Unit) {
    val scope=rememberCoroutineScope()
    LaunchedEffect(id){while(true){book.network?.refreshOrder(id);delay(5000)}}
    val order=book.order(id) ?: return;val buyer=order.buyer==book.userName
    var intervention by androidx.compose.runtime.saveable.rememberSaveable{mutableStateOf(false)}
    var decision by remember{mutableStateOf<Boolean?>(null)};var withdraw by remember{mutableStateOf(false)};var preview by remember{mutableStateOf(false)}
    val format=DateTimeFormatter.ofPattern("MM-dd HH:mm:ss")
    val state=order.intervention?.resultText ?: when(order.refund){RefundState.Requested->"等待卖家处理";RefundState.Approved->"退款已完成";RefundState.Rejected->"卖家未同意退款";else->"退款申请已撤回"}
    Box(Modifier.fillMaxSize()) {
        LazyColumn(contentPadding=PaddingValues(start=20.dp,end=20.dp,top=topInset,bottom=150.dp),verticalArrangement=Arrangement.spacedBy(20.dp)) {
            item{Column(Modifier.fillMaxWidth().padding(vertical=12.dp),horizontalAlignment=Alignment.CenterHorizontally){Icon(if(order.refund==RefundState.Approved)Icons.Outlined.CheckCircle else Icons.Outlined.Forum,null,Modifier.size(40.dp),tint=MaterialTheme.colorScheme.primary);Text(state,Modifier.padding(top=14.dp),style=MaterialTheme.typography.headlineSmall);Text("${credit(order.amount)} 信用点",Modifier.padding(top=10.dp),style=MaterialTheme.typography.titleLarge)}}
            item{LabCard{
                Text("退款进度",style=MaterialTheme.typography.titleMedium)
                RefundTimelineRow("申请已提交",order.refundRequestedAt?.format(format) ?: "—",true)
                val notified=book.notices.firstOrNull{it.orderId==id&&it.recipient==order.seller&&it.kind.startsWith("refund")}
                RefundTimelineRow("已通知卖家 ${order.seller}",notified?.at?.format(format) ?: "通知记录待更新",notified!=null)
                RefundTimelineRow(if(order.refund==RefundState.Requested)"卖家处理中" else "处理结果",order.refundResolvedAt?.format(format) ?: "请保持沟通，等待卖家回复",order.refundResolvedAt!=null)
                if(order.refund==RefundState.Approved)RefundTimelineRow("已退回钱包",order.refundedAt?.format(format) ?: "",true)
            }}
            order.intervention?.let{case->item{InterventionSummary(case){intervention=true}}}
            item{LabCard{Text("申请信息",style=MaterialTheme.typography.titleMedium);DetailRow("退款原因",order.refundReason);DetailRow("当前结果",if(order.refund==RefundState.Rejected)"卖家未同意退款" else order.refundMessage);if(order.rejectionReason.isNotBlank())DetailRow("拒绝理由",order.rejectionReason);DetailRow("申请次数","${order.refundAttempts} 次");DetailRow("订单编号",order.id);PlainButton({onOrder(id)},Modifier.fillMaxWidth()){Text("查看订单详情")}}}
            if(order.stage in listOf(OrderStage.Shipped,OrderStage.Completed)&&order.refund!=RefundState.Approved)item{LabCard{Text("本订单不能再次申请退款",style=MaterialTheme.typography.titleMedium);Text("${if(order.construction)"开工" else "发货"}后仅可申请一次退款。拒绝后可申请平台介入；撤回或拒绝均不会恢复退款次数。",Modifier.padding(top=9.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant);OrderCountdown(order,book.nowMillis)}}
        }
        Surface(Modifier.align(Alignment.BottomCenter).fillMaxWidth(),color=MaterialTheme.colorScheme.surface){Row(Modifier.navigationBarsPadding().padding(18.dp),horizontalArrangement=Arrangement.spacedBy(12.dp)){
            if(!buyer&&order.refund==RefundState.Requested){SecondaryButton({decision=false},Modifier.weight(1f)){Text("拒绝退款")};MotionButton({decision=true},Modifier.weight(1f)){Text("同意退款")}}
            else {SecondaryButton({onChat(if(buyer)order.seller else order.buyer)},Modifier.weight(1f)){Text(if(buyer)"联系卖家" else "联系买家")};if(buyer&&order.refund==RefundState.Requested)MotionButton({withdraw=true},Modifier.weight(1f)){Text("撤回申请")}else if(order.intervention!=null||buyer&&order.canIntervene)MotionButton({intervention=true},Modifier.weight(1f)){Text(if(order.intervention!=null)"查看平台介入" else "申请平台介入")}else MotionButton({onOrder(id)},Modifier.weight(1f)){Text("返回订单")}}
        }}
    }
    if(intervention)OrderInterventionSheet(book,order){intervention=false}
    decision?.let{approve->RefundDecisionDialog(approve,order.amount,{decision=null}){reason->(book.network?.orderAction(id,"resolve",reason,approve)==true).also{if(it)decision=null}}}
    if(withdraw)IosDialog({withdraw=false},{Text("确认撤回？")},{Text("撤回后无法再次申请，自动确认倒计时将继续。")},{PlainButton({scope.launch{if(book.network?.orderAction(id,"withdraw")==true)withdraw=false}}){Text("确认撤回")}}, {PlainButton({withdraw=false}){Text("保留申请")}})
}
@Composable
private fun RefundTimelineRow(title:String,detail:String,done:Boolean){Row(Modifier.fillMaxWidth().padding(top=20.dp),verticalAlignment=Alignment.Top){Box(Modifier.padding(top=5.dp).size(10.dp).background(if(done)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outlineVariant,CircleShape));Column(Modifier.padding(start=13.dp)){Text(title,style=MaterialTheme.typography.bodyLarge);Text(detail,Modifier.padding(top=4.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}}

@Composable
fun TradeNoticesPage(book:CommerceBook,topInset:Dp,allowed:Boolean,onEnable:()->Unit,onOpen:(String)->Unit,onRead:(String)->Unit=book::markNoticeRead) {
    val notices=book.notices.filter{it.recipient==book.userName}
    LazyColumn(contentPadding=PaddingValues(start=18.dp,end=18.dp,top=topInset,bottom=40.dp),verticalArrangement=Arrangement.spacedBy(12.dp)) {
        if(!allowed)item{LabCard{Text("开启系统交易提醒",style=MaterialTheme.typography.titleMedium);Text("退款申请、卖家回复和到账消息也会显示在通知栏。",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant);PlainButton(onEnable){Text("开启通知")}}}
        if(notices.isEmpty())item{Text("暂无交易通知",Modifier.fillMaxWidth().padding(vertical=60.dp),textAlign=androidx.compose.ui.text.style.TextAlign.Center,color=MaterialTheme.colorScheme.onSurfaceVariant)}
        items(notices,key={it.id}){notice->Surface(onClick={onRead(notice.id);onOpen(notice.route)},shape=RoundedCornerShape(20.dp),color=MaterialTheme.colorScheme.surface){Column(Modifier.padding(17.dp)){Row(verticalAlignment=Alignment.CenterVertically){if(!notice.read)Box(Modifier.padding(end=8.dp).size(7.dp).background(MaterialTheme.colorScheme.primary,CircleShape));Text(notice.title,style=MaterialTheme.typography.titleMedium)};Text(notice.body,Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant);Text(notice.at.format(DateTimeFormatter.ofPattern("MM-dd HH:mm")),Modifier.padding(top=9.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}}
    }
}
