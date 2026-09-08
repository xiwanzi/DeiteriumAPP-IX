package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import java.util.UUID
import kotlinx.coroutines.launch

fun interventionFundsText(case:InterventionCase,held:Boolean):String=when {
    case.serverResolution!=null->case.serverResolution.ifBlank{if(case.pending)"等待平台处理，状态以服务器记录为准" else case.status.label}
    case.pending&&held->"平台审核中，担保资金冻结，计时已暂停"
    case.pending->"交易已结算，等待平台人工审核"
    case.refunded>0||case.paidToPayee>0->"已退款 ${credit(case.refunded)} · 已结算 ${credit(case.paidToPayee)}"
    held->"按平台判决继续履约，担保与剩余计时已恢复"
    else->"原交易结算保持不变，${case.resultText}"
}

@Composable fun InterventionSummary(case:InterventionCase,onOpen:()->Unit) {
    LabCard {
        Text("平台介入",style=MaterialTheme.typography.titleMedium)
        Text(case.resultText,Modifier.padding(top=9.dp),style=MaterialTheme.typography.bodyLarge,color=MaterialTheme.colorScheme.primary)
        Text(if(case.pending)"已保留交易与退款快照，等待平台审核。" else case.decisionReason,Modifier.padding(top=7.dp),style=MaterialTheme.typography.bodyMedium)
        PlainButton(onOpen,Modifier.fillMaxWidth()){Text(if(case.pending)"查看介入进度" else "查看平台判决")}
    }
}

@Composable fun OrderInterventionSheet(book:CommerceBook,order:CommerceOrder,onClose:()->Unit) {
    if(order.interventionCaseId!=null&&order.intervention==null){LaunchedEffect(order.interventionCaseId){book.interventions?.refresh(order.interventionCaseId)};IosSheet(onClose){Text(book.interventions?.error ?: "正在读取平台案件…",Modifier.padding(24.dp))};return}
    val snapshot=remember(order.id){InterventionSnapshot(order.id,order.lines.joinToString("、"){it.title},order.buyer,order.seller,order.amount,order.location,
        "${order.method.label} · ${order.projectName} · ${order.confirmationHours} 小时；"+order.lines.joinToString("；"){"${it.title} × ${it.quantity}：${it.subtitle}"},order.createdAt,order.refundReason,order.rejectionReason,order.status,LocalDateTime.now())}
    InterventionSheet(order.intervention,snapshot,order.held,"ORDER",book.interventions?.error,onClose,{key,form->book.interventions?.submit("ORDER",order.id,order.version,key,form)==true})
}
@Composable fun CommissionInterventionSheet(book:CommissionBook,entry:Commission,onClose:()->Unit) {
    if(entry.interventionCaseId!=null&&entry.intervention==null){LaunchedEffect(entry.interventionCaseId){book.interventions?.refresh(entry.interventionCaseId)};IosSheet(onClose){Text(book.interventions?.error ?: "正在读取平台案件…",Modifier.padding(24.dp))};return}
    val snapshot=remember(entry.id){InterventionSnapshot(entry.id,entry.draft.title,entry.owner,entry.worker ?: "",entry.draft.reward,entry.draft.location,
        "${entry.draft.description}；履约 ${entry.draft.workHours} 小时，完成后 72 小时验收；完成说明：${entry.completionNote}",entry.createdAt,entry.refundReason,entry.rejectionReason,entry.status,LocalDateTime.now())}
    InterventionSheet(entry.intervention,snapshot,entry.held,"COMMISSION",book.interventions?.error,onClose,{key,form->book.interventions?.submit("COMMISSION",entry.id,entry.version,key,form)==true})
}

@Composable private fun InterventionSheet(case:InterventionCase?,preview:InterventionSnapshot,held:Boolean,evidenceBusinessType:String,error:String?,onClose:()->Unit,
    onSubmit:suspend (String,InterventionForm)->Boolean) {
    val scope=rememberCoroutineScope()
    var reason by rememberSaveable{mutableStateOf(InterventionReason.RefundDisagreement)}
    var wish by rememberSaveable{mutableStateOf(InterventionWish.FullRefund)}
    var description by rememberSaveable{mutableStateOf("")};var amount by rememberSaveable{mutableStateOf("")}
    var photos by rememberSaveable{mutableStateOf(arrayListOf<String>())};var busy by remember{mutableStateOf(false)}
    var confirm by rememberSaveable{mutableStateOf(false)}
    var showSnapshot by rememberSaveable{mutableStateOf(false)}
    val requestKey=rememberSaveable{UUID.randomUUID().toString()}
    val cents=runCatching{amount.toBigDecimal().movePointRight(2).longValueExact()}.getOrNull()
    val form=InterventionForm(reason,description,wish,cents,photos.toList())
    val snapshot=case?.snapshot ?: preview
    val scroll=rememberScrollState()
    LaunchedEffect(case?.id,confirm){scroll.scrollTo(0)}
    IosSheet({if(!busy)onClose()}) {
        Column(Modifier.fillMaxWidth().verticalScroll(scroll).padding(start=24.dp,end=24.dp,bottom=26.dp),verticalArrangement=Arrangement.spacedBy(14.dp)) {
            Text(if(case==null)if(confirm)"确认介入申请" else "申请平台介入" else "平台介入详情",style=MaterialTheme.typography.headlineSmall)
            if(case==null) {
                Text("${snapshot.title} · ${credit(snapshot.amount)} 信用点",style=MaterialTheme.typography.titleMedium)
                Text(if(held)"提交后暂停确认计时，款项继续冻结，最终按平台判决处理。" else "交易已经结算，平台将人工审核争议。",style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
                if(confirm) {
                    DetailRow("介入原因",reason.label);Text(description);DetailRow("你的诉求",wish.label)
                    if(wish==InterventionWish.PartialRefund)DetailRow("申请退回","${credit(cents ?: 0)} 信用点")
                    DetailRow("凭证图片","${photos.size} 张")
                    Text("申请将附上原交易、退款说明及拒绝理由。",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    MotionButton({if(!busy)scope.launch{busy=true;if(onSubmit(requestKey,form))confirm=false;busy=false}},Modifier.fillMaxWidth(),enabled=!busy&&form.valid(snapshot.amount)){Text("确认提交介入申请")}
                    PlainButton({confirm=false},Modifier.fillMaxWidth()){Text("返回修改")}
                } else {
                    Text("介入原因",style=MaterialTheme.typography.titleMedium)
                    InterventionChoices(InterventionReason.entries,reason,{reason=it}){it.label}
                    RefinedField(description,{description=it.take(3000)},Modifier.fillMaxWidth(),label={Text("事实与原因（10–3000 字）")},minLines=4,maxLines=7)
                    Text("你的诉求",style=MaterialTheme.typography.titleMedium)
                    InterventionChoices(InterventionWish.entries,wish,{wish=it}){it.label}
                    if(wish==InterventionWish.PartialRefund)RefinedField(amount,{amount=it.take(16)},Modifier.fillMaxWidth(),label={Text("申请退款金额（信用点，最多两位小数）")})
                    ListingPhotoEditor(photos,{photos=ArrayList(it)},{busy=it},maxImages=10,evidence=true,businessType=evidenceBusinessType,businessRef=snapshot.transactionId)
                    Text("案件状态与资金结果以平台处理结果为准。",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    MotionButton({confirm=true},Modifier.fillMaxWidth(),enabled=!busy&&form.valid(snapshot.amount)){Text(if(busy)"凭证读取中…" else "继续提交")}
                }
                PlainButton({showSnapshot=!showSnapshot},Modifier.fillMaxWidth()){Text(if(showSnapshot)"收起交易快照" else "查看自动附带的交易快照")}
                if(showSnapshot)InterventionSnapshotView(snapshot)
            } else {
                Text(case.resultText,style=MaterialTheme.typography.titleLarge,color=MaterialTheme.colorScheme.primary)
                Text(interventionFundsText(case,held),style=MaterialTheme.typography.bodyMedium)
                DetailRow("案件编号",case.id)
                DetailRow("提交时间",case.snapshot.capturedAt.format(DateTimeFormatter.ofPattern("MM-dd HH:mm:ss")))
                DetailRow("最近更新",case.updatedAt.format(DateTimeFormatter.ofPattern("MM-dd HH:mm:ss")))
                if(!case.serverResolution.isNullOrBlank()){Text("平台处理结果",style=MaterialTheme.typography.titleMedium);Text(case.serverResolution)}
                if(case.decision!=null){
                    Text("判决理由",style=MaterialTheme.typography.titleMedium);Text(case.decisionReason)
                    DetailRow("本次实际退款","${credit(case.refunded)} 信用点");DetailRow("本次实际结算","${credit(case.paidToPayee)} 信用点")
                }
                SettingsDivider();Text("申请材料",style=MaterialTheme.typography.titleMedium)
                DetailRow("介入原因",case.reasonDisplay ?: case.form.reason.label);Text(case.form.description);DetailRow("申请诉求",case.form.wish.label)
                case.form.requestedRefund?.let{DetailRow("申请退款金额","${credit(it)} 信用点")}
                if(case.form.evidence.isNotEmpty())Row(Modifier.horizontalScroll(rememberScrollState()),horizontalArrangement=Arrangement.spacedBy(10.dp)){case.form.evidence.forEach{uri->ListingImage(MarketListing("evidence","凭证","","","",1,1,"","",emptySet(),"",imageUri=uri),Modifier.size(130.dp))}}
                PlainButton({showSnapshot=!showSnapshot},Modifier.fillMaxWidth()){Text(if(showSnapshot)"收起原交易快照" else "查看原交易与退款快照")}
                if(showSnapshot)InterventionSnapshotView(snapshot)

                Text("案件状态与资金结果以平台处理结果为准。",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
            }
            error?.let{Text(it,color=MaterialTheme.colorScheme.error,style=MaterialTheme.typography.bodySmall)}
        }
    }
}

@Composable private fun <T> InterventionChoices(values:List<T>,selected:T,onSelect:(T)->Unit,label:(T)->String) {
    Column(verticalArrangement=Arrangement.spacedBy(5.dp)){values.forEach{value->ChoiceChip(selected==value,{onSelect(value)},{Text(label(value))})}}
}
@Composable private fun InterventionSnapshotView(s:InterventionSnapshot) {
    Text("原交易快照 · 不可修改",style=MaterialTheme.typography.titleMedium)
    DetailRow("交易编号",s.transactionId);DetailRow("交易内容",s.title);DetailRow("付款方",s.payer);DetailRow("收款方",s.payee)
    DetailRow("实付金额","${credit(s.amount)} 信用点");DetailRow("约定地点",s.location);DetailRow("交易时间",s.transactionCreatedAt.toString())
    Text(s.terms,style=MaterialTheme.typography.bodyMedium);DetailRow("原退款原因",s.refundReason);DetailRow("拒绝理由",s.rejectionReason);DetailRow("提交时状态",s.transactionStatus)
}
