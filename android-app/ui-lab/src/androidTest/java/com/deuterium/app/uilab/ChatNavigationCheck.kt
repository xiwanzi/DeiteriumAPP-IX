package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.ContextWrapper
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import android.view.KeyEvent
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.dp
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import kotlinx.coroutines.*
import okhttp3.Protocol
import okhttp3.Response
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import org.json.JSONArray
import org.json.JSONObject
import java.util.concurrent.atomic.AtomicInteger

/** Isolated session and intercepted requests; never contacts production accounts. */
object ChatNavigationCheck {
    fun run(test:Instrumentation){
        val result=Bundle()
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val instance=BackendApi::class.java.getDeclaredField("instance").apply{isAccessible=true}
        val previous=instance.get(null);val previousPlayers=Players.toList()
        val fixture=AtomicInteger(0);val requests=AtomicInteger(0)
        val context=object:ContextWrapper(test.targetContext){override fun getSharedPreferences(name:String,mode:Int)=super.getSharedPreferences("qa-chat-navigation-$name",mode)}
        try{
            context.getSharedPreferences("backend-v2",0).edit().clear().commit()
            val api=BackendApi(context)
            fun player(name:String,registered:Boolean=true)=JSONObject().put("gameId",name).put("playerRef","ref-$name").put("registered",registered).put("online",true)
            val client=api.http.newBuilder().addInterceptor{chain->
                val online=chain.request().url.encodedPath.endsWith("/chat/online-players")
                val stage=fixture.get()
                if(online){check(chain.request().method=="GET");requests.incrementAndGet()}
                val players=JSONArray().apply{
                    if(stage==0){put(player("Observer"));put(player("Fresh"));put(player("Guest",false))}
                    if(stage==4)repeat(40){put(player("Player${it.toString().padStart(2,'0')}"))}
                }
                val data=if(online)JSONObject().put("available",stage!=3).put("onlineCount",players.length()).put("players",players) else JSONObject()
                val body=if(online&&stage==2)JSONObject().put("error",JSONObject().put("code","UNAVAILABLE").put("message","读取失败，请重试")) else JSONObject().put("data",data)
                Response.Builder().request(chain.request()).protocol(Protocol.HTTP_1_1).code(if(online&&stage==2)503 else 200).message("fixture")
                    .body(body.toString().toResponseBody("application/json".toMediaType())).build()
            }.build()
            BackendApi::class.java.getDeclaredField("http").apply{isAccessible=true}.set(api,client);instance.set(null,api)
            val activity=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            lateinit var state:LabState
            test.runOnMainSync{
                activity.setContent{Text("聊天回归检查")}
                BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}.invoke(api,JSONObject().put("token","fixture-chat").put("user",player("Observer").put("userId","fixture-observer")))
                Players.clear();Players.add(PlayerProfile("Stale",online=true,playerRef="ref-Stale"))
                state=LabState(scope,userName="Observer",api=api)
            }
            fun refresh(stage:Int){fixture.set(stage);runBlocking{withContext(Dispatchers.Main){state.refreshOnlinePlayers()}}}
            refresh(0)
            check(state.onlinePlayers.map{it.name}==listOf("Observer","Fresh","Guest"))
            check(state.onlineCount==3&&state.onlinePlayersKnown&&!state.onlinePlayers.last().registered)
            test.runOnMainSync{Players.replaceAll{it.copy(online=false)}}
            check(state.onlinePlayers.size==3){"Profile cache overwrote live list"}
            refresh(1);check(state.onlinePlayersKnown&&state.onlinePlayers.isEmpty()&&state.onlineCount==0)
            refresh(2);check(state.onlinePlayersError!=null&&!state.onlinePlayersKnown&&state.onlineCount==null)
            refresh(3);check(state.onlinePlayersError=="服务器在线状态暂不可用"&&!state.onlinePlayersKnown)
            refresh(0);check(state.onlinePlayersError==null&&state.onlinePlayers.size==3)
            result.putString("live_list_empty_unavailable_failure_recovery","PASS")

            fun find(node:AccessibilityNodeInfo?,predicate:(AccessibilityNodeInfo)->Boolean):AccessibilityNodeInfo?{
                if(node==null)return null
                if(predicate(node))return node
                for(i in 0 until node.childCount)find(node.getChild(i),predicate)?.let{return it}
                return null
            }
            fun named(text:String)=find(test.uiAutomation.rootInActiveWindow){it.text?.toString()==text||it.contentDescription?.toString()==text}
            fun waitFor(predicate:()->Boolean){repeat(150){if(predicate())return;Thread.sleep(20)};error("Chat UI did not reach expected state")}
            fun click(text:String){waitFor{named(text)!=null};var node=named(text)!!;while(!node.isClickable&&node.parent!=null)node=node.parent;check(node.performAction(AccessibilityNodeInfo.ACTION_CLICK))}
            fun capture(label:String){test.waitForIdleSync();Thread.sleep(400);test.uiAutomation.takeScreenshot()?.let{bitmap->test.targetContext.filesDir.resolve("qa-chat-$label.png").outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}}
            val showing=mutableStateOf(true)
            var sheetRevision=0
            fun sheet(theme:Int){val revision=++sheetRevision;test.runOnMainSync{showing.value=true;activity.setContent{key(revision){LabTheme(theme,false,false){CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface){if(showing.value)OnlinePlayersSheet(state,{showing.value=false}){} else Text("已关闭")}}}}};test.waitForIdleSync()}
            sheet(1);waitFor{named("Fresh")!=null&&named("Guest")!=null};capture("online-light")
            check(named("Stale")==null)
            click("@ 提及");waitFor{!showing.value};check(state.chatDraft.text.contains("@Fresh "))
            fixture.set(2);sheet(2);waitFor{named("重试")!=null};capture("error-dark")
            fixture.set(0);click("重试");waitFor{named("Fresh")!=null};capture("online-dark")
            test.runOnMainSync{showing.value=false};test.waitForIdleSync()
            fixture.set(1);sheet(1);waitFor{named("当前没有玩家在线")!=null};capture("empty")
            test.runOnMainSync{showing.value=false};test.waitForIdleSync()
            fixture.set(4);sheet(1);waitFor{named("Player00")!=null}
            var reachedEnd=false
            repeat(20){if(named("Player39")!=null)reachedEnd=true else {find(test.uiAutomation.rootInActiveWindow){it.isScrollable}?.performAction(AccessibilityNodeInfo.ACTION_SCROLL_FORWARD);Thread.sleep(300)}}
            capture("long-list")
            check(reachedEnd||named("Player39")!=null){"Long online list cannot scroll to last player; scrollable=${find(test.uiAutomation.rootInActiveWindow){it.isScrollable}!=null}"}
            test.runOnMainSync{showing.value=false};test.waitForIdleSync()
            result.putString("sheet_light_dark_retry_empty_mention_long_scroll","PASS")

            val page=mutableStateOf("聊天页");val imeHeight=AtomicInteger(0)
            fun imeVisible():Boolean{var visible=false;test.runOnMainSync{visible=ViewCompat.getRootWindowInsets(activity.window.decorView)?.isVisible(WindowInsetsCompat.Type.ime())==true};return visible}
            test.runOnMainSync{activity.setContent{LabTheme(1,false,false){
                var draft by remember{mutableStateOf("")}
                val density=LocalDensity.current;val height=WindowInsets.ime.getBottom(density)
                SideEffect{imeHeight.set(height)}
                NavigationBackHandler(true){page.value="信息页"}
                Column(Modifier.fillMaxSize().statusBarsPadding().imePadding().padding(24.dp)){
                    Text(page.value)
                    if(page.value=="聊天页")OutlinedTextField(draft,{draft=it},label={Text("消息输入框")})
                }
            }}}
            test.waitForIdleSync()
            var callbacksWhileIme=true
            for(gap in listOf(0L,30L,100L,500L)){
                test.runOnMainSync{page.value="聊天页"};test.waitForIdleSync()
                click("消息输入框");waitFor{imeVisible()&&imeHeight.get()>0}
                test.runOnMainSync{callbacksWhileIme=callbacksWhileIme&&activity.onBackPressedDispatcher.hasEnabledCallbacks()}
                test.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
                if(gap>0)Thread.sleep(gap)
                test.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
                waitFor{!imeVisible()}
                check(!activity.isFinishing&&!activity.isDestroyed){"Rapid back finished the activity at ${gap}ms"}
                // Some IMEs consume both very fast events. Once hidden the app
                // must navigate, even if an IME animation still reports height.
                if(page.value=="聊天页")test.runOnMainSync{activity.onBackPressedDispatcher.onBackPressed()}
                waitFor{page.value=="信息页"}
            }
            check(callbacksWhileIme){"Page Back callback was disabled while keyboard was visible"}
            result.putString("back_0_30_100_500ms_no_activity_exit","PASS")
            result.putString("page_callback_stays_registered_during_ime","PASS")
            result.putInt("online_requests",requests.get())
            test.finish(-1,result)
        }catch(error:Throwable){result.putString("error",error.stackTraceToString());test.finish(1,result)}
        finally{scope.cancel();test.runOnMainSync{Players.clear();Players.addAll(previousPlayers)};instance.set(null,previous)}
    }
}
