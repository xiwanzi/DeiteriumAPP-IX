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
import androidx.compose.ui.platform.*
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.*
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.launch
import kotlinx.coroutines.delay

@Composable
fun OrdersPage(book:CommerceBook,topInset:Dp,onOrder:(String)->Unit) {
    var deleting by remember{mutableStateOf<CommerceOrder?>(null)}
    LaunchedEffect(Unit){book.network?.refreshOrders()}
    var channel by rememberSaveable{mutableIntStateOf(0)};var selling by rememberSaveable{mutableIntStateOf(0)};var filter by rememberSaveable{mutableStateOf("全部")}
    val orders=book.orders.filter{it.channel==(if(channel==0)OrderChannel.Official else OrderChannel.Market)&&(if(channel==1&&selling==1)it.seller==book.userName else it.buyer==book.userName)}.filter{
        when(filter){"进行中"->it.serverStatus !in setOf("CANCELLED","REFUNDED")&&it.stage !in listOf(OrderStage.Confirmed,OrderStage.Claimed)&&it.refund!=RefundState.Approved;"已完成"->it.stage in listOf(OrderStage.Confirmed,OrderStage.Claimed);"退款"->it.refund!=RefundState.None;else->true}
    }.sortedWith(compareByDescending<CommerceOrder>{it.createdAt}.thenByDescending{it.id})
    LazyColumn(contentPadding=PaddingValues(start=16.dp,end=16.dp,top=topInset,bottom=40.dp),verticalArrangement=Arrangement.spacedBy(18.dp)) {
        item { SegmentedControl(listOf("商城订单","市场订单"),channel,{channel=it;filter="全部"}) }
        if(channel==1)item { SegmentedControl(listOf("我买到的","我卖出的"),selling,{selling=it},Modifier.fillMaxWidth(.64f)) }
        item { Row(Modifier.horizontalScroll(rememberScrollState()),horizontalArrangement=Arrangement.spacedBy(7.dp)){listOf("全部","进行中","已完成","退款").forEach{ChoiceChip(filter==it,{filter=it},{Text(it)})}} }
        if(orders.isEmpty())item { Column(Modifier.fillMaxWidth().padding(vertical=70.dp),horizontalAlignment=Alignment.CenterHorizontally){Icon(Icons.Outlined.ReceiptLong,null,Modifier.size(44.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant);Text("暂无相关订单",Modifier.padding(top=18.dp),style=MaterialTheme.typography.titleMedium);Text("你的购买与交易记录会显示在这里",Modifier.padding(top=9.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)} }
        items(orders,key={it.id}){order->Surface(onClick={onOrder(order.id)},shape=RoundedCornerShape(24.dp),color=MaterialTheme.colorScheme.surface){Column(Modifier.padding(18.dp)) {
            Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){Text(if(selling==1&&channel==1)"买家 ${order.buyer}" else order.seller,Modifier.weight(1f),style=MaterialTheme.typography.titleMedium);Text(order.status,color=if(order.refund==RefundState.Approved)MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.primary,style=MaterialTheme.typography.bodyMedium)}
            order.lines.take(2).forEach{line->Row(Modifier.fillMaxWidth().padding(top=16.dp),verticalAlignment=Alignment.CenterVertically){OrderThumbnail(line,Modifier.size(66.dp,76.dp));Column(Modifier.weight(1f).padding(start=14.dp)){Text(line.title,style=MaterialTheme.typography.bodyLarge);Text("${credit(line.unitPrice)} × ${line.quantity}",Modifier.padding(top=5.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}}
            Row(Modifier.fillMaxWidth().padding(top=16.dp),verticalAlignment=Alignment.CenterVertically){Text(order.createdAt.format(DateTimeFormatter.ofPattern("MM月dd日 HH:mm")),Modifier.weight(1f),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant);Text("共 ${order.lines.sumOf{it.quantity}} 件  ${credit(order.amount)}",style=MaterialTheme.typography.titleMedium)}
            if(order.canHideRecord)DeleteRecordButton({deleting=order},Modifier.align(Alignment.End))
        }}}
    }
    deleting?.let{order->DeleteRecordDialog(order.lines.firstOrNull()?.title ?: "订单",{deleting=null}){book.network?.hideOrder(order.id)==true}}
}

@Composable
fun OrderProgress(order:CommerceOrder) {
    val labels=if(order.channel==OrderChannel.Official)listOf("已发货","未领取","已领取") else if(order.construction)listOf("待开工","施工中","已完成","已验收") else listOf("未发货","已发货","已确认")
    val active=when(order.stage){OrderStage.AwaitingShipment->0;OrderStage.Shipped,OrderStage.AwaitingClaim->1;OrderStage.Completed->2;else->labels.lastIndex}
    Column(Modifier.fillMaxWidth().padding(vertical=12.dp)) {
        Row(Modifier.fillMaxWidth().padding(horizontal=24.dp),verticalAlignment=Alignment.CenterVertically){repeat(labels.size){index->
            val reached=index<=active&&order.refund!=RefundState.Approved
            Box(Modifier.size(19.dp).background(if(reached)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant,CircleShape),contentAlignment=Alignment.Center){if(index<active)Icon(Icons.Outlined.Check,null,Modifier.size(12.dp),tint=Color.White) else if(index==active)Box(Modifier.size(7.dp).background(Color.White,CircleShape))}
            if(index<labels.lastIndex)Box(Modifier.weight(1f).height(2.dp).background(if(index<active)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant))
        }}
        Row(Modifier.fillMaxWidth().padding(top=10.dp)){labels.forEachIndexed{index,label->Text(label,Modifier.weight(1f),textAlign=androidx.compose.ui.text.style.TextAlign.Center,style=MaterialTheme.typography.bodySmall,color=if(index==active)MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.onSurfaceVariant)}}
    }
}

@Composable
fun OrderDetailPage(book:CommerceBook,id:String,topInset:Dp,onChat:(String)->Unit,onRefundStatus:(String)->Unit={},onDeleted:()->Unit={}) {
    LaunchedEffect(id){while(true){book.network?.refreshOrder(id);delay(5000)}}
    val scope=rememberCoroutineScope();var deleting by remember{mutableStateOf(false)}
    val order=book.order(id) ?: return
    if(order.isSaki){SakiOrderDetail(book,order,topInset,{onChat("AI 助手")},onDeleted);return}
    val buyer=order.buyer==book.userName
    val clipboard=LocalClipboardManager.current;val context=LocalContext.current
    var intervention by rememberSaveable{mutableStateOf(false)}
    var confirmReceipt by remember{mutableStateOf(false)};var refund by remember{mutableStateOf(false)};var tools by remember{mutableStateOf(false)}
    var complete by remember{mutableStateOf(false)};var simulatedSeller by remember{mutableStateOf(false)}
    var feedback by remember{mutableStateOf(false)};var withdraw by remember{mutableStateOf(false)}
    var claimHelp by remember{mutableStateOf(false)};var ship by remember{mutableStateOf(false)};var approveRefund by remember{mutableStateOf<Boolean?>(null)}
    val format=DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")
    fun copied(value:String){clipboard.setText(AnnotatedString(value));android.widget.Toast.makeText(context,"已复制",android.widget.Toast.LENGTH_SHORT).show()}
    Box(Modifier.fillMaxSize()) {
        LazyColumn(contentPadding=PaddingValues(start=20.dp,end=20.dp,top=topInset,bottom=145.dp),verticalArrangement=Arrangement.spacedBy(20.dp)) {
            item { Column(Modifier.fillMaxWidth().padding(vertical=10.dp),horizontalAlignment=Alignment.CenterHorizontally) {
                Icon(if(order.refund==RefundState.Approved)Icons.Outlined.Undo else if(order.stage in listOf(OrderStage.Confirmed,OrderStage.Claimed))Icons.Outlined.CheckCircle else if(order.construction)Icons.Outlined.Construction else Icons.Outlined.LocalShipping,null,Modifier.size(36.dp),tint=MaterialTheme.colorScheme.primary)
                Text(order.status,Modifier.padding(top=13.dp),style=MaterialTheme.typography.headlineSmall)
                Text(when{order.intervention!=null->interventionFundsText(order.intervention,order.held);order.refund==RefundState.Approved->"款项已原路退回钱包";order.refund==RefundState.Requested->"货款继续由平台保管，等待卖家处理";order.held->"货款由平台担保，${order.receiptLabel}后结算";order.stage==OrderStage.AwaitingClaim->"已发送至游戏内邮箱，等待领取";order.stage==OrderStage.Claimed->"物品已从游戏内邮箱领取";else->"交易完成，货款已结算给卖家"},Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant,textAlign=androidx.compose.ui.text.style.TextAlign.Center)
            } }
            item { LabCard { OrderProgress(order);OrderCountdown(order,book.nowMillis) } }
            item { LabCard {
                Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){if(buyer&&order.channel==OrderChannel.Official)LocalAvatar(order.sellerAvatarUri,order.seller,Modifier.size(44.dp)) else PlayerAvatar(if(buyer)order.seller else order.buyer);Column(Modifier.weight(1f).padding(start=13.dp)){Text(if(buyer)order.seller else order.buyer,style=MaterialTheme.typography.titleMedium);Text(if(buyer)"${if(order.channel==OrderChannel.Official)"官方商家" else "卖家"}" else "买家",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)};PlainButton({onChat(if(buyer)if(order.channel==OrderChannel.Official)"官方客服" else order.seller else order.buyer)}){Text(if(buyer)"联系卖家" else "联系买家")}}
                if(order.channel==OrderChannel.Market&&buyer){SettingsDivider();Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){Text("卖家 QQ",Modifier.weight(1f),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant);Text(order.sellerQQ,style=MaterialTheme.typography.bodyMedium);IconButton({copied(order.sellerQQ)}){Icon(Icons.Outlined.ContentCopy,"复制卖家QQ",Modifier.size(18.dp),tint=MaterialTheme.colorScheme.primary)}}}
                order.lines.forEach{line->Row(Modifier.fillMaxWidth().padding(top=18.dp),verticalAlignment=Alignment.CenterVertically){OrderThumbnail(line,Modifier.size(72.dp,86.dp));Column(Modifier.weight(1f).padding(start=14.dp)){Text(line.title,style=MaterialTheme.typography.titleMedium);Text(line.subtitle,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=2);Text("${credit(line.unitPrice)} × ${line.quantity}",Modifier.padding(top=7.dp),style=MaterialTheme.typography.bodyMedium)}}}
                if(order.productDiscount>0)DetailRow("商品优惠","−${credit(order.productDiscount)}")
                if(order.couponDiscount>0){DetailRow("优惠券","−${credit(order.couponDiscount)}");order.couponName?.let{Text(it,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
                DetailRow("实付金额","${credit(order.amount)} 信用点")
            } }
            item { LabCard {
                Text("交付信息",style=MaterialTheme.typography.titleMedium)
                if(order.construction){DetailRow("建筑项目",order.projectName);DetailRow("约定工期",durationHours(order.confirmationHours))}
                DetailRow("交付方式",order.method.label);DetailRow(if(order.method==DeliveryMethod.Mailbox)"领取账户" else "约定地点",order.location)
                if(order.channel==OrderChannel.Official)DetailRow("预计送达","支付完成后 1 分钟内")
                DetailRow(if(order.construction)"开始施工" else "发货时间",order.shippedAt?.format(format) ?: if(order.construction)"尚未开工" else "尚未发货")
                order.completedAt?.let{DetailRow("完成时间",it.format(format))}
                order.finishedAt?.let{DetailRow(if(order.channel==OrderChannel.Official)"领取时间" else "确认时间",it.format(format))}
            } }
            item { LabCard {
                Text("订单信息",style=MaterialTheme.typography.titleMedium)
                DetailRow("订单编号",order.id);PlainButton({copied(order.id)},Modifier.align(Alignment.End)){Text("复制订单编号")}
                DetailRow("交易时间",order.createdAt.format(format));DetailRow("付款方式","钱包余额")
                DetailRow("资金状态",if(order.intervention!=null)interventionFundsText(order.intervention,order.held) else if(order.refund==RefundState.Approved)"已退回" else if(order.held)"平台冻结 ${credit(order.amount)}" else if(order.channel==OrderChannel.Market)"已结算给卖家" else "已支付")
                if(order.refund!=RefundState.None){DetailRow("退款原因",order.refundReason);DetailRow("退款进度",order.refundMessage);order.refundedAt?.let{DetailRow("退款时间",it.format(format))}}
            } }
            if(order.refundAttempts>0)item{SettingsGroup{SettingsRow("退款详情",Icons.Outlined.ReceiptLong,detail=if(order.refund==RefundState.Requested)"待卖家处理" else "查看进度"){onRefundStatus(id)}}}
            order.intervention?.let{case->item{InterventionSummary(case){intervention=true}}}
            if(order.interventionCaseId!=null&&order.intervention==null)item{PlainButton({intervention=true}){Text("查看平台介入")}}
            if(order.canHideRecord)item{DeleteRecordButton({deleting=true},Modifier.fillMaxWidth())}
        }
        Surface(Modifier.align(Alignment.BottomCenter).fillMaxWidth(),color=MaterialTheme.colorScheme.surface) {
            Row(Modifier.navigationBarsPadding().padding(18.dp),horizontalArrangement=Arrangement.spacedBy(12.dp)) {
                if(order.pendingOperationId!=null)SecondaryButton({},Modifier.fillMaxWidth(),enabled=false){Text(order.status)} else if(buyer) {
                    SecondaryButton({if(order.canIntervene||order.intervention!=null)intervention=true else if(order.refund==RefundState.Requested)withdraw=true else refund=true},Modifier.weight(1f).height(51.dp),enabled=order.canIntervene||order.intervention!=null||order.canRefund||order.refund==RefundState.Requested){Text(if(order.intervention!=null)"查看平台介入" else if(order.canIntervene)"申请平台介入" else if(order.refund==RefundState.Requested)"撤回退款" else if(!order.canRefund&&order.refundAttempts>0)"退款机会已用" else "申请退款")}
                    if(order.channel==OrderChannel.Official)MotionButton({claimHelp=true},Modifier.weight(1f).height(51.dp),enabled=order.stage==OrderStage.AwaitingClaim&&order.refund!=RefundState.Approved){Text(if(order.stage==OrderStage.Claimed)"已领取" else "领取说明")}
                    else MotionButton({confirmReceipt=true},Modifier.weight(1f).height(51.dp),enabled=order.canReceive){Text(if(order.stage==OrderStage.Confirmed)if(order.construction)"已验收" else "已确认收货" else order.receiptLabel)}
                } else if(order.platformPending)SecondaryButton({intervention=true},Modifier.fillMaxWidth()){Text("查看平台介入进度")}
                else if(order.refund==RefundState.Requested) {
                    SecondaryButton({approveRefund=false},Modifier.weight(1f)){Text("拒绝退款")};MotionButton({approveRefund=true},Modifier.weight(1f)){Text("同意退款")}
                } else if(order.construction&&order.stage==OrderStage.Shipped&&order.refund!=RefundState.Approved)MotionButton({complete=true},Modifier.fillMaxWidth().height(51.dp)){Text("提交已完成")}
                else MotionButton({ship=true},Modifier.fillMaxWidth().height(51.dp),enabled=!order.platformPending&&order.stage==OrderStage.AwaitingShipment&&order.refund!=RefundState.Approved){Text(if(order.stage==OrderStage.AwaitingShipment)order.shipLabel else if(order.stage==OrderStage.Confirmed)"交易已完成" else if(order.construction)"等待买家验收" else "等待买家确认")}
            }
        }
    }
    if(deleting)DeleteRecordDialog(order.lines.firstOrNull()?.title ?: "订单",{deleting=false}){(book.network?.hideOrder(id)==true).also{if(it)onDeleted()}}
    if(intervention)OrderInterventionSheet(book,order){intervention=false}
    if(confirmReceipt)IosDialog({confirmReceipt=false},{Text(if(order.construction)"确认工程已验收？" else "确认已收到商品？")},{Text("请检查交付内容。确认后，平台将 ${credit(order.amount)} 信用点结算给 ${order.seller}，订单标记为已完成。")},{PlainButton({scope.launch{if(book.network?.orderAction(id,"confirm")==true)confirmReceipt=false}}){Text(order.receiptLabel)}}, {PlainButton({confirmReceipt=false}){Text("暂不确认")}})
    if(ship)IosDialog({ship=false},{Text(if(order.construction)"确认开始施工？" else "确认已经交付？")},{Text(if(order.construction)"将开始 ${durationHours(order.confirmationHours)} 的工期倒计时，请确认已经预留验收时间。" else "请先按约定交付物品。发货后开始72小时自动确认倒计时。")},{PlainButton({scope.launch{if(book.network?.orderAction(id,if(order.construction)"start-work" else "ship")==true)ship=false}}){Text(order.shipLabel)}}, {PlainButton({ship=false}){Text("取消")}})
    if(refund)RefundSheet(order,{refund=false}){reason->val accepted=book.network?.orderAction(id,"request-refund",reason)==true;if(accepted){refund=false;feedback=true};accepted}
    if(feedback)ActionFeedback(if(order.refund==RefundState.Approved)"退款成功" else "申请已提交",if(order.refund==RefundState.Approved)"款项已退回钱包" else "已通知卖家，正在等待处理"){feedback=false;onRefundStatus(id)}
    if(withdraw)IosDialog({withdraw=false},{Text("撤回退款申请？")},{Text("撤回后会继续自动确认倒计时，并且无法再次申请退款。请先与卖家沟通。")},{PlainButton({scope.launch{if(book.network?.orderAction(id,"withdraw")==true){withdraw=false;onRefundStatus(id)}}}){Text("确认撤回")}}, {PlainButton({withdraw=false}){Text("保留申请")}})
    approveRefund?.let{approve->RefundDecisionDialog(approve,order.amount,{approveRefund=null;simulatedSeller=false}){reason->(book.network?.orderAction(id,"resolve",reason,approve)==true).also{if(it){approveRefund=null;simulatedSeller=false}}}}
    if(complete)IosDialog({complete=false},{Text("确认工程已完成？")},{Text("提交后将通知买家验收。现有工期倒计时保持不变，请确认约定内容已经完成。")},{PlainButton({scope.launch{if(book.network?.orderAction(id,"complete-work")==true)complete=false}}){Text("提交完成")}}, {PlainButton({complete=false}){Text("取消")}})

    LaunchedEffect(claimHelp){if(claimHelp)book.network?.refreshMailbox(id)}
    if(claimHelp)IosDialog({claimHelp=false},{Text("从游戏内邮箱领取")},{Text("登录 Deuterium IX，打开游戏内邮箱，找到订单 ${order.id.takeLast(8)} 并领取。领取成功后，此订单会更新为已领取，无法再退款。")},{PlainButton({claimHelp=false}){Text("知道了")}})

}

@Composable
private fun RefundSheet(order:CommerceOrder,onClose:()->Unit,onSubmit:suspend (String)->Boolean) {
    val scope=rememberCoroutineScope();var busy by remember{mutableStateOf(false)};var submitError by remember{mutableStateOf<String?>(null)}
    var selected by remember{mutableStateOf("不再需要")};var note by remember{mutableStateOf("")};var confirm by remember{mutableStateOf(false)}
    IosSheet(onClose){Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).padding(24.dp)){
        Text("申请退款",style=MaterialTheme.typography.headlineSmall);Text("${credit(order.amount)} 信用点",Modifier.padding(top=9.dp),style=MaterialTheme.typography.titleLarge)
        Text(if(order.channel==OrderChannel.Official)"未领取的物品将从邮箱撤回，款项原路退回钱包。" else if(order.stage==OrderStage.AwaitingShipment)"卖家尚未${if(order.construction)"开工" else "发货"}，确认提交后立即退款。" else "${if(order.construction)"开工" else "发货"}后只能申请一次退款。请先与卖家充分沟通，申请后自动确认计时会暂停。卖家拒绝或你撤回后，不能再申请。",Modifier.padding(vertical=18.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        listOf("不再需要","无法按约定交付",if(order.construction)"工程内容不符" else "商品信息不符","其他原因").forEach{reason->Row(Modifier.fillMaxWidth().clickable{selected=reason}.padding(vertical=12.dp),verticalAlignment=Alignment.CenterVertically){Text(reason,Modifier.weight(1f));if(selected==reason)Icon(Icons.Outlined.Check,null,tint=MaterialTheme.colorScheme.primary)}}
        RefinedField(note,{note=it.take(200)},Modifier.fillMaxWidth().padding(top=10.dp),label={Text("补充说明（选填）")},maxLines=3)
        submitError?.let{Text(it,color=MaterialTheme.colorScheme.error)}
        MotionButton({confirm=true},Modifier.fillMaxWidth().padding(top=24.dp).height(52.dp),enabled=order.canRefund){Text("提交退款申请")}
    }}
    if(confirm)IosDialog({confirm=false},{Text(if(order.stage in listOf(OrderStage.Shipped,OrderStage.Completed))"确认使用唯一退款机会？" else "确认申请退款？")},{Text(if(order.stage in listOf(OrderStage.Shipped,OrderStage.Completed))"请确认已经与卖家沟通。提交后将通知卖家，且本订单不能再次申请退款。" else "确认后将取消本订单，并把 ${credit(order.amount)} 信用点原路退回钱包。")},{PlainButton({if(!busy)scope.launch{busy=true;if(onSubmit(selected+if(note.isBlank())"" else " · $note"))confirm=false else {submitError="操作未完成或结果待确认，请查看交易状态";confirm=false};busy=false}},enabled=order.canRefund&&!busy){Text("确认提交")}}, {PlainButton({confirm=false}){Text("再想想")}})
}
