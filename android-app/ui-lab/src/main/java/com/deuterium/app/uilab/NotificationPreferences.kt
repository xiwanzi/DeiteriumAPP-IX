package com.deuterium.app.uilab

import android.content.Context
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.*
import java.security.MessageDigest
import java.util.UUID
import org.json.JSONObject

enum class NotificationTopic(val key:String,val title:String,val detail:String) {
    Direct("directMessages","私聊消息","其他玩家发来的私信"),
    Mentions("mentions","有人提及我","公共频道中的 @ 消息"),
    Followed("followedPlayers","特别关心","关注玩家的新消息"),
    Wallet("wallet","钱包与转账","转账到账与资金状态"),
    Market("marketOrders","商城与市场订单","履约、退款处理与结算"),
    Commissions("commissions","委托进度","接取、完成、协商与报酬到账"),
    Announcements("announcements","官方公告","服务器维护与重要消息"),
    Updates("appUpdates","应用与资源更新","新版本和可用资源包")
}
class NotificationPreferences(context:Context,userName:String) {
    private val api=BackendApi.get(context)
    private val suffix=MessageDigest.getInstance("SHA-256").digest(userName.toByteArray()).take(8).joinToString(""){"%02x".format(it)}
    private val prefs=context.getSharedPreferences("notification-preferences-$suffix",Context.MODE_PRIVATE)
    val values=mutableStateMapOf<String,Boolean>().apply{(listOf("enabled","showPreviews")+NotificationTopic.entries.map{it.key}).forEach{put(it,prefs.getBoolean(it,true))}}
    fun get(key:String)=values[key] ?: true
    fun set(key:String,value:Boolean){values[key]=value;prefs.edit().putBoolean(key,value).apply()}
    fun permits(topic:NotificationTopic)=get("enabled")&&get(topic.key)
    var error by mutableStateOf<String?>(null);private set
    var busy by mutableStateOf(false);private set
    private var version=0L
    private fun apply(value:JSONObject) {
        val edit=prefs.edit()
        try { values.keys.toList().forEach{key->if(value.has(key)){val next=value.getBoolean(key);values[key]=next;edit.putBoolean(key,next)}} }
        finally { edit.apply() }
        version=value.getLong("version");error=null
    }
    suspend fun sync(){runCatching{api.request("GET","/notifications/preferences")}.onSuccess(::apply).onFailure{error=it.message}}
    suspend fun update(key:String,value:Boolean){
        if(busy)return
        busy=true
        runCatching{val current=api.request("GET","/notifications/preferences");apply(current);api.request("PATCH","/notifications/preferences",JSONObject().put("clientRequestId",UUID.randomUUID().toString()).put("expectedVersion",version).put(key,value))}
            .onSuccess(::apply).onFailure{error=it.message}
        busy=false
    }
}
