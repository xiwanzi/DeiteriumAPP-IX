package com.deuterium.app.uilab

import androidx.compose.runtime.*
import org.json.JSONArray
import org.json.JSONObject
import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneId

/** Player case submission and read only. This client has no administrator decision endpoint. */
class BackendInterventions(private val api:BackendApi,private val state:LabState) {
    private val cases=mutableMapOf<String,InterventionCase>()
    var error by mutableStateOf<String?>(null);private set
    fun cached(id:String?)=id?.let{cases[it]}
    private fun date(value:JSONObject,key:String):LocalDateTime=Instant.parse(value.getString(key)).atZone(ZoneId.systemDefault()).toLocalDateTime()
    private fun reason(code:String)=when(code){"NOT_DELIVERED"->InterventionReason.NotDelivered;"NOT_AS_DESCRIBED"->InterventionReason.NotAsDescribed;"REFUND_DISAGREEMENT"->InterventionReason.RefundDisagreement;else->InterventionReason.Other}
    private fun put(value:JSONObject,submitted:InterventionForm?=null):InterventionCase {
        val evidence=value.getJSONArray("evidenceAssetIds");val kind=value.getString("transactionKind");val snapshot=value.getJSONObject("snapshot")
        val transaction=snapshot.getJSONObject(if(kind=="ORDER")"order" else "commission")
        val content=transaction.optJSONObject("content");val refund=transaction.optJSONObject("refund")
        val items=transaction.optJSONArray("items")
        val title=content?.optString("title") ?: (0 until (items?.length() ?: 0)).joinToString("、"){items!!.getJSONObject(it).getString("title")}
        val payer=transaction.getJSONObject(if(kind=="ORDER")"buyer" else "owner").getString("displayName")
        val payee=transaction.optJSONObject(if(kind=="ORDER")"seller" else "worker")?.optString("displayName").orEmpty()
        val total=apiCents(content?.getString("reward") ?: transaction.getString("amount"))
        val captured=InterventionSnapshot(value.getString("transactionId"),title,payer,payee,total,content?.optString("location") ?: transaction.getJSONObject("delivery").optString("location"),
            content?.optString("description") ?: (0 until (items?.length() ?: 0)).joinToString("；"){items!!.getJSONObject(it).optString("description")},date(transaction,"createdAt"),refund?.optString("reason").orEmpty(),refund?.optString("rejectionReason").orEmpty(),transaction.getString("status"),date(snapshot,"capturedAt"))
        val form=InterventionForm(submitted?.reason ?: reason(value.optString("reasonCode")),value.getString("description"),when(value.getString("desiredResolution")){"FULL_REFUND"->InterventionWish.FullRefund;"PARTIAL_REFUND"->InterventionWish.PartialRefund;"CONTINUE_FULFILLMENT"->InterventionWish.ContinueFulfillment;else->InterventionWish.Other},
            value.optString("requestedRefundAmount").takeUnless{it.isBlank()||it=="null"}?.let(::apiCents),(0 until evidence.length()).map{"asset:${evidence.getString(it)}"})
        val status=when(value.getString("status")){"SUBMITTED"->InterventionStatus.Submitted;"IN_REVIEW"->InterventionStatus.InReview;"WAITING_EVIDENCE"->InterventionStatus.WaitingEvidence;"RESOLVING"->InterventionStatus.Resolving;"RESOLVED"->InterventionStatus.Resolved;"WITHDRAWN"->InterventionStatus.Withdrawn;else->error("案件状态不受支持")}
        val case=InterventionCase(value.getString("caseId"),value.getString("caseId"),form,captured,value.getBoolean("fundsHeldForReview"),status,date(value,"updatedAt"),decisionReason=value.optString("resolution"),serverResolution=value.optString("resolution"),reasonDisplay=if(submitted==null&&!value.has("reasonCode"))"见申请说明" else null)
        cases[case.id]=case
        if(kind=="ORDER"){val index=state.commerce.orders.indexOfFirst{it.id==case.snapshot.transactionId};if(index>=0)state.commerce.orders[index]=state.commerce.orders[index].copy(intervention=case,interventionCaseId=case.id)}
        else {val index=state.commissions.entries.indexOfFirst{it.id==case.snapshot.transactionId};if(index>=0)state.commissions.entries[index]=state.commissions.entries[index].copy(intervention=case,interventionCaseId=case.id)}
        error=null;return case
    }
    suspend fun refresh(id:String){runCatching{put(api.request("GET","/interventions/$id"))}.onFailure{error=it.message;state.storageMessage=error}}
    suspend fun submit(kind:String,id:String,version:Long,key:String,form:InterventionForm):Boolean=runCatching{
        require(form.evidence.all{it.startsWith("asset:")}){"请先完成凭证上传"}
        val request=JSONObject().put("clientRequestId",key).put("expectedVersion",version).put("reasonCode",when(form.reason){InterventionReason.NotDelivered->"NOT_DELIVERED";InterventionReason.NotAsDescribed->"NOT_AS_DESCRIBED";InterventionReason.RefundDisagreement->"REFUND_DISAGREEMENT";InterventionReason.Other->"OTHER"})
            .put("description",form.description.trim()).put("desiredResolution",when(form.wish){InterventionWish.FullRefund->"FULL_REFUND";InterventionWish.PartialRefund->"PARTIAL_REFUND";InterventionWish.ContinueFulfillment->"CONTINUE_FULFILLMENT";InterventionWish.Other->"OTHER"})
            .put("evidenceAssetIds",JSONArray(form.evidence.map{it.removePrefix("asset:")}))
        form.requestedRefund?.let{request.put("requestedRefundAmount",java.math.BigDecimal(it).movePointLeft(2).toPlainString())}
        val value=api.request("POST","/${if(kind=="ORDER")"orders" else "commissions"}/$id/interventions",request)
        require(value.getString("transactionId")==id&&value.getString("transactionKind")==kind){"案件关联交易不正确"}
        put(value,form)
        if(kind=="ORDER")state.commerce.network?.refreshOrder(id) else state.commissions.network?.refreshOne(id)
        true
    }.getOrElse{error=it.message;state.storageMessage=error;false}
}
