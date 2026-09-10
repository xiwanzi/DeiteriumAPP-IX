package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.ContextWrapper
import android.content.Intent
import android.content.SharedPreferences
import android.graphics.Rect
import android.os.Bundle
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.*
import okhttp3.mockwebserver.*
import org.json.*
import java.time.*
import java.util.Collections
import java.util.concurrent.atomic.AtomicBoolean
import kotlin.math.abs

/** Real pages with an isolated HTTP fixture; no live commerce mutations. */
object HistoryLayoutCheck {
    fun run(test:Instrumentation) {
        val result=Bundle();val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val server=MockWebServer();var host:DeuteriumActivity?=null
        val context=object:ContextWrapper(test.targetContext){override fun getSharedPreferences(name:String,mode:Int):SharedPreferences=super.getSharedPreferences("qa-history-$name",mode)}
        try {
            val requests=Collections.synchronizedList(mutableListOf<String>());val failNext=AtomicBoolean(false)
            val now=Instant.now().minusSeconds(90)
            server.dispatcher=object:Dispatcher(){override fun dispatch(request:RecordedRequest):MockResponse {
                check(request.method=="GET");val url=request.requestUrl!!;requests.add(request.path.orEmpty())
                if(url.encodedPath!="/api/v1/wallet/records")return MockResponse().setResponseCode(404)
                if(failNext.compareAndSet(true,false))return MockResponse().setResponseCode(503).setBody("""{"error":{"code":"UNAVAILABLE","message":"账单暂时无法读取"}}""")
                check(url.queryParameter("to")==null){"Current history must not send the device cutoff"}
                val more=url.queryParameter("cursor")!=null
                if(more)check(url.queryParameterNames==setOf("cursor","limit"))
                val direction=url.queryParameter("direction") ?: "expense"
                val row=JSONObject().put("recordId",if(more)"econ_8" else "econ_9").put("title",if(more)"较早流水" else if(direction=="income")"游戏内收入" else "游戏内支出").put("source","GAME").put("status","success").put("direction",direction).put("amount","12.00").put("occurredAt",now.minusSeconds(if(more)60 else 0).toString())
                return MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",JSONObject().put("records",JSONArray().put(row))).put("page",JSONObject().put("nextCursor",if(more)JSONObject.NULL else "fixture_cursor")).toString())
            }}
            server.start();val api=BackendApi(context,server.url("/").toString())
            val save=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            test.runOnMainSync{save.invoke(api,JSONObject().put("token","isolated-history-fixture").put("user",JSONObject().put("gameId","FixtureSelf").put("playerRef","fixture-self")))}
            val state=LabState(scope,userName="FixtureSelf",api=api)
            val book=CommerceBook("FixtureSelf",emptyList(),{0L},{_,_,_->error("No financial mutations")},remoteOnly=true)
            book.orders.add(CommerceOrder("fixture-order","fixture-key",OrderChannel.Official,"FixtureSelf","Deuterium 官方商店","",listOf(OrderLine("fixture-product","测试商品","",1234500,1)),DeliveryMethod.Mailbox,"",LocalDateTime.now(),OrderStage.AwaitingClaim,refund=RefundState.Approved,canHideRecord=true,serverStatus="REFUNDED"))
            var screen by mutableStateOf("history");var mode by mutableIntStateOf(1);var feedback by mutableStateOf(false);var dialog by mutableStateOf(false)
            var attempts=0;var deleted=false
            host=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            val activity=host
            test.runOnMainSync{activity.setContent{LabTheme(mode,false,false){val backdrop=rememberGraphicsLayer();CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalOverlayBackdrop provides backdrop){
                Box(Modifier.fillMaxSize().recordGlassBackdrop(backdrop).background(MaterialTheme.colorScheme.background)) {
                    when(screen) {
                        "history"->BillHistoryPage(state,"all",92.dp){}
                        "orders"->OrdersPage(book,92.dp){}
                    }
                    Text(if(screen=="history")"历史账单" else "我的订单",Modifier.align(Alignment.TopCenter).statusBarsPadding().padding(top=16.dp),style=MaterialTheme.typography.titleLarge)
                    if(dialog)DeleteRecordDialog("测试商品",{dialog=false}){attempts++;delay(100);if(attempts==1)false else {deleted=true;true}}
                    if(feedback)ActionFeedback("退款成功","款项已退回钱包"){feedback=false}
                }
            }}}}
            fun find(node:AccessibilityNodeInfo?,text:String):AccessibilityNodeInfo?{if(node==null)return null;if(node.text?.toString()==text)return node;for(i in 0 until node.childCount)find(node.getChild(i),text)?.let{return it};return null}
            fun node(text:String)=find(test.uiAutomation.rootInActiveWindow,text)
            fun waitFor(label:String,condition:()->Boolean){repeat(100){if(condition())return;Thread.sleep(50)};error("Timeout: $label")}
            fun click(text:String){waitFor(text){node(text)!=null};var target=node(text);while(target!=null&&!target.isClickable)target=target.parent;check(target?.performAction(AccessibilityNodeInfo.ACTION_CLICK)==true){"Cannot click $text"}}
            fun bounds(text:String)=Rect().also{checkNotNull(node(text)){"Missing $text"}.getBoundsInScreen(it)}
            fun screenshot(name:String){test.waitForIdleSync();Thread.sleep(500);test.uiAutomation.takeScreenshot().let{bitmap->test.targetContext.filesDir.resolve("history-$name.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}}
            waitFor("history loaded"){node("游戏内支出")!=null};click("加载更多账单");waitFor("second page"){node("较早流水")!=null}
            check(bounds("游戏内支出").top<bounds("较早流水").top)
            click("今天");waitFor("today"){node("游戏内支出")!=null};click("收入");waitFor("income"){node("游戏内收入")!=null}
            screenshot("light-bills");click("支出");waitFor("expense"){node("游戏内支出")!=null}
            failNext.set(true);click("近7天");waitFor("read failure"){node("重新读取")!=null}
            check(node("0 笔交易 · 北京时间")==null){"Failure must not claim empty financial history"}
            click("重新读取");waitFor("retry"){node("游戏内支出")!=null}
            result.putString("history_dates_filters_pagination_failure_retry","PASS")
            for(theme in listOf(1,2)) {
                test.runOnMainSync{mode=theme;screen="orders"};waitFor("orders"){node("删除记录")!=null};Thread.sleep(160);screenshot(if(theme==1)"light-orders" else "dark-orders")
                click("删除记录");waitFor("delete dialog"){node("确认")!=null}
                check(node("删除记录")==null){"Dialog contains a nested page delete button"}
                val cancel=bounds("取消");val confirm=bounds("确认");check(abs(cancel.centerY()-confirm.centerY())<=2)
                screenshot(if(theme==1)"light-confirm" else "dark-confirm");click("取消");waitFor("cancel preserves order"){node("测试商品")!=null&&node("确认")==null};check(book.orders.size==1)
                test.runOnMainSync{feedback=true};waitFor("feedback"){node("退款成功")!=null}
                val title=bounds("退款成功");val detail=bounds("款项已退回钱包")
                Thread.sleep(140);val bitmap=test.uiAutomation.takeScreenshot();check(abs(title.centerX()-bitmap.width/2)<=2);check(abs(detail.centerX()-bitmap.width/2)<=2)
                test.targetContext.filesDir.resolve("history-${if(theme==1)"light" else "dark"}-feedback.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()
                waitFor("feedback completes"){node("退款成功")==null}
            }
            test.runOnMainSync{dialog=true};waitFor("isolated delete"){node("确认")!=null};click("确认");waitFor("delete failure"){node("暂时无法删除，记录可能仍需处理。请刷新后重试。")!=null};check(!deleted)
            click("确认");waitFor("delete retry"){deleted&&node("确认")==null};check(attempts==2)
            result.putString("delete_cancel_failure_retry_and_two_plain_actions","PASS")
            result.putString("light_dark_full_pages_and_centered_feedback","PASS")
            test.finish(-1,result)
        }catch(error:Throwable){result.putString("error",error.stackTraceToString());test.finish(1,result)}
        finally{scope.cancel();test.runOnMainSync{host?.finish()};server.shutdown();context.getSharedPreferences("backend-v2",0).edit().clear().commit()}
    }
}
