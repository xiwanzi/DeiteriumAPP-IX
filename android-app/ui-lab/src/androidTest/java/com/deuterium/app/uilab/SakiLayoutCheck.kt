package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.*
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
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.*
import kotlinx.coroutines.*
import okhttp3.mockwebserver.*
import org.json.*
import java.time.Instant
import java.util.concurrent.atomic.AtomicInteger

object SakiLayoutCheck {
    fun run(test:Instrumentation){
        val result=Bundle();val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate);val server=MockWebServer();var host:MainActivity?=null
        val context=object:ContextWrapper(test.targetContext){override fun getSharedPreferences(name:String,mode:Int):SharedPreferences=super.getSharedPreferences("qa-saki-$name",mode)}
        try{
            val purchases=AtomicInteger();val now=Instant.now().minusSeconds(90);val expiry=Instant.now().plusSeconds(7L*86400+12*3600).toString()
            fun plan(code:String,name:String,price:String,quota:Int)=JSONObject().put("planId","plan_$code").put("code",code).put("name",name).put("description",if(code=="free")"日常问答" else "更多问答额度，让灵感随时延续。").put("price",price).put("quotaPerWindow",quota).put("windowHours",24).put("durationDays",if(code=="free")0 else 30).put("purchasable",code!="free").put("version",2)
            val free=plan("free","基础套餐","0.00",20);val plus=plan("pro","小祥 Plus","12.50",80);val ultra=plan("ultra","小祥 Ultra","42.50",200)
            val order=JSONObject().put("orderId","order_saki_fixture").put("orderNo","S20260909").put("orderType","AI_SUBSCRIPTION").put("channel","OFFICIAL_STORE").put("buyer",JSONObject().put("displayName","FixtureSelf")).put("seller",JSONObject().put("displayName","Saki AI")).put("delivery",JSONObject().put("method","DIGITAL")).put("items",JSONArray().put(JSONObject().put("productId","plan_pro").put("title","小祥 Plus").put("subtitle","80 次 / 24 小时 · 30 天").put("unitPrice","12.50").put("quantity",1))).put("amount","12.50").put("status","CONFIRMED").put("fundsStatus","SETTLED").put("version",3).put("availableActions",JSONArray()).put("createdAt",now.toString()).put("confirmedAt",now.toString()).put("aiExpiresAt",expiry)
            server.dispatcher=object:Dispatcher(){override fun dispatch(request:RecordedRequest):MockResponse{
                val path=request.requestUrl!!.encodedPath.removePrefix("/api/v1")
                val data=when(path){
                    "/ai/plans"->JSONObject().put("plans",JSONArray().put(free).put(plus).put(ultra))
                    "/ai/me"->JSONObject().put("assistantName","客服小祥").put("plan",if(purchases.get()>1)ultra else if(purchases.get()>0)plus else free).put("expiresAt",if(purchases.get()>0)expiry else JSONObject.NULL).put("conversation",JSONObject().put("conversationId","aic_fixture")).put("quota",JSONObject().put("limit",if(purchases.get()>0)80 else 20).put("remaining",if(purchases.get()>0)80 else 20).put("used",0).put("windowHours",24).put("resetsAt",now.plusSeconds(86400).toString()))
                    "/ai/messages"->JSONObject().put("messages",JSONArray())
                    "/ai/purchase-quotes"->{val body=JSONObject(request.body.readUtf8());val upgrade=body.getString("planId")=="plan_ultra";JSONObject().put("quoteId",if(upgrade)"quote_upgrade" else "quote_new").put("plan",if(upgrade)ultra else plus).put("kind",if(upgrade)"UPGRADE" else "PURCHASE").put("totalAmount",if(upgrade)"8.00" else "12.50").put("currency","CREDIT").put("previousPrice","12.50").put("priceDifference","30.00").put("remainingDays",if(upgrade)8 else 0).put("entitlementExpiresAt",if(upgrade)expiry else JSONObject.NULL).put("expiresAt",Instant.now().plusSeconds(90).toString())}
                    "/ai/purchases"->{check(request.method=="POST");val body=JSONObject(request.body.readUtf8());check(body.getLong("expectedPlanVersion")==2L&&body.has("quoteId"));if(purchases.incrementAndGet()>1){check(body.getString("quoteId")=="quote_upgrade");order.put("amount","8.00");order.getJSONArray("items").getJSONObject(0).put("unitPrice","8.00").put("title","小祥 Ultra")};JSONObject().put("operation",JSONObject().put("operationId","op_saki").put("resourceId","order_saki_fixture").put("status","COMPLETED")).put("order",order)}
                    "/orders/order_saki_fixture"->order
                    "/wallet/records"->JSONObject().put("records",JSONArray().put(JSONObject().put("recordId","econ_1").put("status","success").put("direction","income").put("amount","2047092.50").put("title","信用点收入").put("occurredAt",now.toString())).put(JSONObject().put("recordId","econ_2").put("status","success").put("direction","expense").put("amount","2059444.99").put("title","商城交易").put("occurredAt",now.toString())))
                    else->return MockResponse().setResponseCode(404).setBody("{}")
                }
                return MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",data).toString())
            }}
            server.start();val api=BackendApi(context,server.url("/").toString())
            val save=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            test.runOnMainSync{save.invoke(api,JSONObject().put("token","isolated-saki-fixture").put("user",JSONObject().put("gameId","FixtureSelf").put("playerRef","fixture-self")))}
            val state=LabState(scope,userName="FixtureSelf",api=api)
            var page by mutableStateOf("history");var mode by mutableIntStateOf(1);var font by mutableFloatStateOf(1f);var contacted=false;var animated by mutableStateOf(true)
            host=test.startActivitySync(Intent(test.targetContext,MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as MainActivity
            val activity=host
            test.runOnMainSync{activity.setContent{LabTheme(mode,animated,false){val backdrop=rememberGraphicsLayer();val density=LocalDensity.current
                CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalOverlayBackdrop provides backdrop,LocalDensity provides Density(density.density,font)){
                    Box(Modifier.fillMaxSize().recordGlassBackdrop(backdrop).background(MaterialTheme.colorScheme.background)){
                        when(page){"history"->BillHistoryPage(state,"all",92.dp){};"plans"->SakiPlansPage(state,92.dp);"chat"->DirectChatPage(state,"AI 助手",92.dp,{page="plans"}){};"order"->SakiOrderDetail(state.commerce,state.commerce.orders.single(),92.dp,{contacted=true},{})}
                        Surface(Modifier.align(Alignment.TopCenter).fillMaxWidth().height(92.dp),color=MaterialTheme.colorScheme.background){Box(contentAlignment=Alignment.Center){Text(if(page=="history")"历史账单" else if(page=="plans")"套餐与额度" else if(page=="chat")"小祥 AI" else "订单详情",Modifier.statusBarsPadding(),style=MaterialTheme.typography.titleLarge)}}
                    }
                }
            }}}
            fun find(node:AccessibilityNodeInfo?,text:String):AccessibilityNodeInfo?{if(node==null)return null;if(node.text?.toString()==text||node.contentDescription?.toString()==text)return node;for(i in 0 until node.childCount)find(node.getChild(i),text)?.let{return it};return null}
            fun node(text:String)=find(test.uiAutomation.rootInActiveWindow,text)
            fun waitFor(label:String,condition:()->Boolean){repeat(150){if(condition())return;Thread.sleep(60)};error("Timeout: $label")}
            fun click(text:String){waitFor(text){node(text)!=null};var n=node(text);while(n!=null&&!n.isClickable)n=n.parent;check(n?.performAction(AccessibilityNodeInfo.ACTION_CLICK)==true){"Cannot click $text"}}
            fun scroll(forward:Boolean){
                fun findScroll(n:AccessibilityNodeInfo?):AccessibilityNodeInfo?{if(n==null)return null;if(n.isScrollable)return n;for(i in 0 until n.childCount)findScroll(n.getChild(i))?.let{return it};return null}
                checkNotNull(findScroll(test.uiAutomation.rootInActiveWindow)).performAction(if(forward)AccessibilityNodeInfo.ACTION_SCROLL_FORWARD else AccessibilityNodeInfo.ACTION_SCROLL_BACKWARD);Thread.sleep(650)
            }
            fun bounds(text:String)=Rect().also{checkNotNull(node(text)).getBoundsInScreen(it)}
            fun screenshot(name:String){test.waitForIdleSync();Thread.sleep(400);val bitmap=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("saki-$name.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}
            waitFor("bill totals"){node("+2,047,092.50")!=null&&node("−2,059,444.99")!=null}
            for(theme in listOf(1,2))for(scale in listOf(1f,1.6f)){
                test.runOnMainSync{mode=theme;font=scale};Thread.sleep(450)
                val a=bounds("+2,047,092.50");val b=bounds("−2,059,444.99");check(!Rect.intersects(a,b)&&a.width()>0&&b.width()>0){"Amounts overlap"}
                screenshot("history-$theme-$scale")
            }
            result.putString("large_totals_light_dark_font_scale","PASS")
            test.runOnMainSync{page="plans";mode=1;font=1f};waitFor("plans"){node("小祥 Plus")!=null};screenshot("plans-light")
            click("小祥 Plus");click("购买 小祥 Plus");waitFor("quote"){node("购买账号：FixtureSelf")!=null&&node("本次付款")!=null};screenshot("checkout-light");click("取消");check(purchases.get()==0)
            click("购买 小祥 Plus");waitFor("quote"){node("购买账号：FixtureSelf")!=null&&node("本次付款")!=null};val checkoutWindow=test.uiAutomation.rootInActiveWindow.windowId;click("确认购买");waitFor("scan"){node("请看向屏幕")!=null};check(test.uiAutomation.rootInActiveWindow.windowId==checkoutWindow){"Checkout recreated the overlay"};screenshot("payment-scan");result.putString("checkout_payment_same_window","PASS");waitFor("paid"){node("套餐已开通")!=null};check(purchases.get()==1);screenshot("payment-success");click("完成")
            waitFor("stay on plans"){node("套餐与额度")!=null&&node("当前套餐 · 小祥 Plus")!=null};check(page=="plans")
            scroll(true);click("小祥 Ultra");click("升级至 小祥 Ultra");waitFor("upgrade quote"){node("升级套餐")!=null&&node("原价 42.50 信用点")!=null&&node("8.00")!=null};check(node("剩余天数向上取整")==null&&node("差价 30.00 × 8 天 ÷ 15")==null);screenshot("upgrade-light")
            test.runOnMainSync{mode=2};Thread.sleep(400);screenshot("upgrade-dark")
            test.runOnMainSync{font=1.3f};Thread.sleep(400);screenshot("upgrade-large-text");test.runOnMainSync{font=1f;animated=false};Thread.sleep(650)
            click("确认购买");waitFor("upgraded"){node("套餐已升级")!=null};check(purchases.get()==2);click("完成")
            waitFor("current ultra"){state.ai?.currentPlan?.optString("code")=="ultra"&&node("续购 小祥 Ultra")!=null};scroll(false);click("小祥 Plus");waitFor("downgrade blocked"){node("不支持降级")!=null};screenshot("plans-dark")
            test.runOnMainSync{page="chat";mode=1};waitFor("empty chat entry"){node("Saki AI · 套餐与额度")!=null};screenshot("empty-chat")
            test.runOnMainSync{state.conversation("AI 助手").add(ChatLine(9901,"小祥","你好呀，有什么可以帮到你？",false,"12:00"))};waitFor("chat entry hidden"){node("你好呀，有什么可以帮到你？")!=null&&node("Saki AI · 套餐与额度")==null};screenshot("active-chat")
            result.putString("upgrade_quote_dark_large_text_and_chat_entry","PASS")
            test.runOnMainSync{page="order"};waitFor("Saki order"){node("联系卖家 · 小祥")!=null};check(node("申请退款")==null&&node("领取说明")==null);click("联系卖家 · 小祥");check(contacted);screenshot("order")
            result.putString("confirm_cancel_payment_order_contact","PASS");test.finish(-1,result)
        }catch(failure:Throwable){runCatching{val bitmap=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("saki-failure.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()};result.putString("error",failure.stackTraceToString());test.finish(1,result)}
        finally{scope.cancel();test.runOnMainSync{host?.finish()};server.shutdown();context.getSharedPreferences("backend-v2",0).edit().clear().commit()}
    }
}
