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
import androidx.compose.material.icons.outlined.ConfirmationNumber
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
import java.util.concurrent.atomic.AtomicReference

/** Real shopping-bag, coupon and payment composables against an isolated HTTP fixture. */
object CommercePromotionsCheck {
    fun run(test:Instrumentation) {
        val result=Bundle();val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val server=MockWebServer();var host:DeuteriumActivity?=null
        val context=object:ContextWrapper(test.targetContext){override fun getSharedPreferences(name:String,mode:Int):SharedPreferences=super.getSharedPreferences("qa-promotions-$name",mode)}
        try {
            val purchases=AtomicInteger();val cart=AtomicInteger(2);val clock=AtomicReference(Instant.now());val expires=clock.get().plusSeconds(7200)
            val item=JSONObject().put("productId","product_gift").put("title","EOS 探索补给包").put("subtitle","为下一次出发，备好所需。").put("unitPrice","8.88").put("quantity",2).put("subtotal","17.76")
            val coupon=JSONObject().put("couponId","coupon_gift").put("name","开业礼遇").put("type","ORDER").put("benefit","FIXED").put("amountOff","3.00").put("discountRate",9000).put("minimumSpend","10.00").put("maxDiscount","0.00").put("stackWithProductDiscount",true).put("active",true).put("scopeDescription","EOS Lab旗舰店 · 全部商品").put("startsAt",clock.get().minusSeconds(60).toString()).put("endsAt",expires.toString())
            val applied=JSONObject().put("couponId","coupon_gift").put("name","开业礼遇").put("type","ORDER").put("discountAmount","3.00")
            fun prices(value:JSONObject)=value.put("originalTotal","22.20").put("productDiscount","4.44").put("couponDiscount","3.00").put("discountTotal","7.44").put("coupon",applied)
            val product=JSONObject().put("productId","product_gift").put("storeId","store_eos").put("version",2).put("effectivePrice","8.88").put("availableStock",8).put("content",JSONObject().put("title","EOS 探索补给包").put("subtitle","为下一次出发，备好所需。").put("price","11.10").put("discountRate",8000).put("brandId","brand").put("categoryId","category").put("description","从日常补给开始，继续探索你的世界。").put("includedItems",JSONArray().put("探索物资")).put("galleryAssetIds",JSONArray()).put("limitPerOrder",5).put("deliveryCredits",20).put("deliverySummary","通过游戏内邮箱领取").put("estimatedDelivery","付款后自动发放").put("purchaseLimits",JSONObject().put("daily",5)))
            val order=prices(JSONObject().put("orderId","order_gift").put("orderNo","D20260910TEST").put("channel","OFFICIAL_STORE").put("buyer",JSONObject().put("displayName","FixtureSelf")).put("seller",JSONObject().put("displayName","EOS Lab旗舰店").put("contactQq","123456789")).put("delivery",JSONObject().put("method","MAILBOX").put("location","FixtureSelf")).put("items",JSONArray().put(item)).put("amount","14.76").put("status","AWAITING_CLAIM").put("fundsStatus","HELD").put("version",3).put("availableActions",JSONArray()).put("createdAt",clock.get().toString()))
            server.dispatcher=object:Dispatcher(){override fun dispatch(request:RecordedRequest):MockResponse {
                val path=request.requestUrl!!.encodedPath.removePrefix("/api/v1")
                val data:Any=when(path) {
                    "/store/products"->JSONArray().put(product)
                    "/store/products/product_gift"->product
                    "/store/categories"->JSONArray().put(JSONObject().put("categoryId","category").put("name","探索补给"))
                    "/store/brands"->JSONArray().put(JSONObject().put("brandId","brand").put("name","EOS Lab"))
                    "/store/cart"->JSONObject().put("version",if(cart.get()>0)1 else 2).put("items",JSONArray().apply{if(cart.get()>0)put(JSONObject().put("productId","product_gift").put("quantity",cart.get()).put("productVersion",2))})
                    "/store/coupons"->JSONArray().put(coupon)
                    "/checkout/quotes"->prices(JSONObject().put("quoteId","quote_gift").put("version",1).put("channel","OFFICIAL_STORE").put("storeName","EOS Lab旗舰店").put("items",JSONArray().put(item)).put("totalAmount","14.76").put("expiresAt",Instant.now().plusSeconds(300).toString()).put("delivery",JSONObject().put("method","MAILBOX")))
                    "/store/orders"->{check(request.method=="POST");check(purchases.incrementAndGet()==1){"Duplicate payment"};cart.set(0);JSONObject().put("operation",JSONObject().put("operationId","operation_gift").put("status","COMPLETED").put("resourceId","order_gift")).put("order",order)}
                    "/orders"->JSONArray().apply{if(purchases.get()>0)put(order)}
                    "/wallet/balance","/wallet/balance/refresh"->JSONObject().put("balance",JSONObject().put("amount",if(purchases.get()==0)"20.00" else "5.24").put("currency","CREDIT").put("todayIncome","0.00").put("todayExpense",if(purchases.get()==0)"0.00" else "14.76").put("refreshedAt",Instant.now().toString()))
                    "/wallet/records"->JSONObject().put("records",JSONArray())
                    else->return MockResponse().setResponseCode(404).setBody("{}")
                }
                return MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",data).put("serverTime",clock.get().toString()).toString())
            }}
            server.start();val api=BackendApi(context,server.url("/").toString())
            val save=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            test.runOnMainSync{save.invoke(api,JSONObject().put("token","isolated-commerce-fixture").put("user",JSONObject().put("gameId","FixtureSelf").put("userId","fixture-user").put("playerRef","fixture-self")))}
            val state=LabState(scope,userName="FixtureSelf",api=api)
            runBlocking{withContext(Dispatchers.Main){state.commerce.network!!.refreshStore();state.refresh()}}
            var page by mutableStateOf("bag");var theme by mutableIntStateOf(1);var font by mutableFloatStateOf(1f)
            host=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            val activity=host
            test.runOnMainSync{activity.setContent{LabTheme(theme,true,false){val backdrop=rememberGraphicsLayer();val density=LocalDensity.current
                CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalOverlayBackdrop provides backdrop,LocalDensity provides Density(density.density,font)) {
                    Box(Modifier.fillMaxSize().recordGlassBackdrop(backdrop).background(MaterialTheme.colorScheme.background)) {
                        when(page){"bag"->ShoppingBag(state,96.dp,{},{page="done"});"coupons"->StoreCouponsPage(state,96.dp);"profile"->ProfilePage(state,null,state.userName,96.dp,"",{page=it},{});"done"->Column(Modifier.fillMaxWidth().padding(top=140.dp),horizontalAlignment=Alignment.CenterHorizontally){Text("订单已创建");Text(state.commerce.orders.single().seller);Text(credit(state.commerce.orders.single().amount))}}
                        Surface(Modifier.align(Alignment.TopCenter).fillMaxWidth().height(90.dp),color=MaterialTheme.colorScheme.background){Row(Modifier.statusBarsPadding().fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){IconButton({page="bag"}){Icon(Icons.Outlined.ChevronLeft,"返回")};Text(if(page=="coupons")"优惠券" else "购物袋",Modifier.weight(1f),style=MaterialTheme.typography.titleMedium);IconButton({page="coupons"}){Icon(Icons.Outlined.ConfirmationNumber,"优惠券")}}}
                    }
                }
            }}}
            fun find(node:AccessibilityNodeInfo?,text:String):AccessibilityNodeInfo?{if(node==null)return null;if(node.text?.toString()==text||node.contentDescription?.toString()==text)return node;for(i in 0 until node.childCount)find(node.getChild(i),text)?.let{return it};return null}
            fun node(text:String)=find(test.uiAutomation.rootInActiveWindow,text)
            fun waitFor(label:String,condition:()->Boolean){repeat(180){if(condition())return;Thread.sleep(60)};error("Timeout: $label")}
            fun click(text:String){
                fun target(n:AccessibilityNodeInfo?):AccessibilityNodeInfo?{if(n==null)return null;if(n.text?.toString()==text||n.contentDescription?.toString()==text){var candidate:AccessibilityNodeInfo?=n;while(candidate!=null){if(candidate.isClickable)return candidate;candidate=candidate.parent}};for(i in 0 until n.childCount)target(n.getChild(i))?.let{return it};return null}
                waitFor(text){target(test.uiAutomation.rootInActiveWindow)!=null};check(target(test.uiAutomation.rootInActiveWindow)?.performAction(AccessibilityNodeInfo.ACTION_CLICK)==true){"Cannot click $text"}
            }
            fun screenshot(name:String,settle:Boolean=true){test.waitForIdleSync();if(settle)Thread.sleep(300);val bitmap=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("commerce-$name.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}
            runBlocking{withContext(Dispatchers.Main){state.commerce.network!!.refreshStore()}}
            waitFor("discounted bag"){node("去结算 14.76")!=null};screenshot("bag-light")
            click("优惠券");waitFor("coupon card"){node("开业礼遇")!=null};screenshot("coupons-light")
            test.runOnMainSync{theme=2};screenshot("coupons-dark")
            test.runOnMainSync{font=1.4f};screenshot("coupons-large-text")
            result.putString("coupon_light_dark_large_text","PASS")
            clock.set(expires.plusSeconds(1));runBlocking{withContext(Dispatchers.Main){state.commerce.network!!.refreshCoupons()}}
            waitFor("expired coupon removal"){node("期待下一份优惠")!=null&&node("开业礼遇")==null};screenshot("expired-removed")
            result.putString("expired_coupon_disappears","PASS")
            clock.set(Instant.now());test.runOnMainSync{theme=1;font=1f;page="bag"}
            waitFor("payment amount"){node("去结算 14.76")!=null};click("去结算 14.76");waitFor("confirmation"){node("是否支付 14.76 信用点？")!=null};screenshot("confirmation")
            click("确认付款");waitFor("renamed store on payment"){node("EOS Lab旗舰店")!=null&&node("请看向屏幕")!=null};screenshot("payment-store-name",false)
            waitFor("completed order"){node("订单已创建")!=null}
            check(purchases.get()==1);check(state.cart.isEmpty());check(state.commerce.orders.single().amount==1476L)
            result.putString("renamed_store_discounted_payment_single_submit","PASS")
            test.runOnMainSync{page="bag"};waitFor("empty bag"){node("购物袋还是空的")!=null};screenshot("bag-cleared")
            result.putString("successful_payment_clears_bag","PASS")
            test.runOnMainSync{page="profile"};waitFor("my offers entry"){node("我的优惠")!=null};check(node("历史账单")==null);screenshot("profile-offers-entry")
            click("我的优惠");waitFor("profile coupon destination"){node("为你准备的优惠")!=null};check(page=="coupons")
            result.putString("profile_my_offers_entry","PASS")
            test.finish(-1,result)
        } catch(error:Throwable){
            runCatching{val bitmap=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("commerce-failure.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}
            fun text(n:AccessibilityNodeInfo?):String{if(n==null)return "";return listOfNotNull(n.text?.toString(),n.contentDescription?.toString()).joinToString(" | ")+"\n"+(0 until n.childCount).joinToString(""){text(n.getChild(it))}}
            result.putString("visible",text(test.uiAutomation.rootInActiveWindow));result.putString("failure",error.stackTraceToString());test.finish(0,result)
        }
        finally{scope.cancel();runCatching{server.shutdown()};host?.let{test.runOnMainSync{it.finish()}}}
    }
}
