package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.*
import android.os.Bundle
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ChevronLeft
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
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicReference

/** Actual arrival host and coupon page; local HTTP, isolated preferences, no live operations. */
object CouponArrivalCheck {
    fun run(test:Instrumentation) {
        val result=Bundle();val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val server=MockWebServer();var host:DeuteriumActivity?=null
        val context=object:ContextWrapper(test.targetContext){override fun getSharedPreferences(name:String,mode:Int):SharedPreferences=super.getSharedPreferences("qa-coupon-arrivals-$name",mode)}
        try {
            context.getSharedPreferences("backend-v2",0).edit().clear().commit()
            val clock=AtomicReference(Instant.now());val coupons=CopyOnWriteArrayList<JSONObject>()
            val receipts=ConcurrentHashMap<String,Int>();val failAck=AtomicBoolean(true)
            fun coupon(id:String,name:String,restricted:Boolean=false)=JSONObject().put("couponId",id).put("name",name).put("type",if(restricted)"ITEM" else "ORDER").put("benefit",if(restricted)"PERCENT" else "FIXED").put("amountOff","20.00").put("discountRate",8500).put("minimumSpend",if(restricted)"0.00" else "100.00").put("maxDiscount",if(restricted)"50.00" else "0.00").put("stackWithProductDiscount",true).put("storeIds",JSONArray().apply{if(restricted)put("store_eos")}).put("productIds",JSONArray()).put("scopeDescription",if(restricted)"EOS Lab旗舰店 · 全部商品" else "全部店铺 · 全部商品").put("startsAt",clock.get().minusSeconds(10).toString()).put("endsAt",clock.get().plusSeconds(7200).toString())
            coupons.add(coupon("coupon_one","初见礼遇"))
            server.dispatcher=object:Dispatcher(){override fun dispatch(request:RecordedRequest):MockResponse {
                val path=request.requestUrl!!.encodedPath.removePrefix("/api/v1")
                val owner=request.getHeader("Authorization").orEmpty()
                val data:Any=when {
                    path=="/store/coupons/attention"&&request.method=="POST" -> {
                        if(failAck.get())return MockResponse().setResponseCode(503).setBody("{}")
                        val value=JSONObject(request.body.readUtf8());val ids=value.getJSONArray("couponIds");val level=if(value.getBoolean("viewed"))2 else 1
                        for(i in 0 until ids.length())receipts.merge("$owner:${ids.getString(i)}",level,::maxOf)
                        JSONObject().put("acknowledged",true)
                    }
                    path=="/store/coupons/attention"||path=="/store/coupons" -> JSONArray().apply {
                        coupons.forEach{coupon->val level=receipts["$owner:${coupon.getString("couponId")}"] ?: 0
                            if(Instant.parse(coupon.getString("endsAt")).isAfter(clock.get())&&(path=="/store/coupons"||level<2))put(JSONObject(coupon.toString()).put("announced",level>0))}
                    }
                    else -> return MockResponse().setResponseCode(404).setBody("{}")
                }
                return MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",data).put("serverTime",clock.get().toString()).put("page",JSONObject().put("hasMore",false).put("nextCursor",JSONObject.NULL)).toString())
            }}
            server.start();val api=BackendApi(context,server.url("/").toString())
            val save=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            fun login(owner:String){test.runOnMainSync{save.invoke(api,JSONObject().put("token",owner).put("user",JSONObject().put("gameId",owner).put("playerRef",owner)))}}
            login("FixtureSelf")
            var state by mutableStateOf(LabState(scope,userName="FixtureSelf",api=api))
            var page by mutableStateOf("profile");var ready by mutableStateOf(false);var busy by mutableStateOf(false)
            var theme by mutableIntStateOf(1);var font by mutableFloatStateOf(1f);var glass by mutableStateOf(true);var motion by mutableStateOf(true)
            host=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            test.runOnMainSync{host!!.setContent{LabTheme(theme,motion,false){
                val density=LocalDensity.current;val backdrop=rememberGraphicsLayer();val overlays=remember{IosOverlayRegistry()}
                CompositionLocalProvider(LocalDensity provides Density(density.density,font),LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalOverlayBackdrop provides backdrop,LocalIosOverlayRegistry provides overlays,LocalOverlayGlassEnabled provides glass) {
                    Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)) {
                      Box(Modifier.fillMaxSize().recordGlassBackdrop(backdrop)) {
                        if(page=="profile")ProfilePage(state,null,"FixtureSelf",100.dp,"",{page=it},{}) else StoreCouponsPage(state,100.dp)
                        Surface(Modifier.align(Alignment.TopCenter).fillMaxWidth().height(90.dp),color=MaterialTheme.colorScheme.background){Row(Modifier.statusBarsPadding().fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){IconButton({page="profile"}){Icon(Icons.Outlined.ChevronLeft,"返回")};Text(if(page=="profile")"我的" else "我的优惠",Modifier.weight(1f),style=MaterialTheme.typography.titleLarge)}}
                      }
                      CouponArrivalHost(state.commerce.network!!.couponAttention,ready&&page=="profile",98.dp,{page="coupons"},Modifier.fillMaxSize())
                    }
                    if(busy)IosDialog({},{Text("正在确认付款")},{Text("优惠提醒应等待此浮层关闭。")},{PlainButton({busy=false}){Text("完成")}})
                }
            }}}
            fun find(n:AccessibilityNodeInfo?,text:String):AccessibilityNodeInfo?{if(n==null)return null;if(n.text?.toString()==text||n.contentDescription?.toString()==text)return n;for(i in 0 until n.childCount)find(n.getChild(i),text)?.let{return it};return null}
            fun node(text:String)=find(test.uiAutomation.rootInActiveWindow,text)
            fun waitFor(label:String,condition:()->Boolean){repeat(200){if(condition())return;Thread.sleep(60)};error("Timeout: $label")}
            fun click(text:String){waitFor(text){node(text)!=null};var n=node(text);while(n!=null&&!n.isClickable)n=n.parent;check(n?.performAction(AccessibilityNodeInfo.ACTION_CLICK)==true){"Cannot click $text"}}
            fun shot(name:String){test.waitForIdleSync();Thread.sleep(900);val image=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("coupon-arrival-$name.png").outputStream().use{image.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};image.recycle()}
            fun refresh(){runBlocking{withContext(Dispatchers.Main){check(state.commerce.network!!.couponAttention.refresh())}}}
            waitFor("initial coupon fetch"){state.commerce.network!!.couponAttention.arrivals.size==1}
            Thread.sleep(650);check(node("收到新的优惠")==null)
            test.runOnMainSync{busy=true;ready=true};waitFor("payment overlay"){node("正在确认付款")!=null};Thread.sleep(800);check(node("收到新的优惠")==null)
            click("完成");waitFor("simple coupon popup"){node("收到新的优惠")!=null};check(node("去看看")==null);shot("simple-light")
            test.runOnMainSync{theme=2};shot("simple-dark")
            val regularTitle=android.graphics.Rect().also{node("收到新的优惠")!!.getBoundsInScreen(it)}
            test.runOnMainSync{font=1.4f}
            waitFor("large text applied inside dialog"){val bounds=android.graphics.Rect();node("收到新的优惠")?.getBoundsInScreen(bounds);bounds.height()>regularTitle.height()*1.2f}
            shot("simple-large-text")
            test.runOnMainSync{font=1f;glass=false;motion=false};shot("simple-no-glass")
            click("好的");waitFor("simple viewed"){!state.commerce.network!!.couponAttention.hasUnread}
            result.putString("single_action_light_dark_large_text_and_overlay_deferral","PASS")
            // Ack remains unavailable; reconstruct all app state against the old server response.
            test.runOnMainSync{state=LabState(scope,userName="FixtureSelf",api=api);theme=1;glass=true;motion=true}
            refresh();Thread.sleep(850);check(node("收到新的优惠")==null);check(!state.commerce.network!!.couponAttention.hasUnread)
            failAck.set(false);runBlocking{withContext(Dispatchers.Main){state.commerce.network!!.couponAttention.flush()}}
            check(receipts["Bearer FixtureSelf:coupon_one"]==2)
            result.putString("offline_ack_survives_recreation_and_syncs","PASS")
            login("FixtureOther");val other=CouponAttention(api)
            runBlocking{withContext(Dispatchers.Main){check(other.refresh());check(other.arrivals.size==1)}}
            login("FixtureSelf")
            result.putString("account_receipts_are_isolated","PASS")
            coupons.add(coupon("coupon_two","探索者单品礼遇",true));coupons.add(coupon("coupon_three","周末满减礼遇"))
            test.runOnMainSync{state=LabState(scope,userName="FixtureSelf",api=api)}
            waitFor("grouped arrival"){node("收到 2 份新优惠")!=null};check(node("好的")!=null&&node("去看看")!=null);shot("multiple-light")
            test.runOnMainSync{theme=2;font=1.4f};shot("multiple-dark-large-text")
            test.runOnMainSync{theme=1;font=1f}
            click("去看看");waitFor("coupon wallet"){node("为你准备的优惠")!=null&&node("探索者单品礼遇")!=null};waitFor("wallet marked read"){!state.commerce.network!!.couponAttention.hasUnread}
            check(page=="coupons");result.putString("grouped_popup_navigation_and_badge_read","PASS")
            click("返回")
            coupons.add(coupon("coupon_four","午后惊喜"));refresh()
            waitFor("foreground banner"){node("收到一份新优惠")!=null};check(node("收到新的优惠")==null);check(state.commerce.network!!.couponAttention.hasUnread);shot("foreground-banner")
            click("收到一份新优惠");waitFor("banner navigation"){page=="coupons"};waitFor("banner coupon read"){!state.commerce.network!!.couponAttention.hasUnread}
            result.putString("foreground_banner_and_badge","PASS")
            click("返回")
            val expiring=coupon("coupon_five","即将结束的礼遇",true).put("endsAt",clock.get().plusSeconds(4).toString());coupons.add(expiring)
            test.runOnMainSync{state=LabState(scope,userName="FixtureSelf",api=api)}
            waitFor("expiry popup"){node("收到新的优惠")!=null};check(node("去看看")!=null)
            waitFor("expired popup removed"){node("收到新的优惠")==null};check(!state.commerce.network!!.couponAttention.hasUnread)
            result.putString("cached_popup_expiry_removes_badge_and_dialog","PASS")
            test.finish(-1,result)
        } catch(error:Throwable) {
            runCatching{val image=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("coupon-arrival-failure.png").outputStream().use{image.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};image.recycle()}
            fun dump(n:AccessibilityNodeInfo?):String=if(n==null)"" else listOfNotNull(n.text,n.contentDescription).joinToString(" | ")+"\n"+(0 until n.childCount).joinToString(""){dump(n.getChild(it))}
            result.putString("visible",dump(test.uiAutomation.rootInActiveWindow));result.putString("failure",error.stackTraceToString());test.finish(0,result)
        } finally {scope.cancel();runCatching{server.shutdown()};host?.let{test.runOnMainSync{it.finish()}}}
    }
}
