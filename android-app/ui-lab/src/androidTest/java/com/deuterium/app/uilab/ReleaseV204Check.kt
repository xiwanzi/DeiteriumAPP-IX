package com.deuterium.app.uilab

import android.app.Instrumentation
import android.app.UiModeManager
import android.content.ContextWrapper
import android.content.Intent
import android.content.SharedPreferences
import android.os.Build
import android.os.Bundle
import android.graphics.Rect
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.*
import okhttp3.mockwebserver.*
import org.json.*
import java.time.Instant
import java.util.Collections
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.TimeUnit

/** Isolated HTTP fixtures and real Compose/Activity configuration behavior. */
object ReleaseV204Check {
    fun run(test:Instrumentation) {
        val result=Bundle();val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate);val server=MockWebServer()
        val fixtureContext=object:ContextWrapper(test.targetContext){override fun getSharedPreferences(name:String,mode:Int):SharedPreferences=super.getSharedPreferences("qa-v204-$name",mode)}
        var host:MainActivity?=null
        val previousTheme=test.targetContext.getSharedPreferences("ui-lab",0).getInt("theme",1)
        try {
            val requests=Collections.synchronizedList(mutableListOf<String>())
            val delayRecord=AtomicBoolean(false)
            fun ledger(id:Int,time:String)=JSONObject().put("recordId","econ_$id").put("title",if(id==9)"游戏内支出" else "流水 $id").put("source",if(id==9)"GAME" else "APP").put("status","success").put("direction","expense").put("amount","1.00").put("occurredAt",time)
            val at=Instant.now()
            server.dispatcher=object:Dispatcher(){override fun dispatch(request:RecordedRequest):MockResponse{
                requests.add(request.path.orEmpty());check(request.method=="GET"){"No money or business mutations in this fixture"}
                val data=when(request.requestUrl!!.encodedPath){
                    "/api/v1/wallet/balance"->JSONObject().put("balance",JSONObject().put("amount","100.00").put("heldAmount","0.00").put("fresh",true).put("today",JSONObject().put("income","20.00").put("expense","9.00")))
                    "/api/v1/wallet/records"->JSONObject().put("records",JSONArray().put(ledger(1,at.minusSeconds(300).toString())).put(ledger(9,at.toString())).put(ledger(3,at.minusSeconds(60).toString())))
                    else->return MockResponse().setResponseCode(404)
                }
                return MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",data).put("page",JSONObject().put("nextCursor",JSONObject.NULL)).toString()).apply{if(request.requestUrl!!.encodedPath=="/api/v1/wallet/records"&&delayRecord.compareAndSet(true,false))setBodyDelay(500,TimeUnit.MILLISECONDS)}
            }}
            server.start();val api=BackendApi(fixtureContext,server.url("/").toString())
            val remember=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            test.runOnMainSync{remember.invoke(api,JSONObject().put("token","v204-fixture-token").put("user",JSONObject().put("gameId","FixtureSelf").put("playerRef","fixture-self")))}
            val state=LabState(scope,userName="FixtureSelf",api=api)
            host=test.startActivitySync(Intent(test.targetContext,MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as MainActivity
            val activity=host
            var theme by mutableIntStateOf(1);var screen by mutableStateOf("wallet");var compositions=0
            test.runOnMainSync{activity.setContent{remember{compositions++;Any()};LabTheme(theme,false,false){CompositionLocalProvider(androidx.compose.material3.LocalContentColor provides MaterialTheme.colorScheme.onSurface){Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)){
                when(screen){"wallet"->WalletPage(state,{},{},32.dp);"delete"->Column(Modifier.fillMaxSize().padding(24.dp)){Text("订单记录",style=MaterialTheme.typography.headlineLarge);DeleteRecordButton({},Modifier.padding(top=24.dp));DeleteRecordButton({},Modifier.fillMaxWidth().padding(top=24.dp))}}
            }}}}}
            fun find(node:AccessibilityNodeInfo?,name:String):AccessibilityNodeInfo?{if(node==null)return null;if(node.text?.toString()==name)return node;for(i in 0 until node.childCount)find(node.getChild(i),name)?.let{return it};return null}
            fun waitFor(condition:()->Boolean){repeat(70){if(condition())return;Thread.sleep(100)};error("UI did not reach expected state")}
            fun node(name:String)=find(test.uiAutomation.rootInActiveWindow,name)
            fun top(name:String)=Rect().also{checkNotNull(node(name)){"Missing $name"}.getBoundsInScreen(it)}.top
            waitFor{state.ledgerKnown&&!state.refreshing&&node("游戏内支出")!=null}
            check(state.ledger.map{it.id}==listOf(9L,3L,1L));check(state.todayIncome==2000L&&state.todayExpense==900L)
            check(top("游戏内支出")<top("流水 3"));check(requests.count{it.startsWith("/api/v1/wallet/records") }==1){"Recent wallet must not download every history page"}
            result.putString("recent_ledger_order_game_labels_and_daily_totals","PASS")
            delayRecord.set(true);test.runOnMainSync{state.refresh()};waitFor{requests.count{it.startsWith("/api/v1/wallet/records")}==2}
            test.runOnMainSync{state.refresh()};waitFor{!state.refreshing&&requests.count{it.startsWith("/api/v1/wallet/records")}==3}
            result.putString("refresh_after_transfer_is_not_lost_during_inflight_read","PASS")
            fun screenshot(name:String){test.waitForIdleSync();test.uiAutomation.takeScreenshot().let{bitmap->test.targetContext.filesDir.resolve("$name.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}}
            screenshot("v204-wallet-light")
            for(mode in listOf(2,1,2)){
                test.runOnMainSync{theme=mode;screen="delete";if(Build.VERSION.SDK_INT>=31)activity.getSystemService(UiModeManager::class.java).setApplicationNightMode(if(mode==2)UiModeManager.MODE_NIGHT_YES else UiModeManager.MODE_NIGHT_NO)}
                waitFor{node("订单记录")!=null};test.waitForIdleSync();check(!activity.isDestroyed&&compositions==1){"Theme switch recreated the Activity or composition"}
                screenshot(if(mode==2)"v204-delete-dark" else "v204-delete-light")
            }
            result.putString("theme_switch_retains_activity_and_composition","PASS");result.putString("delete_buttons_light_dark_render","PASS")
            test.finish(-1,result)
        }catch(error:Throwable){runCatching{test.uiAutomation.takeScreenshot().let{bitmap->test.targetContext.filesDir.resolve("v204-failure.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}};result.putString("error",error.stackTraceToString());test.finish(1,result)}
        finally{
            scope.cancel();test.runOnMainSync{if(Build.VERSION.SDK_INT>=31)test.targetContext.getSystemService(UiModeManager::class.java).setApplicationNightMode(when(previousTheme){1->UiModeManager.MODE_NIGHT_NO;2->UiModeManager.MODE_NIGHT_YES;else->UiModeManager.MODE_NIGHT_AUTO});host?.finish()};server.shutdown();fixtureContext.getSharedPreferences("backend-v2",0).edit().clear().commit()
        }
    }
}
