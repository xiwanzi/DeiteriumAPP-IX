package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Context
import android.content.ContextWrapper
import android.content.Intent
import android.content.SharedPreferences
import android.graphics.Bitmap
import android.os.Bundle
import android.text.Spanned
import android.view.View
import android.view.ViewGroup
import android.view.accessibility.AccessibilityNodeInfo
import android.widget.TextView
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.*
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import okhttp3.mockwebserver.SocketPolicy
import org.json.JSONArray
import org.json.JSONObject
import java.time.Instant
import java.time.LocalDateTime
import java.util.Collections
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger

/** Only the separate QA APK can start this loopback fixture. No production business writes. */
object ReleaseV203Check {
    fun run(test: Instrumentation) {
        val result=Bundle()
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val server=MockWebServer()
        val previousPlayers=Players.toList();val previousCatalog=ShopCatalog.toList()
        val image=test.targetContext.cacheDir.resolve("v203-fixture.png")
        val historyGate=CountDownLatch(1)
        var activity:MainActivity?=null
        val fixtureContext=object:ContextWrapper(test.targetContext){
            override fun getSharedPreferences(name:String,mode:Int):SharedPreferences=super.getSharedPreferences("qa-v203-$name",mode)
        }
        try {
            Bitmap.createBitmap(240,240,Bitmap.Config.ARGB_8888).apply{eraseColor(0xff81b59c.toInt())}.let{bitmap->image.outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}
            val product=ShopProduct("fixture-product","石材建造礼包","建材","新上架的商品",1230,Color.White,photos=listOf(image.toURI().toString()),description="用于隔离环境验证新商品图片和加购流程。",stock=9,limit=9)
            val cartQuantity=AtomicInteger();val cartWrites=AtomicInteger();val hideCalls=AtomicInteger();val readCalls=AtomicInteger();val unread=AtomicInteger(2)
            val disconnectOnce=AtomicBoolean(true)
            val publicRequests=Collections.synchronizedList(mutableListOf<String>())
            val routes=Collections.synchronizedList(mutableListOf<String>())
            fun message(id:String,name:String,ref:String,text:String)=JSONObject().put("messageId",id).put("kind","private_chat").put("content",text).put("sentAt",Instant.now().toString()).put("sender",JSONObject().put("gameId",name).put("playerRef",ref))
            val directMessage=message("dm-fixture","FixtureFriend","fixture-friend","收到一条新的私聊消息")
            fun cart()=JSONObject().put("version",cartWrites.get()+1).put("items",JSONArray().apply{if(cartQuantity.get()>0)put(JSONObject().put("productId",product.id).put("quantity",cartQuantity.get()))})
            val quote=JSONObject().put("quoteId","fixture-quote").put("version",1).put("channel","OFFICIAL_STORE").put("totalAmount","12.30")
                .put("items",JSONArray().put(JSONObject().put("productId",product.id).put("title",product.name).put("unitPrice","12.30").put("quantity",1).put("subtotal","12.30")))
                .put("warnings",JSONArray().put("报价不预留库存，付款时重新校验。"))
            server.dispatcher=object:Dispatcher(){
                override fun dispatch(request:RecordedRequest):MockResponse {
                    val path=request.requestUrl!!.encodedPath;routes.add("${request.method} $path")
                    check(request.getHeader("Authorization")?.startsWith("Bearer v203-fixture-")==true){"Fixture request lost authentication"}
                    val data=when {
                        path=="/api/v1/store/products/${product.id}"->JSONObject().put("productId",product.id).put("version",1).put("availableStock",9).put("content",JSONObject().put("title",product.name).put("subtitle",product.subtitle).put("price","12.30").put("description",product.description).put("deliverySummary","游戏内邮箱").put("estimatedDelivery","完成付款后交付").put("galleryAssetIds",JSONArray()).put("includedItems",JSONArray().put("建筑石材")))
                        path=="/api/v1/store/cart"->cart()
                        path=="/api/v1/store/cart/items/${product.id}"->{cartQuantity.set(JSONObject(request.body.readUtf8()).getInt("quantity"));cartWrites.incrementAndGet();cart()}
                        path=="/api/v1/checkout/quotes"->quote
                        path=="/api/v1/commissions/ended/hide"->{hideCalls.incrementAndGet();JSONObject().put("hidden",true)}
                        path=="/api/v1/commissions"||path=="/api/v1/commissions/me"->JSONArray()
                        path=="/api/v1/chat/conversations"->JSONArray().put(JSONObject().put("conversationId","fixture-conversation").put("unreadCount",unread.get()).put("lastMessage",directMessage).put("otherPlayer",JSONObject().put("gameId","FixtureFriend").put("playerRef","fixture-friend").put("qq","12345678")))
                        path=="/api/v1/chat/conversations/fixture-conversation/messages"->JSONArray().put(directMessage)
                        path=="/api/v1/chat/conversations/fixture-conversation/read"->{readCalls.incrementAndGet();unread.set(0);JSONObject().put("unreadCount",0)}
                        path=="/api/v1/chat/messages"&&request.method=="POST"->{
                            val input=JSONObject(request.body.readUtf8());publicRequests.add(input.toString())
                            if(disconnectOnce.getAndSet(false))return MockResponse().setSocketPolicy(SocketPolicy.DISCONNECT_AFTER_REQUEST)
                            JSONObject().put("status","accepted").put("messageId","public-fixture").put("message",message("public-fixture","FixtureSelf","fixture-self",input.getString("content")).put("kind","public_chat"))
                        }
                        path=="/api/v1/chat/messages"->{historyGate.await(15,TimeUnit.SECONDS);JSONObject().put("messages",JSONArray())}
                        else->return MockResponse().setResponseCode(404).setBody("""{"error":{"code":"NOT_FOUND","message":"Unknown fixture route"}}""")
                    }
                    return MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",data).toString())
                }
            }
            server.start()
            val api=BackendApi(fixtureContext,server.url("/").toString())
            val session=JSONObject().put("token","v203-fixture-"+"x".repeat(40)).put("user",JSONObject().put("gameId","FixtureSelf").put("playerRef","fixture-self"))
            val remember=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            test.runOnMainSync{remember.invoke(api,session)}
            val state=LabState(scope,userName="FixtureSelf",api=api)
            val display=LabState(scope,userName="FixtureSelf")
            var screen by mutableStateOf("product");var theme by mutableIntStateOf(1);var flying by mutableStateOf<ShopProduct?>(null)
            val added=AtomicInteger()
            val host=test.startActivitySync(Intent(test.targetContext,MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as MainActivity
            activity=host
            val markdown="# 公告与 AI 回复\n\n**重点内容**，普通正文与 ~~已取消~~。\n\n- 第一项\n- 第二项\n\n> 引用说明\n\n```kotlin\nval count = 2\n```\n\n| 商品 | 价格 |\n| --- | --- |\n| 石材 | 12.30 |\n\n[查看详情](https://example.invalid)"
            test.runOnMainSync {
                state.balanceKnown=true;state.balance=10000
                ShopCatalog.clear();ShopCatalog.add(product)
                host.setContent{LabTheme(theme,false,false){CompositionLocalProvider(androidx.compose.material3.LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalOverlayGlassEnabled provides false){Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)){
                    when(screen){
                        "product"->ProductPage(product,state,28.dp,{screen="bag"}){p,_->added.incrementAndGet();flying=p}
                        "bag"->ShoppingBag(state,28.dp,{},{error("Fixture must not create an order")})
                        "search"->SearchPage("Shop",display,{},{},active=false)
                        "hall"->CommissionHallPage(display,28.dp,{})
                        "mine"->CommissionHallPage(display,28.dp,{},mine=true)
                        "contacts"->InfoPage(state,28.dp,""){}
                        "direct"->DirectChatPage(state,"FixtureFriend",28.dp){}
                        "announcement-markdown"->CommunityDetail(false,display,28.dp)
                        "ai-markdown"->DirectChatPage(display,"AI 助手",28.dp){}
                    }
                    if(screen=="product")flying?.let{ShoppingBagFlight(it,Offset(220f,700f),Offset(900f,80f),.45f)}
                }}}}
            }
            fun find(node:AccessibilityNodeInfo?,predicate:(AccessibilityNodeInfo)->Boolean):AccessibilityNodeInfo? {
                if(node==null)return null
                if(predicate(node))return node
                for(i in 0 until node.childCount)find(node.getChild(i),predicate)?.let{return it}
                return null
            }
            fun named(text:String)=find(test.uiAutomation.rootInActiveWindow){it.text?.toString()==text}
            fun waitFor(label:String,check:()->Boolean){repeat(80){if(check())return;Thread.sleep(100)};error("Timed out: $label")}
            fun click(text:String){waitFor(text){named(text)!=null};var node=checkNotNull(named(text));while(!node.isClickable)node=checkNotNull(node.parent);check(node.performAction(AccessibilityNodeInfo.ACTION_CLICK))}
            fun screenshot(name:String){test.waitForIdleSync();Thread.sleep(400);test.uiAutomation.takeScreenshot()?.let{bitmap->test.targetContext.filesDir.resolve("qa-v203-$name.png").outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}}
            waitFor("product button"){named("加入购物袋")!=null}
            click("加入购物袋")
            waitFor("successful add and visible flight"){added.get()==1&&cartWrites.get()==1&&state.cart[product.id]==1}
            screenshot("cart-flight")
            check(product.image==0){"Fixture did not exercise network-only product"}
            test.runOnMainSync{ShopCatalog.clear();ShopCatalog.add(product)}
            click("查看购物袋")
            waitFor("shopping bag"){named("你的购物袋")!=null}
            click("去结算 12.30")
            waitFor("confirmation"){named("是否支付 12.30 信用点？")!=null&&named("确认付款")!=null}
            check(find(test.uiAutomation.rootInActiveWindow){it.text?.contains("报价")==true}==null){"Internal quote text exposed"}
            screenshot("checkout-light")
            click("取消")
            result.putString("real_cart_callback_zero_resource_and_order_confirmation","PASS")
            test.runOnMainSync{screen="search";ShopCatalog.clear();ShopCatalog.add(product)}
            waitFor("search"){find(test.uiAutomation.rootInActiveWindow){it.isEditable}!=null}
            check(find(test.uiAutomation.rootInActiveWindow){it.text?.toString() in setOf("Apple","NVIDIA","AMD","iPhone")}==null)
            find(test.uiAutomation.rootInActiveWindow){it.isEditable}!!.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT,Bundle().apply{putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE,"石材")})
            waitFor("search thumbnail"){find(test.uiAutomation.rootInActiveWindow){it.contentDescription?.toString()==product.name}!=null}
            screenshot("search-thumbnail")
            result.putString("real_catalog_search_and_thumbnail","PASS")
            val draft=CommissionDraft("可接取委托","仍然需要完成的工作","主城",200,24,Urgency.Normal,image.toURI().toString())
            val open=Commission("open","open","FixtureSelf",draft,LocalDateTime.now(),serverStatus="OPEN",fundsStatus="HELD")
            val ended=open.copy(id="ended",key="ended",draft=draft.copy(title="已取消的旧委托"),stage=CommissionStage.Cancelled,serverStatus="CANCELLED",fundsStatus="REFUNDED",canHideRecord=true)
            test.runOnMainSync{display.commissions.entries.addAll(listOf(open,ended));screen="hall"}
            waitFor("open commission"){named(draft.title)!=null};check(named(ended.draft.title)==null)
            test.runOnMainSync{screen="mine"}
            waitFor("own terminal commission"){named(ended.draft.title)!=null&&named("删除记录")!=null}
            // Use the real network adapter against the fixture for the deletion callback.
            test.runOnMainSync{display.commissions.network=BackendCommissions(api,display)}
            click("删除记录");waitFor("delete confirmation"){named("删除这条记录？")!=null}
            click("删除记录");waitFor("server-confirmed deletion"){hideCalls.get()==1&&display.commissions.entries.none{it.id=="ended"}&&named(ended.draft.title)==null}
            result.putString("hall_filters_terminal_and_delete_callback","PASS")
            runBlocking{withContext(Dispatchers.Main){state.refreshContacts()}}
            test.runOnMainSync{screen="contacts"}
            waitFor("unread contact"){find(test.uiAutomation.rootInActiveWindow){it.contentDescription?.toString()=="有未读消息"}!=null}
            screenshot("unread-light")
            test.runOnMainSync{state.activeConversation="FixtureFriend";state.foreground=true;screen="direct"}
            waitFor("read cursor acknowledged"){readCalls.get()>0&&state.conversationUnread["FixtureFriend"]==0}
            test.runOnMainSync{screen="contacts";state.activeConversation=null}
            waitFor("cleared unread dot"){named("联系人")!=null&&find(test.uiAutomation.rootInActiveWindow){it.contentDescription?.toString()=="有未读消息"}==null}
            result.putString("unread_summary_dot_and_read_ack","PASS")
            test.runOnMainSync{state.chatDraft=androidx.compose.ui.text.input.TextFieldValue("公共消息测试 @FixtureFriend");state.sendChat()}
            waitFor("lost response retained"){!state.chatReplyPending&&api.pendingChat()!=null}
            test.runOnMainSync{state.sendChat()}
            waitFor("accepted before history completes"){!state.chatReplyPending&&state.chat.any{it.remoteId=="public-fixture"}}
            check(historyGate.count==1L&&publicRequests.size==2)
            check(publicRequests[0]==publicRequests[1]){"Unknown-result retry changed the original request"}
            check(api.pendingChat()==null)
            historyGate.countDown()
            result.putString("public_send_without_socket_history_wait_and_same_key_retry","PASS")
            fun nativeTexts(view:View):List<TextView> = (if(view is TextView)listOf(view)else emptyList())+(if(view is ViewGroup)(0 until view.childCount).flatMap{nativeTexts(view.getChildAt(it))}else emptyList())
            test.runOnMainSync{
                display.announcements.add(Announcement("fixture-announcement","官方公告","## 更新说明\n\n"+markdown,"2026-09-09"))
                display.conversation("AI 助手").add(ChatLine(500,"AI 助手",markdown,false,remoteId="ai-markdown"))
            }
            for(page in listOf("announcement-markdown","ai-markdown"))for(mode in listOf(1,2)){
                test.runOnMainSync{screen=page;theme=mode}
                waitFor("rendered Markdown"){find(test.uiAutomation.rootInActiveWindow){it.text?.contains("重点内容")==true}!=null}
                waitFor("Markdown theme applied"){
                    val correct=AtomicBoolean()
                    test.runOnMainSync{correct.set(nativeTexts(host.window.decorView).any{it.text.contains("重点内容")&&it.currentTextColor==(if(mode==2)0xfff5f5f7.toInt()else 0xff151518.toInt())})}
                    correct.get()
                }
                val markdownFailure=java.util.concurrent.atomic.AtomicReference<Throwable>()
                test.runOnMainSync{runCatching{
                    val text=checkNotNull(nativeTexts(host.window.decorView).firstOrNull{it.text.contains("重点内容")})
                    check(!text.text.contains("**"));val spans=text.text as Spanned
                    val start=spans.toString().indexOf("重点内容");val paint=android.text.TextPaint()
                    spans.getSpans(start,start+4,android.text.style.MetricAffectingSpan::class.java).forEach{it.updateDrawState(paint)}
                    check(paint.isFakeBoldText||paint.typeface?.isBold==true){"Strong emphasis not rendered bold"}
                    check(spans.getSpans(0,spans.length,Any::class.java).any{it.javaClass.name.contains("Table")}){"Table not rendered"}
                }.onFailure{markdownFailure.set(it)}}
                markdownFailure.get()?.let{throw it}
                screenshot(page+if(mode==1)"-light"else "-dark")
            }
            check(routes.none{it.contains("/store/orders")||it.contains("/wallet/")||it.endsWith("/chat/ws")})
            result.putString("native_markdown_headings_bold_lists_code_tables_both_themes","PASS")
            test.finish(-1,result)
        } catch(error:Throwable){result.putString("error",error.stackTraceToString());test.finish(1,result)}
        finally {
            historyGate.countDown();scope.cancel()
            test.runOnMainSync{activity?.finish();Players.clear();Players.addAll(previousPlayers);ShopCatalog.clear();ShopCatalog.addAll(previousCatalog)}
            server.shutdown();fixtureContext.getSharedPreferences("backend-v2",0).edit().clear().commit();image.delete()
        }
    }
}
