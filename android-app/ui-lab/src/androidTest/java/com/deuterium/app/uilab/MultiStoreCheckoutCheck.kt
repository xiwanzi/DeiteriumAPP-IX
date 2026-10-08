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
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.*
import okhttp3.mockwebserver.*
import org.json.*
import java.time.Instant
import java.util.concurrent.atomic.AtomicInteger

/** Actual coupon/bag/payment/order screens; all business data stays in MockWebServer. */
object MultiStoreCheckoutCheck {
 fun run(test:Instrumentation) {
  val result=Bundle();val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
  val server=MockWebServer();var host:DeuteriumActivity?=null
  val context=object:ContextWrapper(test.targetContext){override fun getSharedPreferences(name:String,mode:Int):SharedPreferences=super.getSharedPreferences("qa-checkout-v221-$name",mode)}
  try {
   val purchases=AtomicInteger();val now=Instant.now()
   val coupon=JSONObject("""{"couponId":"coupon_a","name":"开业礼遇","type":"ORDER","benefit":"FIXED","amountOff":"3.00","discountRate":9000,"minimumSpend":"10.00","maxDiscount":"0.00","stackWithProductDiscount":true,"active":true,"scopeDescription":"EOS Lab旗舰店 · 全部商品"}""").put("startsAt",now.minusSeconds(60).toString()).put("endsAt",now.plusSeconds(7200).toString())
   fun line(id:String,title:String,price:String)=JSONObject().put("productId",id).put("title",title).put("unitPrice",price).put("quantity",1).put("subtotal",price).put("productVersion",2)
   val a=line("product_a","EOS 补给包","10.00");val b=line("product_b","宜家台灯","20.00")
   fun group(store:String,name:String,item:JSONObject,total:String,discount:String)=JSONObject().put("storeId",store).put("storeName",name).put("channel","OFFICIAL_STORE").put("items",JSONArray().put(item)).put("totalAmount",total).put("originalTotal",item.getString("subtotal")).put("productDiscount","0.00").put("couponDiscount",discount).put("discountTotal",discount).put("delivery",JSONObject().put("method","MAILBOX"))
   val ga=group("store_a","EOS Lab旗舰店",a,"7.00","3.00").put("coupon",JSONObject().put("couponId","coupon_a").put("name","开业礼遇"))
   val gb=group("store_b","宜家家居",b,"20.00","0.00")
   val quote=JSONObject().put("quoteId","quote_group").put("version",1).put("channel","OFFICIAL_STORE").put("storeName","多店合并付款").put("items",JSONArray().put(a).put(b)).put("storeOrders",JSONArray().put(ga).put(gb)).put("totalAmount","27.00").put("originalTotal","30.00").put("productDiscount","0.00").put("couponDiscount","3.00").put("discountTotal","3.00").put("expiresAt",now.plusSeconds(300).toString())
   fun product(item:JSONObject,store:String)=JSONObject().put("productId",item.getString("productId")).put("storeId",store).put("version",2).put("availableStock",8).put("content",JSONObject().put("title",item.getString("title")).put("subtitle","隔离结算验收").put("description","隔离结算验收").put("price",item.getString("unitPrice")).put("brandId","brand").put("categoryId","category").put("galleryAssetIds",JSONArray()).put("includedItems",JSONArray()).put("limitPerOrder",5).put("deliverySummary","游戏邮箱").put("estimatedDelivery","付款后投递"))
   val products=JSONArray().put(product(a,"store_a")).put(product(b,"store_b"))
   fun order(id:String,group:JSONObject)=JSONObject(group.toString()).put("orderId",id).put("orderNo","D-$id").put("amount",group.getString("totalAmount")).put("buyer",JSONObject().put("displayName","FixtureSelf").put("playerRef","fixture-self")).put("seller",JSONObject().put("displayName",group.getString("storeName"))).put("status","AWAITING_CLAIM").put("fundsStatus","HELD").put("version",3).put("availableActions",JSONArray()).put("createdAt",now.toString())
   val oa=order("order_a",ga);val ob=order("order_b",gb)
   val op=JSONObject().put("operationId","operation_group").put("resourceId","order_a").put("resourceIds",JSONArray().put("order_a").put("order_b")).put("status","COMPLETED").put("amount","27.00")
   server.dispatcher=object:Dispatcher(){override fun dispatch(request:RecordedRequest):MockResponse {
    val path=request.requestUrl!!.encodedPath.removePrefix("/api/v1")
    val data:Any=when(path) {
     "/store/products"->products
     "/store/products/product_a"->products.getJSONObject(0)
     "/store/products/product_b"->products.getJSONObject(1)
     "/store/categories"->JSONArray().put(JSONObject().put("categoryId","category").put("name","测试分类"))
     "/store/brands"->JSONArray().put(JSONObject().put("brandId","brand").put("name","测试品牌"))
     "/store/cart"->JSONObject().put("version",1+purchases.get()).put("items",JSONArray().apply{if(purchases.get()==0){put(JSONObject().put("productId","product_a").put("quantity",1));put(JSONObject().put("productId","product_b").put("quantity",1))}})
     "/store/coupons"->JSONArray().put(coupon)
     "/checkout/quotes"->quote
     "/store/orders"->{check(request.method=="POST"&&purchases.incrementAndGet()==1){"Repeated payment"};JSONObject().put("operation",op).put("order",oa).put("orders",JSONArray().put(oa).put(ob))}
     "/operations/operation_group"->op
     "/orders/order_a"->oa
     "/orders/order_b"->ob
     "/orders"->JSONArray().apply{if(purchases.get()>0){put(oa);put(ob)}}
     "/wallet/balance","/wallet/balance/refresh"->JSONObject().put("balance",JSONObject().put("amount",if(purchases.get()==0)"100.00"else"73.00").put("currency","CREDIT").put("todayIncome","0.00").put("todayExpense",if(purchases.get()==0)"0.00"else"27.00").put("refreshedAt",Instant.now().toString()))
     "/wallet/records"->JSONObject().put("records",JSONArray())
     else->return MockResponse().setResponseCode(404).setBody("{}")
    }
    return MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",data).put("serverTime",Instant.now().toString()).toString())
   }}
   server.start();val api=BackendApi(context,server.url("/").toString())
   val save=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
   test.runOnMainSync{save.invoke(api,JSONObject().put("token","isolated-checkout").put("user",JSONObject().put("gameId","FixtureSelf").put("userId","fixture-user").put("playerRef","fixture-self")));ShopCatalog.clear()}
   val state=LabState(scope,userName="FixtureSelf",api=api)
   runBlocking{withContext(Dispatchers.Main){state.commerce.network!!.refreshStore();state.refresh()}}
   var page by mutableStateOf("coupons");var theme by mutableIntStateOf(1)
   host=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
   test.runOnMainSync{host!!.setContent{LabTheme(theme,true,false){val backdrop=rememberGraphicsLayer();CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalOverlayBackdrop provides backdrop){
    Box(Modifier.fillMaxSize().recordGlassBackdrop(backdrop).background(MaterialTheme.colorScheme.background)) {
     when(page){"coupons"->StoreCouponsPage(state,96.dp);"bag"->ShoppingBag(state,96.dp,{},{error("Group payment must open all orders")},{page="orders"});"orders"->OrdersPage(state.commerce,96.dp,{})}
     Surface(Modifier.align(Alignment.TopCenter).fillMaxWidth().height(90.dp),color=MaterialTheme.colorScheme.background){Text(if(page=="coupons")"优惠券"else if(page=="bag")"购物袋"else"我的订单",Modifier.statusBarsPadding().padding(20.dp),style=MaterialTheme.typography.titleMedium)}
    }
   }}}}
   fun find(node:AccessibilityNodeInfo?,text:String):AccessibilityNodeInfo?{if(node==null)return null;if(node.text?.toString()==text||node.contentDescription?.toString()==text)return node;for(i in 0 until node.childCount)find(node.getChild(i),text)?.let{return it};return null}
   fun node(text:String)=find(test.uiAutomation.rootInActiveWindow,text)
   fun waitFor(label:String,condition:()->Boolean){repeat(300){if(condition())return;Thread.sleep(60)};error("Timeout: $label")}
   fun click(text:String){
    fun target(n:AccessibilityNodeInfo?):AccessibilityNodeInfo?{if(n==null)return null;if(n.text?.toString()==text||n.contentDescription?.toString()==text){var candidate:AccessibilityNodeInfo?=n;while(candidate!=null){if(candidate.isClickable)return candidate;candidate=candidate.parent}};for(i in 0 until n.childCount)target(n.getChild(i))?.let{return it};return null}
    waitFor(text){target(test.uiAutomation.rootInActiveWindow)!=null};check(target(test.uiAutomation.rootInActiveWindow)?.performAction(AccessibilityNodeInfo.ACTION_CLICK)==true)
   }
   fun screenshot(name:String){test.waitForIdleSync();Thread.sleep(350);val bitmap=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("checkout-$name.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}
   fun checkCouponInk(expected:Int){val bounds=Rect();node(storeCoupon(coupon).benefitText)!!.getBoundsInScreen(bounds);val bitmap=test.uiAutomation.takeScreenshot();var matches=0;for(y in bounds.top.coerceAtLeast(0) until bounds.bottom.coerceAtMost(bitmap.height))for(x in bounds.left.coerceAtLeast(0) until bounds.right.coerceAtMost(bitmap.width)){val pixel=bitmap.getPixel(x,y);if(kotlin.math.abs(android.graphics.Color.red(pixel)-expected)<4&&kotlin.math.abs(android.graphics.Color.green(pixel)-expected)<4)matches++};bitmap.recycle();check(matches>40){"Coupon benefit not in original theme ink: $matches"}}
   waitFor("coupon"){node("开业礼遇")!=null};screenshot("coupons-light");checkCouponInk(21)
   test.runOnMainSync{theme=2};screenshot("coupons-dark");checkCouponInk(245)
   result.putString("coupon_original_black_white_blue","PASS")
   test.runOnMainSync{theme=1;page="bag"};waitFor("combined amount"){node("去结算 ¥27.00")!=null};screenshot("bag")
   click("去结算 ¥27.00");waitFor("one confirmation"){node("是否支付 ¥27.00？")!=null};check(node("将生成 2 笔店铺订单，分别发送游戏邮件。")!=null);screenshot("confirmation")
   click("确认付款");waitFor("split history"){page=="orders"&&state.commerce.orders.size==2};screenshot("orders")
   check(purchases.get()==1&&state.cart.isEmpty());check(state.commerce.orders.map{it.amount}.sorted()==listOf(700L,2000L));check(node("EOS Lab旗舰店")!=null&&node("宜家家居")!=null)
   result.putString("one_payment_two_order_history","PASS")
   runBlocking{withContext(Dispatchers.Main){state.commerce.orders.clear();api.saveOperation("STORE_PURCHASE",JSONObject().put("request",JSONObject().put("clientRequestId","original-group").put("quoteId","quote_group").put("expectedQuoteVersion",1)).put("operationId","operation_group"),api.financialScope());state.commerce.network!!.recoverOrder("STORE_PURCHASE")}}
   check(state.commerce.orders.size==2&&purchases.get()==1&&api.pendingOperation("STORE_PURCHASE")==null)
   result.putString("all_orders_recovered_without_repayment","PASS");test.finish(-1,result)
  }catch(error:Throwable){runCatching{val bitmap=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("checkout-failure.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()};result.putString("failure",error.stackTraceToString());test.finish(0,result)}
  finally{scope.cancel();runCatching{server.shutdown()};host?.let{test.runOnMainSync{it.finish()}}}
 }
}
