package com.deuterium.app.uilab

import java.time.LocalDateTime

enum class InterventionReason(val label:String) { NotDelivered("未按约定交付"),NotAsDescribed("内容与约定不符"),RefundDisagreement("退款协商未达成一致"),Other("其他原因") }
enum class InterventionWish(val label:String) { FullRefund("全额退款"),PartialRefund("部分退款"),ContinueFulfillment("继续按约定履约"),Other("其他诉求") }
enum class InterventionStatus(val label:String) { Submitted("已提交 · 等待受理"),InReview("平台审核中"),WaitingEvidence("等待补充材料"),Resolving("判决执行中"),Resolved("平台已作出判决"),Withdrawn("申请已撤回") }
enum class PlatformDecision(val label:String) { FullRefund("全额退款"),PartialRefund("部分退款并结算剩余"),ReleaseToPayee("结算给收款方"),ContinueFulfillment("继续履约"),NoAction("不采取资金动作"),ManualRecovery("转人工协商处理") }
data class InterventionForm(val reason:InterventionReason,val description:String,val wish:InterventionWish,val requestedRefund:Long?,val evidence:List<String>) {
    fun valid(amount:Long)=description.trim().length in 10..3000&&evidence.size<=10&&evidence.all{it.isNotBlank()}&&
        (wish!=InterventionWish.PartialRefund||(requestedRefund!=null&&requestedRefund in 1 until amount))
    fun normalized()=copy(description=description.trim(),requestedRefund=if(wish==InterventionWish.PartialRefund)requestedRefund else null,evidence=evidence.toList())
}
data class InterventionSnapshot(val transactionId:String,val title:String,val payer:String,val payee:String,val amount:Long,
    val location:String,val terms:String,val transactionCreatedAt:LocalDateTime,val refundReason:String,val rejectionReason:String,
    val transactionStatus:String,val capturedAt:LocalDateTime)
data class InterventionCase(val id:String,val requestKey:String,val form:InterventionForm,val snapshot:InterventionSnapshot,
    val fundsHeldForReview:Boolean,val status:InterventionStatus=InterventionStatus.Submitted,
    val updatedAt:LocalDateTime=snapshot.capturedAt,val decision:PlatformDecision?=null,val decisionReason:String="",
    val refunded:Long=0,val paidToPayee:Long=0,val serverResolution:String?=null,val reasonDisplay:String?=null) {
    val pending:Boolean get()=status in listOf(InterventionStatus.Submitted,InterventionStatus.InReview,InterventionStatus.WaitingEvidence,InterventionStatus.Resolving)
    val resultText:String get()=decision?.label ?: status.label
}
data class InterventionOutcome(val refund:Long=0,val payout:Long=0,val resume:Boolean=false)
fun interventionOutcome(case:InterventionCase,decision:PlatformDecision,refundAmount:Long,reason:String):InterventionOutcome? {
    if(!case.pending||reason.trim().length !in 10..3000)return null
    if(!case.fundsHeldForReview)return if(decision in listOf(PlatformDecision.ManualRecovery,PlatformDecision.NoAction))InterventionOutcome() else null
    val amount=case.snapshot.amount
    return when(decision) {
        PlatformDecision.FullRefund->InterventionOutcome(refund=amount)
        PlatformDecision.PartialRefund->if(refundAmount in 1 until amount)InterventionOutcome(refundAmount,amount-refundAmount)else null
        PlatformDecision.ReleaseToPayee->InterventionOutcome(payout=amount)
        PlatformDecision.ContinueFulfillment,PlatformDecision.NoAction->InterventionOutcome(resume=true)
        PlatformDecision.ManualRecovery->null
    }
}
