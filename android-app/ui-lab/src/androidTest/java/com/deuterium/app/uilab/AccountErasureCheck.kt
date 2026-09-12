package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.ContextWrapper
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.*
import okhttp3.Protocol
import okhttp3.Response
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import org.json.JSONObject
import org.json.JSONArray
import java.time.LocalDateTime
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

/** Synthetic accounts and intercepted HTTP only; never calls a live delete endpoint. */
object AccountErasureCheck {
    fun run(test:Instrumentation){
        val result=Bundle();val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val instance=BackendApi::class.java.getDeclaredField("instance").apply{isAccessible=true};val previous=instance.get(null)
        val oldPlayers=Players.toList();val stage=AtomicInteger(0);val entered=CountDownLatch(1);val release=CountDownLatch(1)
        val context=object:ContextWrapper(test.targetContext){override fun getSharedPreferences(name:String,mode:Int)=super.getSharedPreferences("qa-account-erasure-$name",mode)}
        try{
            context.getSharedPreferences("backend-v2",0).edit().clear().commit()
            val api=BackendApi(context)
            fun profile(name:String,ref:String)=JSONObject().put("gameId",name).put("playerRef",ref).put("qq","10001").put("bio","private old profile").put("version",1)
            fun conversation(name:String,ref:String,id:String)=JSONObject().put("conversationId",id).put("otherPlayer",profile(name,ref)).put("unreadCount",1)
                .put("lastMessage",JSONObject().put("messageId","message-$ref").put("sender",profile(name,ref)).put("content","message-$name").put("sentAt","2026-09-12T00:00:00Z"))
            val client=api.http.newBuilder().addInterceptor{chain->
                val path=chain.request().url.encodedPath
                val data:Any=when{
                    path.endsWith("/account/deletions")->JSONObject().put("cursor",stage.get()).put("hasMore",false).put("items",if(stage.get()==1&&chain.request().url.queryParameter("after")=="0")JSONArray().put(JSONObject().put("sequence",1).put("playerRef","erased-ref"))else JSONArray())
                    path.endsWith("/chat/conversations")->JSONArray().apply{if(stage.get()==0)put(conversation("OldPlayer","erased-ref","old-conv"));put(conversation("KeepPlayer","keep-ref","keep-conv"))}
                    path.endsWith("/chat/player-directory")||path.endsWith("/chat/follows")->JSONObject().put("players",JSONArray().apply{if(stage.get()==0)put(profile("OldPlayer","erased-ref"));put(profile("KeepPlayer","keep-ref"))})
                    path.endsWith("/players/erased-ref")->{entered.countDown();check(release.await(15,TimeUnit.SECONDS));profile("OldPlayer","erased-ref")}
                    else->JSONObject()
                }
                Response.Builder().request(chain.request()).protocol(Protocol.HTTP_1_1).code(200).message("OK").body(JSONObject().put("data",data).toString().toResponseBody("application/json".toMediaType())).build()
            }.build()
            BackendApi::class.java.getDeclaredField("http").apply{isAccessible=true}.set(api,client);instance.set(null,api)
            val activity=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            lateinit var state:LabState
            test.runOnMainSync{
                activity.setContent{Text("本机注销同步验收")}
                BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}.invoke(api,JSONObject().put("token","synthetic-session").put("user",profile("Observer","observer-ref").put("userId","observer-user")))
                Players.clear();state=LabState(scope,initialFollowed=setOf("OldPlayer","KeepPlayer"),userName="Observer",api=api)
                state.commerce.listings.add(MarketListing("old-listing","Old item","subtitle","description","其他",100,1,"OldPlayer","10001",setOf(DeliveryMethod.Pickup),"place",sellerRef="erased-ref"))
                state.commerce.listings.add(MarketListing("keep-listing","Keep item","subtitle","description","其他",100,1,"KeepPlayer","10002",setOf(DeliveryMethod.Pickup),"place",sellerRef="keep-ref"))
                state.chat.add(ChatLine(1,"OldPlayer","old public content",false,remoteId="old-public"))
                state.chat.add(ChatLine(2,"KeepPlayer","keep public",false,reply=ChatReply(1,"OldPlayer","old quote"),remoteId="keep-public"))
            }
            runBlocking{withContext(Dispatchers.Main){state.refreshContacts();api.savePendingDirect("erased-ref",JSONObject().put("clientMessageId","pending").put("content","pending old message"))}}
            check(Players.any{it.playerRef=="erased-ref"})
            val late=scope.async{state.loadProfile("OldPlayer")};check(entered.await(5,TimeUnit.SECONDS))
            stage.set(1)
            runBlocking{withContext(Dispatchers.Main){state.refreshContacts()}}
            release.countDown();runBlocking{late.await()}
            test.runOnMainSync{
                check(Players.none{it.playerRef=="erased-ref"}){"Late profile recreated a deleted player"}
                check("OldPlayer" !in state.directChats&&"KeepPlayer" in state.directChats)
                check(state.followed.toSet()==setOf("KeepPlayer"))
                check(state.commerce.listings.map{it.id}==listOf("keep-listing"))
                check(state.chat.none{it.name=="OldPlayer"}&&state.chat.single().reply?.text=="原消息不可见")
                check(state.isUnavailableAccount("OldPlayer"))
            }
            check(api.pendingDirect("erased-ref")==null)
            check(BackendApi(context).erasedPlayerRefs()==setOf("erased-ref"))
            check(BackendApi(context).accountDeletionCursor()==1L)
            fun find(node:AccessibilityNodeInfo?,text:String):Boolean=node!=null&&(node.text?.toString()==text||(0 until node.childCount).any{find(node.getChild(it),text)})
            fun render(label:String,content:@Composable ()->Unit){
                test.runOnMainSync{activity.setContent{LabTheme(1,false,false){CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface){content()}}}}
                test.waitForIdleSync();Thread.sleep(500)
                test.uiAutomation.takeScreenshot()?.let{image->test.targetContext.filesDir.resolve("qa-erasure-$label.png").outputStream().use{image.compress(Bitmap.CompressFormat.PNG,100,it)};image.recycle()}
            }
            render("contacts"){InfoPage(state,36.dp,""){} }
            check(!find(test.uiAutomation.rootInActiveWindow,"OldPlayer")&&find(test.uiAutomation.rootInActiveWindow,"KeepPlayer"))
            render("profile"){PlayerProfilePage("OldPlayer",state,null,36.dp,{},{})}
            check(find(test.uiAutomation.rootInActiveWindow,"该账号已注销"))
            render("conversation"){DirectChatPage(state,"OldPlayer",36.dp){}}
            check(find(test.uiAutomation.rootInActiveWindow,"该账号已注销"))
            render("unresolved-profile"){PlayerProfilePage("OldNotificationLink",state,null,36.dp,{},{})}
            repeat(30){if(!find(test.uiAutomation.rootInActiveWindow,"用户资料不可用"))Thread.sleep(100)}
            check(find(test.uiAutomation.rootInActiveWindow,"用户资料不可用")&&!find(test.uiAutomation.rootInActiveWindow,"OldNotificationLink"))
            result.putString("contact_follow_listing_message_and_pending_cleanup","PASS")
            result.putString("delayed_profile_cannot_revive_and_markers_persist","PASS")
            result.putString("open_profile_and_conversation_unavailable","PASS")
            test.finish(-1,result)
        }catch(error:Throwable){result.putString("error",error.stackTraceToString());test.finish(1,result)}
        finally{release.countDown();scope.cancel();instance.set(null,previous);test.runOnMainSync{Players.clear();Players.addAll(oldPlayers)}}
    }
}
