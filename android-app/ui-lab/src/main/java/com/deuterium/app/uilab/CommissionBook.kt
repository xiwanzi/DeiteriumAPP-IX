package com.deuterium.app.uilab

import androidx.compose.runtime.*
import java.time.*

enum class CommissionStage { Open, Active, Completed, Confirmed, Cancelled }
enum class Urgency(val label:String) { Normal("普通"), Soon("加急"), Urgent("紧急") }
data class CommissionDraft(val title:String,val description:String,val location:String,val reward:Long,val workHours:Int,val urgency:Urgency,val coverUri:String?,val artKey:String="")
data class Commission(val id:String,val key:String,val owner:String,val draft:CommissionDraft,val createdAt:LocalDateTime,
    val stage:CommissionStage=CommissionStage.Open,val worker:String?=null,val acceptedAt:LocalDateTime?=null,
    val completedAt:LocalDateTime?=null,val confirmedAt:LocalDateTime?=null,val deadlineMillis:Long?=null,
    val completionNote:String="",val refund:RefundState=RefundState.None,val refundAttempts:Int=0,
    val refundReason:String="",val rejectionReason:String="",val pausedMillis:Long?=null,val automatic:Boolean=false,val intervention:InterventionCase?=null,
    val serverStatus:String?=null,val fundsStatus:String?=null,val serverActions:Set<String>?=null,val version:Long=1,val refundId:String?=null,val refundVersion:Long=1,val interventionCaseId:String?=null,val pendingOperationId:String?=null) {
    val platformPending:Boolean get()=fundsStatus=="INTERVENTION_HOLD"||intervention?.pending==true
    val canIntervene:Boolean get()=pendingOperationId==null&&(serverActions?.contains("REQUEST_INTERVENTION") ?: (refund==RefundState.Rejected&&intervention==null))
    val held:Boolean get()=fundsStatus?.let{it in setOf("HELD","INTERVENTION_HOLD","REFUNDING","SETTLING")} ?: (stage !in listOf(CommissionStage.Cancelled,CommissionStage.Confirmed))
    val status:String get()=when{fundsStatus=="UNPAID"->if(serverStatus=="CANCELLED")"已取消 · 未扣款" else "预付未完成";fundsStatus=="SETTLING"->"结算处理中";fundsStatus=="REFUNDING"->"退款处理中";fundsStatus=="UNKNOWN"->"资金结果待确认";pendingOperationId!=null->"操作处理中";serverStatus=="FUNDING"->"预付结果待确认";platformPending->"平台介入中";intervention?.decision!=null->"平台判决 · ${intervention.resultText}";refund==RefundState.Requested->"退款协商中";refund==RefundState.Approved->"已退款";else->when(stage){CommissionStage.Open->"待接取";CommissionStage.Active->"进行中";CommissionStage.Completed->"已完成 · 待确认";CommissionStage.Confirmed->"已确认";CommissionStage.Cancelled->"已取消"}}
    val canRefund:Boolean get()=pendingOperationId==null&&(serverActions?.contains("REQUEST_REFUND") ?: (!platformPending&&stage in listOf(CommissionStage.Active,CommissionStage.Completed)&&refundAttempts==0&&refund==RefundState.None))
}

/** A local prepaid work board. Server-side escrow remains authoritative in the API contract. */
class CommissionBook(val userName:String,private val balance:()->Long,private val cash:(Long,String,String)->Unit,
    private val clock:Clock=Clock.systemDefaultZone(),val remoteOnly:Boolean=false,private val notify:(TradeNotice)->Unit={}) {
    val entries=mutableStateListOf<Commission>()
    var network:BackendCommissions?=null
    var interventions:BackendInterventions?=null
    val payments=mutableStateMapOf<String,Long>()
    var nextSequence=1L;private set
    var nowMillis by mutableLongStateOf(clock.millis());private set
    var error by mutableStateOf<String?>(null);private set
    val heldForUser:Long get()=entries.filter{it.owner==userName&&it.held}.sumOf{it.draft.reward}
    private fun now()=LocalDateTime.now(clock)
    fun find(id:String)=entries.find{it.id==id}
    private fun invalid(message:String):Boolean{error=message;return false}
    private fun replace(entry:Commission){entries[entries.indexOfFirst{it.id==entry.id}]=entry;error=null}
    private fun emit(entry:Commission,recipient:String,title:String,body:String,kind:String){notify(TradeNotice("CN${nextSequence++}",recipient,title,body,entry.id,"commission_$kind",now()))}
    fun validate(draft:CommissionDraft):Boolean {
        if(draft.title.trim().length !in 2..60||draft.description.trim().length !in 5..2000||draft.location.trim().length !in 2..120)return invalid("请填写标题、具体要求与地点")
        if(draft.reward !in 1..999999999L)return invalid("请输入有效报酬，最多两位小数")
        if(draft.workHours !in 1..8760)return invalid("履约时限需为 1 小时至 365 天")
        if(draft.coverUri.isNullOrBlank()&&draft.artKey.isBlank())return invalid("请添加委托封面")
        return true
    }
    @Synchronized fun publish(key:String,draft:CommissionDraft):String? {
        if(remoteOnly){invalid("委托发布服务暂不可用，请稍后重试");return null}
        val clean=draft.copy(title=draft.title.trim(),description=draft.description.trim(),location=draft.location.trim())
        entries.find{it.key==key&&it.owner==userName}?.let{if(it.draft!=clean){invalid("相同请求不能提交不同委托");return null};return it.id}
        if(key.isBlank()||!validate(clean))return null
        if(clean.reward>balance()){invalid("可用余额不足，无法预付报酬");return null}
        val entry=Commission("CM${clock.millis()}-${nextSequence++}",key,userName,clean,now())
        cash(-clean.reward,"委托预付","${clean.title} · 平台担保")
        entries.add(0,entry);error=null
        emit(entry,userName,"委托已发布","${clean.title}：报酬已预付，等待接取。","published")
        return entry.id
    }
    @Synchronized fun accept(id:String,actor:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        advanceTime();val entry=find(id) ?: return false
        if(entry.stage!=CommissionStage.Open||actor.isBlank()||actor==entry.owner)return invalid("委托已被接取，或不能接取自己的委托")
        val accepted=entry.copy(stage=CommissionStage.Active,worker=actor,acceptedAt=now(),deadlineMillis=clock.millis()+entry.draft.workHours*3600000L)
        replace(accepted);emit(accepted,entry.owner,"委托已被接取","$actor 已接取 ${entry.draft.title}。","accepted");emit(accepted,actor,"接取成功","履约时限已开始，请及时与发布者沟通。","assigned");return true
    }
    @Synchronized fun complete(id:String,actor:String,note:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        advanceTime();val entry=find(id) ?: return false
        if(entry.platformPending||entry.worker!=actor||entry.stage!=CommissionStage.Active||entry.refund==RefundState.Requested)return false
        if(note.trim().length !in 2..500)return invalid("请填写 2–500 字的完成说明")
        val completed=entry.copy(stage=CommissionStage.Completed,completedAt=now(),completionNote=note.trim(),deadlineMillis=clock.millis()+72*3600000L)
        replace(completed);emit(completed,entry.owner,"委托已完成，等待确认","$actor 已提交完成说明，72 小时后自动结算。","completed");return true
    }
    private fun settle(entry:Commission,automatic:Boolean):Boolean {
        val worker=entry.worker ?: return false
        if(entry.platformPending||entry.stage!=CommissionStage.Completed||entry.refund==RefundState.Requested)return false
        payments[worker]=(payments[worker] ?: 0)+entry.draft.reward
        if(worker==userName)cash(entry.draft.reward,entry.owner,"委托结算 · ${entry.draft.title}")
        val confirmed=entry.copy(stage=CommissionStage.Confirmed,confirmedAt=now(),deadlineMillis=null,automatic=automatic)
        replace(confirmed);emit(confirmed,worker,"委托报酬已到账","${entry.draft.title} 已确认，报酬已结算。","settled")
        emit(confirmed,entry.owner,if(automatic)"委托已自动确认" else "委托已确认","预付报酬已结算给 $worker。","confirmed");return true
    }
    @Synchronized fun confirm(id:String,actor:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");advanceTime();val entry=find(id) ?: return false;if(entry.owner!=actor)return false;return settle(entry,false)}
    @Synchronized fun cancel(id:String,actor:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val entry=find(id) ?: return false
        if(entry.owner!=actor||entry.stage!=CommissionStage.Open)return invalid("接取后需要与对方协商退款")
        return refund(entry,"发布者取消了尚未接取的委托")
    }
    @Synchronized fun requestRefund(id:String,actor:String,reason:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        advanceTime();val entry=find(id) ?: return false
        if(entry.owner!=actor||!entry.canRefund||reason.trim().length !in 2..500)return invalid("当前无法申请，或退款说明不完整")
        val updated=entry.copy(refund=RefundState.Requested,refundAttempts=1,refundReason=reason.trim(),pausedMillis=((entry.deadlineMillis ?: clock.millis())-clock.millis()).coerceAtLeast(0),deadlineMillis=null)
        replace(updated);emit(updated,entry.worker!!,"委托退款申请","${entry.owner} 提交了退款说明，请沟通后处理。","refund_requested");return true
    }
    @Synchronized fun resolveRefund(id:String,actor:String,approve:Boolean,reason:String=""):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val entry=find(id) ?: return false
        if(entry.platformPending||entry.worker!=actor||entry.refund!=RefundState.Requested)return false
        if(approve)return refund(entry,entry.refundReason)
        if(reason.trim().length !in 2..500)return invalid("请填写拒绝理由")
        val updated=entry.copy(refund=RefundState.Rejected,rejectionReason=reason.trim(),deadlineMillis=clock.millis()+(entry.pausedMillis ?: 0),pausedMillis=null)
        replace(updated);emit(updated,entry.owner,"委托退款未获同意",reason.trim(),"refund_rejected");return true
    }
    @Synchronized fun withdrawRefund(id:String,actor:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val entry=find(id) ?: return false
        if(entry.owner!=actor||entry.refund!=RefundState.Requested)return false
        replace(entry.copy(refund=RefundState.None,deadlineMillis=clock.millis()+(entry.pausedMillis ?: 0),pausedMillis=null))
        emit(entry,entry.worker!!,"委托退款申请已撤回","继续按约定完成委托。","refund_withdrawn");return true
    }
    @Synchronized fun requestIntervention(id:String,actor:String,key:String,form:InterventionForm):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        advanceTime();val entry=find(id) ?: return false
        if(actor!=entry.owner)return false
        val clean=form.normalized()
        entry.intervention?.let{return if(it.requestKey==key&&it.form==clean)true else invalid("该委托已提交平台介入，请查看现有案件")}
        if(!entry.canIntervene||key.isBlank()||!clean.valid(entry.draft.reward)||entry.worker==null)return invalid("请填写完整的介入原因与诉求")
        val snapshot=InterventionSnapshot(id,entry.draft.title,entry.owner,entry.worker,entry.draft.reward,entry.draft.location,
            "${entry.draft.description}；履约 ${entry.draft.workHours} 小时，完成后 72 小时验收；完成说明：${entry.completionNote}",entry.createdAt,entry.refundReason,entry.rejectionReason,entry.status,now())
        val case=InterventionCase("CASE-C${nextSequence++}",key,clean,snapshot,entry.held)
        val updated=entry.copy(intervention=case,pausedMillis=if(entry.held)entry.deadlineMillis?.let{(it-clock.millis()).coerceAtLeast(0)} ?: entry.pausedMillis else null,deadlineMillis=null)
        replace(updated);listOf(entry.owner,entry.worker).forEach{emit(updated,it,"平台介入申请已提交",if(entry.held)"预付报酬已冻结，计时暂停，等待平台处理。" else "委托已结算，平台将人工审核争议。","intervention_submitted")};return true
    }
    @Synchronized fun simulateInterventionReview(id:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val entry=find(id) ?: return false;val case=entry.intervention ?: return false
        if(case.status!=InterventionStatus.Submitted)return false
        replace(entry.copy(intervention=case.copy(status=InterventionStatus.InReview,updatedAt=now())));return true
    }
    /** Local experience tool only; a real verdict is an administrator operation. */
    @Synchronized fun simulatePlatformDecision(id:String,decision:PlatformDecision,refundAmount:Long,reason:String):Boolean {if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");
        val entry=find(id) ?: return false;val case=entry.intervention ?: return false
        val result=interventionOutcome(case,decision,refundAmount,reason) ?: return invalid("判决无效、金额超出担保范围，或案件已处理")
        if(case.fundsHeldForReview&&!entry.held)return invalid("担保状态已变化，请重新查询")
        val resolved=case.copy(status=InterventionStatus.Resolved,updatedAt=now(),decision=decision,decisionReason=reason.trim(),refunded=result.refund,paidToPayee=result.payout)
        var updated=entry.copy(intervention=resolved)
        if(decision==PlatformDecision.FullRefund)refund(updated,entry.refundReason)
        else if(result.refund>0||result.payout>0){
            if(result.refund>0&&entry.owner==userName)cash(result.refund,"平台判决退款",entry.draft.title)
            if(result.payout>0){val worker=entry.worker ?: return false;payments[worker]=(payments[worker] ?: 0)+result.payout;if(worker==userName)cash(result.payout,entry.owner,"平台判决结算 · ${entry.draft.title}")}
            updated=updated.copy(stage=CommissionStage.Confirmed,confirmedAt=now(),deadlineMillis=null,pausedMillis=null);replace(updated)
        } else {if(result.resume)updated=updated.copy(deadlineMillis=entry.pausedMillis?.let{clock.millis()+it},pausedMillis=null);replace(updated)}
        listOfNotNull(entry.owner,entry.worker).forEach{emit(find(id)!!,it,"平台判决：${decision.label}",reason.trim(),"intervention_resolved")};return true
    }
    private fun refund(entry:Commission,reason:String):Boolean {
        if(!entry.held)return false
        if(entry.owner==userName)cash(entry.draft.reward,"委托退款",entry.draft.title)
        val updated=entry.copy(stage=CommissionStage.Cancelled,refund=RefundState.Approved,refundReason=reason,deadlineMillis=null,pausedMillis=null)
        replace(updated);emit(updated,entry.owner,"委托报酬已退回",entry.draft.title,"refunded");entry.worker?.let{emit(updated,it,"委托已退款",reason,"closed")};return true
    }
    @Synchronized fun advanceTime(){nowMillis=clock.millis();if(remoteOnly)return;entries.filter{!it.platformPending&&it.stage==CommissionStage.Completed&&it.refund!=RefundState.Requested&&it.deadlineMillis!=null&&it.deadlineMillis<=nowMillis}.toList().forEach{settle(it,true)}}
    @Synchronized fun previewDeadline(id:String):Boolean{if(remoteOnly)return invalid("交易服务暂不可用，请稍后重试");val entry=find(id) ?: return false;if(entry.platformPending||entry.stage !in listOf(CommissionStage.Active,CommissionStage.Completed)||entry.refund==RefundState.Requested)return false;replace(entry.copy(deadlineMillis=clock.millis()+10000));return true}
    fun restore(saved:List<Commission>,savedPayments:Map<String,Long>,sequence:Long){entries.clear();entries.addAll(saved);payments.clear();payments.putAll(savedPayments);nextSequence=sequence.coerceAtLeast(1)}
    fun reset(){entries.clear();payments.clear();nextSequence=1;error=null}
}
