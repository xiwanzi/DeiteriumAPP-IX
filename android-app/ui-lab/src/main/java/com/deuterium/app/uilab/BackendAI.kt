package com.deuterium.app.uilab

import androidx.compose.runtime.*
import kotlinx.coroutines.*
import okhttp3.Call
import okhttp3.Callback
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.io.Reader
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.UUID
import java.util.concurrent.TimeUnit
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

data class AiSource(val title:String,val url:String,val origin:String)

/** Semantic SSE; EOF is never a successful completion. No reasoning event is accepted. */
fun readAiEvents(reader:Reader,onEvent:(String,JSONObject)->Unit) {
    var event="";val data=StringBuilder();var finished=false
    fun dispatch(){if(data.isEmpty()){event="";return};val payload=JSONObject(data.toString());data.setLength(0);if(event in setOf("meta","status","delta","sources","done","error")){onEvent(event,payload);if(event=="done")finished=true};event=""}
    reader.buffered().use { input ->
        while(true) {
            val line=input.readLine() ?: break
            require(line.length<=262144&&data.length<=262144){"AI 响应过大"}
            when {
                line.isEmpty()->dispatch()
                line.startsWith("event:")->event=line.substringAfter(':').trim()
                line.startsWith("data:")->{if(data.isNotEmpty())data.append('\n');data.append(line.substringAfter(':').removePrefix(" "))}
            }
            if(finished)break
        }
        dispatch()
    }
    if(!finished)throw ApiFailure("AI_STREAM_INTERRUPTED","AI 连接已中断，可使用原请求恢复查看")
}

class BackendAI(private val api:BackendApi,private val state:LabState) {
    private val key="AI 助手"
    private val owner=api.playerRef
    private val http=api.http.newBuilder().callTimeout(215,TimeUnit.SECONDS).readTimeout(30,TimeUnit.SECONDS).build()
    var quotaText by mutableStateOf("正在读取 AI 状态…");private set
    var statusText by mutableStateOf("正在生成回复…");private set
    var conversationId by mutableStateOf<String?>(null);private set
    var assistantName by mutableStateOf("客服小祥");private set
    var recoveredDraft by mutableStateOf(api.pendingAI(owner)?.optString("draft",api.pendingAI(owner)?.optString("content").orEmpty()).orEmpty());private set
    var plans by mutableStateOf<List<JSONObject>>(emptyList());private set
    var busy by mutableStateOf(false);private set
    var lastRequestId:String?=null;private set
    private fun sources(value:JSONArray?):List<AiSource> = (0 until (value?.length() ?: 0)).mapNotNull{index->
        val item=value!!.optJSONObject(index) ?: return@mapNotNull null;val raw=item.optString("url")
        val url=runCatching{java.net.URI(raw)}.getOrNull() ?: return@mapNotNull null
        if(url.scheme !in setOf("https","http")||url.host.isNullOrBlank()||url.userInfo!=null)return@mapNotNull null
        AiSource(item.optString("title").ifBlank{url.host},raw,item.optString("origin","provider_text"))
    }
    private fun quota(value:JSONObject){
        quotaText=if(value.optBoolean("unlimited"))"管理员 · AI 次数不限" else {
            val at=runCatching{Instant.parse(value.getString("resetsAt")).atZone(ZoneId.systemDefault()).format(DateTimeFormatter.ofPattern("MM-dd HH:mm"))}.getOrDefault("待同步")
            "剩余 ${value.getInt("remaining")}/${value.getInt("limit")} · $at 恢复"
        }
    }
    private fun put(value:JSONObject){
        val id=value.getString("messageId");val list=state.conversation(key)
        val mine=value.getString("role")=="user"
        val at=Instant.parse(value.getString("createdAt")).atZone(ZoneId.systemDefault())
        val line=ChatLine(UUID.nameUUIDFromBytes(id.toByteArray()).mostSignificantBits,if(mine)"你" else assistantName,value.getString("content"),mine,at.format(DateTimeFormatter.ofPattern("HH:mm")),remoteId=id,serverAt=at.toInstant().toEpochMilli(),aiStatus=value.optString("status","completed"),sources=sources(value.optJSONArray("sources")),searchUsed=value.optBoolean("searchUsed"))
        val index=list.indexOfFirst{it.remoteId==id};if(index>=0)list[index]=line else list.add(line)
    }
    suspend fun refresh(){
        if(busy||api.playerRef!=owner)return
        runCatching{
            val info=api.request("GET","/ai/me");val history=api.request("GET","/ai/messages?limit=100").getJSONArray("messages")
            if(busy||api.playerRef!=owner)return
            assistantName=info.getString("assistantName");quota(info.getJSONObject("quota"))
            val current=info.getJSONObject("conversation").getString("conversationId")
            if(conversationId!=current){state.conversation(key).clear();conversationId=current}
            for(index in 0 until history.length())put(history.getJSONObject(index))
            val pending=info.optJSONObject("pendingRequest")
            if(api.pendingAI(owner)==null&&pending!=null){api.savePendingAI(owner,JSONObject().put("clientMessageId",pending.getString("clientMessageId")).put("content",pending.getString("content")).put("draft",pending.getString("content")).put("assistantMessageId",pending.getString("assistantMessageId")))}
            val local=api.pendingAI(owner)
            val message=local?.optString("assistantMessageId")?.let{id->state.conversation(key).find{it.remoteId==id}}
            if(message?.aiStatus=="completed"||message?.aiStatus=="failed"){api.savePendingAI(owner,null);recoveredDraft=""}else recoveredDraft=local?.optString("draft",local.optString("content")).orEmpty()
            state.directErrors.remove(key)
            if(pending!=null&&pending.optString("status") in setOf("unknown","incomplete"))state.directErrors[key]="上次回复未完整完成，已保留正文。可恢复原请求，或输入 /new 开始新对话。"
        }.onFailure{state.directErrors[key]=it.message ?: "AI 状态读取失败"}
    }
    suspend fun loadPlans(){runCatching{api.request("GET","/ai/plans").getJSONArray("plans")}.onSuccess{items->plans=(0 until items.length()).map{items.getJSONObject(it)}}.onFailure{state.directErrors[key]=it.message ?: "套餐读取失败"}}
    suspend fun reset():Boolean {
        if(busy){state.directErrors[key]="回复仍在生成，完成后可以开启新对话";return false}
        return runCatching{val next=api.request("POST","/ai/conversation/reset",JSONObject()).getJSONObject("conversation");conversationId=next.getString("conversationId");state.conversation(key).clear();api.savePendingAI(owner,null);recoveredDraft="";state.directErrors.remove(key);refresh();true}.getOrElse{state.directErrors[key]=it.message ?: "新对话创建失败";false}
    }
    suspend fun send(raw:String,reply:ChatReply?=null):Boolean {
        if(raw.trim()=="/new")return reset()
        if(busy||api.playerRef!=owner)return false
        val content=(if(reply==null)raw else "引用：${reply.text}\n\n$raw").trim()
        if(content.codePointCount(0,content.length)>2000){state.directErrors[key]="问题与引用合计不能超过 2000 字";return false}
        val saved=api.pendingAI(owner)
        if(saved!=null&&saved.getString("content")!=content){state.directErrors[key]="上一条 AI 请求需要先恢复；可输入 /new 开始新对话";return false}
        val pending=saved ?: JSONObject().put("clientMessageId",UUID.randomUUID().toString()).put("content",content).put("draft",raw)
        val request=JSONObject().put("clientMessageId",pending.getString("clientMessageId")).put("content",content)
        lastRequestId=pending.getString("clientMessageId")
        api.savePendingAI(owner,pending);recoveredDraft=raw;busy=true;state.directPending[key]=true;state.directErrors.remove(key);statusText="正在连接 AI…"
        var assistantId=pending.optString("assistantMessageId");var answer="";var responseSources=emptyList<AiSource>();var createdAt=Instant.now().toString()
        fun partial(status:String="streaming"){
            if(assistantId.isBlank())return
            val value=JSONObject().put("messageId",assistantId).put("conversationId",conversationId).put("role","assistant").put("content",answer).put("createdAt",createdAt).put("status",status)
                .put("sources",JSONArray(responseSources.map{JSONObject().put("title",it.title).put("url",it.url).put("origin",it.origin)}))
            put(value)
        }
        return try{
            stream(request){event,data->
                if(api.playerRef!=owner)throw ApiFailure("UNAUTHORIZED","登录账号已变化")
                when(event){
                    "meta"->{val current=data.getString("conversationId");if(conversationId!=current){state.conversation(key).clear();conversationId=current};data.optJSONObject("quota")?.let(::quota);data.optJSONObject("userMessage")?.let{createdAt=it.getString("createdAt");put(it)};assistantId=data.optString("assistantMessageId").takeUnless{it=="null"}.orEmpty();pending.put("assistantMessageId",assistantId);api.savePendingAI(owner,pending);answer=""}
                    "status"->statusText=when(data.optString("status")){"queued"->"请求已受理…";"searching"->"正在查找联网资料…";else->"正在生成回复…"}
                    "delta"->{answer+=data.getString("content");partial()}
                    "sources"->{responseSources=sources(data.optJSONArray("sources"));if(answer.isNotEmpty())partial()}
                    "done"->{data.optJSONObject("message")?.let(::put);data.optJSONObject("quota")?.let(::quota);api.savePendingAI(owner,null);recoveredDraft="";state.directErrors.remove(key)}
                    "error"->{val value=data.getJSONObject("error");throw ApiFailure(value.optString("code","AI_RESULT_UNKNOWN"),value.optString("message","AI 回复未完成"))}
                }
            };true
        }catch(failure:Exception){
            val code=(failure as? ApiFailure)?.code
            if(code in setOf("AI_DISABLED","AI_INVALID_MESSAGE","AI_QUOTA_EXCEEDED","AI_PROVIDER_UNAVAILABLE","AI_SERVER_BUSY","AI_REQUEST_CONFLICT","AI_CONVERSATION_CHANGED","UNAUTHORIZED"))api.savePendingAI(owner,null)
            if(assistantId.isNotBlank()&&answer.isNotEmpty())partial(if(code=="AI_RESPONSE_INCOMPLETE")"incomplete" else "unknown")
            state.directErrors[key]=failure.message ?: "AI 连接已中断，可恢复原请求";false
        }finally{busy=false;state.directPending[key]=false}
    }
    private suspend fun stream(input:JSONObject,onEvent:(String,JSONObject)->Unit):Unit=suspendCancellableCoroutine{continuation->
        val credential=api.token ?: run{continuation.resumeWithException(ApiFailure("UNAUTHORIZED","请重新登录"));return@suspendCancellableCoroutine}
        val call=http.newCall(Request.Builder().url(api.baseUrl+"/api/v1/ai/chat/stream").header("Authorization","Bearer $credential").header("Accept","text/event-stream")
            .post(input.toString().toRequestBody("application/json; charset=utf-8".toMediaType())).build())
        continuation.invokeOnCancellation{call.cancel()}
        call.enqueue(object:Callback{
            override fun onFailure(call:Call,e:IOException){if(continuation.isActive)continuation.resumeWithException(ApiFailure("AI_STREAM_INTERRUPTED","AI 连接中断，可使用原请求恢复"))}
            override fun onResponse(call:Call,response:Response){
                try{response.use{
                    if(!it.isSuccessful){val error=runCatching{JSONObject(it.body?.string().orEmpty()).getJSONObject("error")}.getOrNull();if(it.code==401)runBlocking{withContext(Dispatchers.Main){if(api.token==credential)api.forgetSession()}};throw ApiFailure(error?.optString("code") ?: "HTTP_${it.code}",error?.optString("message") ?: "AI 服务暂不可用",it.code)}
                    require(it.header("Content-Type").orEmpty().contains("text/event-stream")){"AI 响应类型不正确"}
                    readAiEvents(it.body?.charStream() ?: throw IOException("empty response")){event,value->runBlocking{withContext(Dispatchers.Main){if(continuation.isActive)onEvent(event,value)}}}
                };if(continuation.isActive)continuation.resume(Unit)}catch(error:Exception){if(continuation.isActive)continuation.resumeWithException(error)}
            }
        })
    }
}
