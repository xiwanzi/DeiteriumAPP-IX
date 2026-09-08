package com.deuterium.app.uilab

import androidx.compose.runtime.*
import org.json.JSONArray
import org.json.JSONObject
import java.time.Instant
import java.time.LocalDateTime
import java.time.ZoneId
import java.util.UUID

class BackendCommissions(private val api:BackendApi,private val state:LabState) {
    private var visibilityRevision=0L
    var error by mutableStateOf<String?>(null);private set
    private fun fail(value:Throwable){error=value.message ?: "委托服务暂不可用";state.storageMessage=error}
    private fun date(value:JSONObject,key:String):LocalDateTime?=value.optString(key).takeUnless{it.isBlank()||it=="null"}?.let{Instant.parse(it).atZone(ZoneId.systemDefault()).toLocalDateTime()}
    private fun epoch(value:JSONObject,key:String):Long?=value.optString(key).takeUnless{it.isBlank()||it=="null"}?.let{Instant.parse(it).toEpochMilli()}
    private fun put(value:JSONObject):String{
        if(value.optBoolean("hiddenFromHistory")){val id=value.getString("commissionId");state.commissions.entries.removeAll{it.id==id};return id}
        val c=value.getJSONObject("content");val refund=value.optJSONObject("refund")
        val actions=value.getJSONArray("availableActions");val set=(0 until actions.length()).map{actions.getString(it)}.toSet()
        val status=value.getString("status")
        val stage=when(status){"ACTIVE"->CommissionStage.Active;"COMPLETED"->CommissionStage.Completed;"CONFIRMED"->CommissionStage.Confirmed;"CANCELLED"->CommissionStage.Cancelled;else->CommissionStage.Open}
        val draft=CommissionDraft(c.getString("title"),c.getString("description"),c.getString("location"),apiCents(c.getString("reward")),c.getInt("workHours"),when(c.getString("urgency")){"SOON"->Urgency.Soon;"URGENT"->Urgency.Urgent;else->Urgency.Normal},"asset:${c.getString("coverAssetId")}")
        val id=value.getString("commissionId")
        val entry=Commission(id,id,value.getJSONObject("owner").getString("displayName"),draft,date(value,"createdAt")!!,stage=stage,worker=value.optJSONObject("worker")?.getString("displayName"),acceptedAt=date(value,"acceptedAt"),completedAt=date(value,"completedAt"),confirmedAt=date(value,"confirmedAt"),deadlineMillis=if(stage==CommissionStage.Completed)epoch(value,"acceptanceDueAt") else epoch(value,"workDueAt"),completionNote=value.optString("completionDescription"),
            refund=when(refund?.optString("status")){"REQUESTED","PROCESSING"->RefundState.Requested;"REJECTED"->RefundState.Rejected;"APPROVED"->RefundState.Approved;else->RefundState.None},refundAttempts=value.optInt("refundAttemptsUsed"),refundReason=refund?.optString("reason").orEmpty(),rejectionReason=refund?.optString("rejectionReason").orEmpty(),pausedMillis=if(value.isNull("pausedRemainingSeconds"))null else value.optLong("pausedRemainingSeconds")*1000,automatic=value.optBoolean("automatic"),
            serverStatus=status,fundsStatus=value.getString("fundsStatus"),serverActions=set,version=value.getLong("version"),refundId=refund?.optString("refundId"),refundVersion=refund?.optLong("version",1) ?: 1,interventionCaseId=value.optString("interventionCaseId").takeUnless{it.isBlank()||it=="null"},intervention=state.interventions?.cached(value.optString("interventionCaseId")),pendingOperationId=value.optString("pendingOperationId").takeUnless{it.isBlank()||it=="null"},canHideRecord=value.optBoolean("canHideRecord"))
        val index=state.commissions.entries.indexOfFirst{it.id==id};if(index>=0)state.commissions.entries[index]=entry else state.commissions.entries.add(0,entry)
        return id
    }
    suspend fun refresh(){val revision=visibilityRevision;runCatching{
        val values=api.listAll("/commissions")+api.listAll("/commissions/me")
        values.associateBy{it.getString("commissionId")}.values
    }.onSuccess{if(revision!=visibilityRevision)return@onSuccess;state.commissions.entries.clear();it.forEach{value->put(value)};error=null}.onFailure(::fail)}
    suspend fun refreshOne(id:String){val revision=visibilityRevision;runCatching{var value=api.request("GET","/commissions/$id");value.optString("pendingOperationId").takeUnless{it.isBlank()||it=="null"}?.let{operationId->runCatching{api.request("GET","/operations/$operationId")}.onSuccess{op->if(op.optString("status") in setOf("COMPLETED","FAILED"))value=api.request("GET","/commissions/$id")}};if(revision==visibilityRevision)put(value);value.optString("interventionCaseId").takeUnless{it.isBlank()||it=="null"}?.let{state.interventions?.refresh(it)}}.onFailure(::fail)}
    suspend fun hide(id:String):Boolean {
        val original=state.commissions.find(id) ?: return false
        return runCatching{
            val response=api.request("POST","/commissions/$id/hide",JSONObject().put("clientRequestId",UUID.nameUUIDFromBytes("${api.playerRef}:hide:commission:$id:${original.version}".toByteArray()).toString()).put("expectedVersion",original.version))
            require(response.getBoolean("hidden")){"删除未完成"};visibilityRevision++;state.commissions.entries.removeAll{it.id==id};error=null;true
        }.getOrElse{fail(it);false}
    }
    suspend fun publish(key:String,draft:CommissionDraft):String?{
        val requestScope=api.financialScope()
        val kind="COMMISSION_PUBLISH"
        if(api.pendingOperation(kind)!=null){recover();fail(ApiFailure("RESULT_UNKNOWN","上一笔预付结果仍需核对，请先查看我的委托"));return null}
        return runCatching{
            require(draft.coverUri?.startsWith("asset:")==true){"请先上传委托封面"}
            val content=JSONObject().put("title",draft.title.trim()).put("description",draft.description.trim()).put("location",draft.location.trim()).put("urgency",draft.urgency.name.uppercase())
                .put("reward",java.math.BigDecimal(draft.reward).movePointLeft(2).toPlainString()).put("workHours",draft.workHours).put("coverAssetId",draft.coverUri!!.removePrefix("asset:"))
            val request=JSONObject().put("clientRequestId",key).put("content",content)
            api.saveOperation(kind,JSONObject().put("request",request),requestScope)
            val result=api.request("POST","/commissions",request);val operation=result.getJSONObject("operation")
            api.saveOperation(kind,JSONObject().put("request",request).put("operationId",operation.getString("operationId")),requestScope)
            requestScope.verifyCurrent(api.financialScope())
            result.optJSONObject("commission")?.let{put(it)}
            if(operation.getString("status")=="FAILED"){api.saveOperation(kind,null,requestScope);error("委托预付失败")}
            require(operation.getString("status")=="COMPLETED"&&result.getJSONObject("commission").getString("fundsStatus")=="HELD"){"预付结果待确认，请勿重复付款"}
            api.saveOperation(kind,null,requestScope);state.refresh();error=null;result.getJSONObject("commission").getString("commissionId")
        }.getOrElse{failure->if(failure is ApiFailure&&(failure.status in listOf(400,401,403,404,409,422,501)||failure.code=="CAPABILITY_UNAVAILABLE"))api.saveOperation(kind,null,requestScope);fail(failure);null}
    }
    suspend fun recover(){
        val kind="COMMISSION_PUBLISH";val pending=api.pendingOperation(kind) ?: return
        val requestScope=api.financialScope()
        runCatching{
            val op=recoverPendingOperation(kind,pending,requestScope,{api.financialScope()},
                {method,path,body->api.request(method,path,body)},{value->api.saveOperation(kind,value,requestScope)})
            when(op.getString("status")){"COMPLETED"->{val commission=api.request("GET","/commissions/${op.getString("resourceId")}");requestScope.verifyCurrent(api.financialScope());put(commission);api.saveOperation(kind,null,requestScope);state.refresh()};"FAILED"->api.saveOperation(kind,null,requestScope);else->fail(ApiFailure("RESULT_UNKNOWN","委托预付结果待确认，请勿重复付款"))}
        }.onFailure(::fail)
    }
    suspend fun action(id:String,action:String,description:String="",approve:Boolean?=null):Boolean=runCatching{
        val original=state.commissions.find(id) ?: return false
        val nested=action in setOf("withdraw","resolve")
        val input=JSONObject().put("expectedVersion",if(nested)original.refundVersion else original.version)
        val suffix=when(action){
            "request-refund"->{input.put("reasonCode","OTHER").put("description",description).put("evidenceAssetIds",JSONArray());"refunds"}
            "withdraw"->"refunds/${original.refundId ?: error("退款记录尚未同步")}/withdraw"
            "resolve"->{input.put("decision",if(approve==true)"APPROVE" else "REJECT").put("reason",description);"refunds/${original.refundId ?: error("退款记录尚未同步")}/resolve"}
            "complete"->{input.put("description",description).put("evidenceAssetIds",JSONArray());action}
            else->action
        }
        input.put("clientRequestId",UUID.nameUUIDFromBytes("${api.playerRef}:$id:$suffix:$input".toByteArray()).toString())
        val response=api.request("POST","/commissions/$id/$suffix",input)
        val value=if(action in setOf("cancel","confirm"))response.optJSONObject("commission") ?: api.request("GET","/commissions/$id") else response
        put(value)
        if(!value.isNull("pendingOperationId")&&value.optString("pendingOperationId").isNotBlank())error("请求正在处理，请勿重复操作；状态将自动更新")
        if(action in setOf("cancel","confirm"))require(response.getJSONObject("operation").getString("status")=="COMPLETED"){"资金处理结果待确认，请勿重复操作"}
        if(action=="resolve"&&approve==true)require(value.getString("fundsStatus")=="REFUNDED"){"退款结果待确认，请勿重复操作"}
        state.refresh();error=null;true
    }.getOrElse{fail(it);runCatching{put(api.request("GET","/commissions/$id"))};false}
}
