package com.deuterium.app.uilab

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.*
import androidx.compose.foundation.shape.*
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.*
import java.time.format.DateTimeFormatter
import java.util.UUID
import kotlinx.coroutines.launch
import kotlinx.coroutines.delay

@Composable
fun CommissionImage(draft:CommissionDraft,modifier:Modifier=Modifier) {
    if(draft.coverUri!=null)ListingImage(MarketListing("commission-cover",draft.title,"","","",1,1,"","",emptySet(),"",imageUri=draft.coverUri),modifier)
    else ProductArt(draft.artKey.ifBlank{"garden"},modifier.background(MaterialTheme.colorScheme.primaryContainer.copy(alpha=.3f)))
}
@Composable
private fun UrgencyBadge(urgency:Urgency) {
    val color=when(urgency){Urgency.Normal->MaterialTheme.colorScheme.onSurfaceVariant;Urgency.Soon->Color(0xFFC47B08);Urgency.Urgent->Color(0xFFE75045)}
    Text(urgency.label,Modifier.background(color.copy(alpha=.10f),RoundedCornerShape(6.dp)).padding(horizontal=7.dp,vertical=3.dp),style=MaterialTheme.typography.labelSmall,color=color)
}
@Composable
fun CommissionHallPage(state:LabState,topInset:Dp,onOpen:(String)->Unit,mine:Boolean=false) {
    LaunchedEffect(Unit){state.commissions.network?.refresh()}
    var deleting by remember{mutableStateOf<Commission?>(null)}
    var role by rememberSaveable{mutableIntStateOf(0)};var filter by rememberSaveable{mutableStateOf("全部")}
    val entries=state.commissions.entries.filter{(mine||it.visibleInHall)&&(!mine||if(role==0)it.owner==state.userName else it.worker==state.userName)&&(when(filter){"待接取"->it.stage==CommissionStage.Open;"进行中"->it.stage==CommissionStage.Active;"待确认"->it.stage==CommissionStage.Completed;"已结束"->it.stage in setOf(CommissionStage.Confirmed,CommissionStage.Cancelled)&&!it.held;else->true})}.sortedWith(compareByDescending<Commission>{it.createdAt}.thenByDescending{it.id})
    LazyColumn(contentPadding=PaddingValues(start=18.dp,end=18.dp,top=topInset,bottom=45.dp),verticalArrangement=Arrangement.spacedBy(15.dp)) {
        item{Text(if(mine)"我的委托" else "互相搭把手",style=MaterialTheme.typography.headlineLarge);Text(if(mine)"查看履约进度，及时沟通与确认。" else LocalAppUpdates.current?.resourceStrings?.get("commissions.subtitle") ?: "报酬已预付，完成后安心结算。",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)}
        if(mine)item{SegmentedControl(listOf("我发布的","我接取的"),role,{role=it;filter="全部"})}
        item{Row(Modifier.horizontalScroll(rememberScrollState()),horizontalArrangement=Arrangement.spacedBy(7.dp)){(if(mine)listOf("全部","待接取","进行中","待确认","已结束") else listOf("全部","待接取")).forEach{label->ChoiceChip(filter==label,{filter=if(filter==label)"全部" else label},{Text(label)})}}}
        state.commissions.network?.error?.let{error->item{Text(error,color=MaterialTheme.colorScheme.error)}}
        if(entries.isEmpty())item{Column(Modifier.fillMaxWidth().padding(vertical=65.dp),horizontalAlignment=Alignment.CenterHorizontally){Icon(Icons.Outlined.Assignment,null,Modifier.size(42.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant);Text("暂时没有相关委托",Modifier.padding(top=17.dp),style=MaterialTheme.typography.titleMedium)}}
        items(entries,key={it.id}){entry->Surface(onClick={onOpen(entry.id)},shape=RoundedCornerShape(23.dp),color=MaterialTheme.colorScheme.surface){Column(Modifier.padding(16.dp)){
            Row{CommissionImage(entry.draft,Modifier.size(91.dp).clip(RoundedCornerShape(17.dp)));Column(Modifier.weight(1f).padding(start=14.dp)){
                Row(verticalAlignment=Alignment.CenterVertically){Text(entry.status,Modifier.weight(1f),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.primary);UrgencyBadge(entry.draft.urgency)}
                Text(entry.draft.title,Modifier.padding(top=9.dp),style=MaterialTheme.typography.titleMedium,maxLines=2,overflow=TextOverflow.Ellipsis)
                Text(entry.draft.location,Modifier.padding(top=5.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=1,overflow=TextOverflow.Ellipsis)
            }}
            Row(Modifier.fillMaxWidth().padding(top=16.dp),verticalAlignment=Alignment.CenterVertically){Text("${credit(entry.draft.reward)}",style=MaterialTheme.typography.titleLarge,color=MaterialTheme.colorScheme.primary);Text(" 信用点",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant);Spacer(Modifier.weight(1f));Text(if(entry.stage==CommissionStage.Open&&entry.held)"已预付 · ${durationHours(entry.draft.workHours)}" else entry.worker?.let{"接取者 $it"} ?: entry.owner,style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}
            if(mine&&entry.canHideRecord)DeleteRecordButton({deleting=entry},Modifier.align(Alignment.End))
        }}}
    }
    deleting?.let{entry->DeleteRecordDialog(entry.draft.title,{deleting=null}){state.commissions.network?.hide(entry.id)==true}}
}

@Composable
fun PublishCommissionPage(state:LabState,topInset:Dp,onPublished:(String)->Unit) {
    var title by rememberSaveable{mutableStateOf("")};var description by rememberSaveable{mutableStateOf("")};var location by rememberSaveable{mutableStateOf("")}
    var reward by rememberSaveable{mutableStateOf("")};var duration by rememberSaveable{mutableStateOf("3")};var unit by rememberSaveable{mutableIntStateOf(1)};var urgency by rememberSaveable{mutableStateOf(Urgency.Normal)}
    var photos by rememberSaveable{mutableStateOf(emptyList<String>())};var copying by remember{mutableStateOf(false)};var error by remember{mutableStateOf<String?>(null)}
    var pending by remember{mutableStateOf(false)};var result by remember{mutableStateOf<String?>(null)};val key=rememberSaveable{UUID.randomUUID().toString()}
    val cents=runCatching{reward.toBigDecimal().movePointRight(2).longValueExact()}.getOrNull() ?: 0
    val draft=CommissionDraft(title,description,location,cents,(duration.toIntOrNull() ?: 0)*(if(unit==1)24 else 1),urgency,photos.firstOrNull())
    Box(Modifier.fillMaxSize()){
        LazyColumn(contentPadding=PaddingValues(start=16.dp,end=16.dp,top=topInset,bottom=150.dp),verticalArrangement=Arrangement.spacedBy(18.dp)){
            item{LabCard{ListingPhotoEditor(photos,{photos=it},{copying=it},maxImages=1)}}
            item{LabCard{RefinedField(title,{title=it.take(60)},Modifier.fillMaxWidth(),label={Text("委托标题（必填）")},placeholder={Text("需要大家帮你做什么？")},maxLines=2)
                RefinedField(description,{description=it.take(2000)},Modifier.fillMaxWidth().padding(top=17.dp),label={Text("具体要求（必填）")},placeholder={Text("工作内容、数量、完成标准与注意事项")},minLines=4,maxLines=8)
                RefinedField(location,{location=it.take(120)},Modifier.fillMaxWidth().padding(top=17.dp),label={Text("地点或交付方式（必填）")},maxLines=3)}}
            item{LabCard{Text("紧急程度",style=MaterialTheme.typography.titleMedium);Row(Modifier.padding(top=13.dp),horizontalArrangement=Arrangement.spacedBy(9.dp)){Urgency.entries.forEach{item->ChoiceChip(urgency==item,{urgency=item},{Text(item.label)})}}
                RefinedField(duration,{duration=it.filter(Char::isDigit).take(4)},Modifier.fillMaxWidth().padding(top=20.dp),label={Text("履约时限")},keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Number),singleLine=true)
                SegmentedControl(listOf("小时","天"),unit,{unit=it},Modifier.padding(top=12.dp))
                Text("从接取时开始计时。提交完成后，另有 72 小时供你确认；确认或到期后结算报酬。",Modifier.padding(top=13.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
            item{LabCard{RefinedField(reward,{reward=it.take(11)},Modifier.fillMaxWidth(),label={Text("预付报酬（信用点）")},keyboardOptions=KeyboardOptions(keyboardType=KeyboardType.Decimal),singleLine=true)
                Text("可用余额 ${credit(state.balance)}",Modifier.padding(top=10.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                Text("发布时预付全部报酬，由平台担保。尚未接取可取消并退款；接取后需与对方协商。",Modifier.padding(top=13.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
                error?.let{Text(it,Modifier.padding(top=12.dp),color=MaterialTheme.colorScheme.error)}}}
        }
        Surface(Modifier.align(Alignment.BottomCenter).fillMaxWidth(),color=MaterialTheme.colorScheme.surface){Column(Modifier.navigationBarsPadding().padding(18.dp)){
            error?.let{Text(it,Modifier.padding(bottom=9.dp),color=MaterialTheme.colorScheme.error,style=MaterialTheme.typography.bodySmall)}
            MotionButton({error=if(!state.balanceKnown)"余额尚未同步，请先刷新钱包" else if(!state.commissions.validate(draft))state.commissions.error else if(cents>state.balance)"可用余额不足" else null;if(error==null)pending=true},Modifier.fillMaxWidth().height(52.dp),enabled=!copying&&!pending){Text(if(copying)"正在上传封面…" else "预付报酬并发布")}
        }}
    }
    if(pending)PaymentExperience(cents,"委托预付 · 平台担保","委托已发布",{result=state.commissions.network?.publish(key,draft);result!=null},autoCloseOnSuccess=true,errorMessage=state.storageMessage){pending=false;result?.let(onPublished)}
}

@Composable
fun CommissionDetailPage(state:LabState,id:String,topInset:Dp,onChat:(String)->Unit,onDeleted:()->Unit={}) {
    val book=state.commissions;val scope=rememberCoroutineScope();var deleting by remember{mutableStateOf(false)}
    LaunchedEffect(id){while(true){book.network?.refreshOne(id);delay(5000)}}
    val entry=book.find(id) ?: return;val owner=entry.owner==state.userName;val worker=entry.worker==state.userName
    var intervention by rememberSaveable{mutableStateOf(false)}
    var action by remember{mutableStateOf<String?>(null)};var tools by remember{mutableStateOf(false)};var note by rememberSaveable{mutableStateOf("")};var error by remember{mutableStateOf<String?>(null)}
    var decision by remember{mutableStateOf<Boolean?>(null)};var decisionActor by remember{mutableStateOf<String?>(null)}
    val format=DateTimeFormatter.ofPattern("MM-dd HH:mm")
    Box(Modifier.fillMaxSize()){
        LazyColumn(contentPadding=PaddingValues(start=18.dp,end=18.dp,top=topInset,bottom=150.dp),verticalArrangement=Arrangement.spacedBy(20.dp)){
            item{CommissionImage(entry.draft,Modifier.fillMaxWidth().height(180.dp).clip(RoundedCornerShape(25.dp)))}
            item{Row(verticalAlignment=Alignment.CenterVertically){Text(entry.status,Modifier.weight(1f),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.primary);UrgencyBadge(entry.draft.urgency)};Text(entry.draft.title,Modifier.padding(top=10.dp),style=MaterialTheme.typography.headlineSmall);Text("${credit(entry.draft.reward)} 信用点",Modifier.padding(top=10.dp),style=MaterialTheme.typography.titleLarge,color=MaterialTheme.colorScheme.primary)}
            item{LabCard{
                val labels=listOf("待接取","进行中","已完成","已确认");val active=entry.stage.ordinal.coerceAtMost(3)
                Row(Modifier.fillMaxWidth(),horizontalArrangement=Arrangement.SpaceBetween){labels.forEachIndexed{index,label->Column(horizontalAlignment=Alignment.CenterHorizontally){Icon(if(index<active)Icons.Outlined.CheckCircle else Icons.Outlined.RadioButtonChecked,null,Modifier.size(19.dp),tint=if(entry.held&&index<=active||entry.stage==CommissionStage.Confirmed)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.outlineVariant);Text(label,Modifier.padding(top=8.dp),style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}}
                if(entry.stage in listOf(CommissionStage.Active,CommissionStage.Completed)){
                    val paused=entry.platformPending||entry.refund==RefundState.Requested;val remaining=(if(paused)entry.pausedMillis else entry.deadlineMillis?.minus(book.nowMillis)) ?: 0
                    TimeRemainingCard(if(entry.platformPending)"平台介入中 · 计时已暂停" else if(paused)"退款协商中 · 计时已暂停" else if(entry.stage==CommissionStage.Completed)"距离自动确认" else if(remaining<=0)"已超过约定履约时间" else "履约剩余时间",remaining,(if(entry.stage==CommissionStage.Completed)72 else entry.draft.workHours)*3600000L,
                        if(entry.platformPending)"等待平台裁决，报酬继续冻结。" else if(paused)"双方处理退款后，继续剩余时间。" else if(entry.stage==CommissionStage.Completed)"请核对完成说明，72 小时后自动结算。" else "履约超时不会自动结算，请及时沟通。",paused)
                }
            }}
            item{LabCard{Text("委托要求",style=MaterialTheme.typography.titleMedium);Text(entry.draft.description,Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodyLarge);DetailRow("地点",entry.draft.location);DetailRow("约定时限",durationHours(entry.draft.workHours))}}
            if(entry.completionNote.isNotBlank())item{LabCard{Text("完成说明",style=MaterialTheme.typography.titleMedium);Text(entry.completionNote,Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodyLarge)}}
            item{LabCard{DetailRow("发布者",entry.owner);DetailRow("接取者",entry.worker ?: "等待接取");DetailRow("资金状态",if(entry.intervention!=null)interventionFundsText(entry.intervention,entry.held) else if(entry.held)"平台担保 ${credit(entry.draft.reward)}" else if(entry.stage==CommissionStage.Confirmed)"已结算给接取者" else "已退回发布者");DetailRow("发布时间",entry.createdAt.format(format));entry.acceptedAt?.let{DetailRow("接取时间",it.format(format))};entry.completedAt?.let{DetailRow("完成时间",it.format(format))};entry.confirmedAt?.let{DetailRow("确认时间",it.format(format))};DetailRow("委托编号",entry.id)
                if(owner||worker)PlainButton({onChat(if(owner)entry.worker ?: entry.owner else entry.owner)},Modifier.fillMaxWidth(),enabled=!owner||entry.worker!=null){Text("联系${if(owner)"接取者" else "发布者"}")}}}
            if(entry.refundAttempts>0)item{LabCard{Text("退款记录",style=MaterialTheme.typography.titleMedium);DetailRow("申请说明",entry.refundReason);if(entry.rejectionReason.isNotBlank())DetailRow("拒绝理由",entry.rejectionReason);Text(if(entry.refund==RefundState.Requested)"等待接取者处理，自动确认暂停。" else "拒绝后可申请平台介入，退款申请次数不恢复。",Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
            if(owner||worker)entry.intervention?.let{case->item{InterventionSummary(case){intervention=true}}}
            if((owner||worker)&&entry.interventionCaseId!=null&&entry.intervention==null)item{PlainButton({intervention=true}){Text("查看平台介入")}}
            if(entry.canHideRecord)item{DeleteRecordButton({deleting=true},Modifier.fillMaxWidth())}
        }
        Surface(Modifier.align(Alignment.BottomCenter).fillMaxWidth(),color=MaterialTheme.colorScheme.surface){Row(Modifier.navigationBarsPadding().padding(18.dp),horizontalArrangement=Arrangement.spacedBy(10.dp)){
            when{
                entry.pendingOperationId!=null->SecondaryButton({},Modifier.fillMaxWidth(),enabled=false){Text(entry.status)}
                entry.serverStatus=="FUNDING"->SecondaryButton({},Modifier.fillMaxWidth(),enabled=false){Text("预付结果待确认")}
                (owner||worker)&&entry.platformPending->MotionButton({intervention=true},Modifier.fillMaxWidth()){Text("查看平台介入进度")}
                owner&&!entry.held&&(entry.canIntervene||entry.intervention!=null)->MotionButton({intervention=true},Modifier.fillMaxWidth()){Text(if(entry.intervention!=null)"查看平台判决" else "申请平台介入")}
                worker&&entry.refund==RefundState.Requested->{SecondaryButton({decisionActor=entry.worker;decision=false},Modifier.weight(1f)){Text("拒绝退款")};MotionButton({decisionActor=entry.worker;decision=true},Modifier.weight(1f)){Text("同意退款")}}
                owner&&entry.refund==RefundState.Requested->SecondaryButton({action="withdraw"},Modifier.fillMaxWidth()){Text("撤回退款申请")}
                owner&&entry.visibleInHall->SecondaryButton({action="cancel"},Modifier.fillMaxWidth()){Text("取消委托并退款")}
                owner&&entry.held->{SecondaryButton({if(entry.canIntervene||entry.intervention!=null)intervention=true else {note="";action="refund"}},Modifier.weight(1f),enabled=entry.canIntervene||entry.intervention!=null||entry.canRefund){Text(if(entry.intervention!=null)"查看平台介入" else if(entry.canIntervene)"申请平台介入" else if(entry.refundAttempts>0)"退款机会已用" else "申请退款")};MotionButton({action="confirm"},Modifier.weight(1f),enabled=!entry.platformPending&&entry.stage==CommissionStage.Completed){Text(if(entry.stage==CommissionStage.Completed)"确认完成" else "等待完成")}}
                worker&&entry.stage==CommissionStage.Active->MotionButton({note="";action="complete"},Modifier.fillMaxWidth()){Text("提交已完成")}
                !owner&&entry.visibleInHall->MotionButton({action="accept"},Modifier.fillMaxWidth()){Text("接取委托")}
                else->SecondaryButton({},Modifier.fillMaxWidth(),enabled=false){Text(if(worker&&entry.stage==CommissionStage.Completed)"等待发布者确认" else entry.status)}
            }
        }}
    }
    if(deleting)DeleteRecordDialog(entry.draft.title,{deleting=false}){(book.network?.hide(id)==true).also{if(it)onDeleted()}}
    if(intervention)CommissionInterventionSheet(book,entry){intervention=false}
    if(action in listOf("accept","cancel","confirm","withdraw")){val selected=action;IosDialog({action=null},{Text(when(selected){"accept"->"确认接取委托？";"cancel"->"取消并退回报酬？";"withdraw"->"撤回退款申请？";else->"确认委托已完成？"})},{Text(when(selected){"accept"->"接取后开始 ${durationHours(entry.draft.workHours)} 的履约时限，请确认能够完成约定要求。";"cancel"->"尚未接取的委托将关闭，预付报酬原路退回钱包。";"withdraw"->"计时会恢复，唯一一次退款机会不会恢复。";else->"${credit(entry.draft.reward)} 信用点将结算给 ${entry.worker}。请先核对交付内容。"})},{PlainButton({scope.launch{if(book.network?.action(id,selected ?: "confirm")==true)action=null else error=book.network?.error}}){Text("确认")}}, {PlainButton({action=null}){Text("取消")}})}
    if(action in listOf("complete","refund"))IosSheet({action=null}){Column(Modifier.verticalScroll(rememberScrollState()).padding(24.dp)){
        val completing=action=="complete";Text(if(completing)"提交完成说明" else "申请委托退款",style=MaterialTheme.typography.headlineSmall)
        Text(if(completing)"说明已完成的内容和验收方式，方便发布者检查。" else "接取后只可申请一次，请先与对方沟通。申请后计时暂停。",Modifier.padding(vertical=16.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
        RefinedField(note,{note=it.take(500);error=null},Modifier.fillMaxWidth(),label={Text(if(completing)"完成说明" else "退款说明")},minLines=4,maxLines=7)
        error?.let{Text(it,color=MaterialTheme.colorScheme.error)}
        MotionButton({if(!completing)action="refund-confirm" else scope.launch{if(book.network?.action(id,"complete",note)==true)action=null else error=book.network?.error ?: "状态已变化，请检查委托"}},Modifier.fillMaxWidth().padding(top=22.dp),enabled=note.trim().length>=2){Text(if(completing)"提交完成" else "继续申请退款")}
    }}
    if(action=="refund-confirm")IosDialog({action="refund"},{Text("确认使用唯一退款机会？")},{Text("${credit(entry.draft.reward)} 信用点的申请将发送给接取者。请先确认已经沟通，撤回或拒绝后也不能再次申请。")},{PlainButton({scope.launch{if(book.network?.action(id,"request-refund",note)==true)action=null else {error=book.network?.error;action="refund"}}}){Text("确认提交")}}, {PlainButton({action="refund"}){Text("再想想")}})
    decision?.let{approve->RefundDecisionDialog(approve,entry.draft.reward,{decision=null}){reason->(book.network?.action(id,"resolve",reason,approve)==true).also{if(it)decision=null}}}

}
