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
import androidx.compose.ui.graphics.layer.drawLayer
import com.kyant.backdrop.backdrops.rememberLayerBackdrop
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.*
import okhttp3.mockwebserver.*
import org.json.*
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.TimeUnit

/** Production composables, isolated 3085-item HTTP fixture, no real purchases. */
object CatalogScaleCheck {
    fun run(test: Instrumentation, polish: Boolean = false) {
        val result=Bundle();val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val server=MockWebServer();var host:DeuteriumActivity?=null
        val context=object:ContextWrapper(test.targetContext){override fun getSharedPreferences(name:String,mode:Int):SharedPreferences=super.getSharedPreferences("qa-catalog-scale-$name",mode)}
        val requests=CopyOnWriteArrayList<String>();val failNext=AtomicBoolean(false)
        fun product(i:Int)=JSONObject().put("productId","product_$i").put("storeId","store_fixture").put("version",2).put("availableStock",JSONObject.NULL)
            .put("content",JSONObject().put("title","家居道具 ${i.toString().padStart(4,'0')}").put("subtitle","隔离目录测试").put("price","100.00")
                .put("inventoryPolicy","UNLIMITED").put("stock",0).put("limitPerOrder",64).put("categoryId",if(i%2==0)"furniture" else "garden")
                .put("brandId","brand").put("description","分页、全量搜索与缓存验收").put("deliverySummary","游戏邮箱").put("estimatedDelivery","付款后投递")
                .put("galleryAssetIds",JSONArray()).put("includedItems",JSONArray()))
        try {
            server.dispatcher=object:Dispatcher(){override fun dispatch(request:RecordedRequest):MockResponse {
                val url=request.requestUrl!!;val path=url.encodedPath.removePrefix("/api/v1");requests.add(url.encodedPath+"?"+url.encodedQuery)
                var next:String?=null
                val data:Any=when(path){
                    "/store/products"->{
                        val offset=url.queryParameter("cursor")?.toInt() ?: 0
                        if(offset>0&&failNext.compareAndSet(true,false))return MockResponse().setResponseCode(503).setBody("{\"error\":{\"code\":\"SERVER_BUSY\",\"message\":\"隔离测试：续页失败\"}}")
                        val query=url.queryParameter("q").orEmpty();val category=url.queryParameter("categoryId").orEmpty()
                        val all=(0 until 3085).filter{(query.isBlank()||product(it).getJSONObject("content").getString("title").contains(query))&&(category.isBlank()||product(it).getJSONObject("content").getString("categoryId")==category)}
                        val ids=all.drop(offset).take(url.queryParameter("limit")!!.toInt())
                        if(offset+ids.size<all.size)next=(offset+ids.size).toString()
                        JSONArray(ids.map(::product))
                    }
                    "/store/categories"->JSONArray().put(JSONObject().put("categoryId","furniture").put("name","家具")).put(JSONObject().put("categoryId","garden").put("name","庭院"))
                    "/store/brands"->JSONArray().put(JSONObject().put("brandId","brand").put("name","测试品牌"))
                    "/store/cart"->JSONObject().put("version",1).put("items",JSONArray().put(JSONObject().put("productId","product_3070").put("quantity",2)))
                    else->if(path.startsWith("/store/products/product_"))product(path.substringAfterLast('_').toInt())else return MockResponse().setResponseCode(404).setBody("{}")
                }
                return MockResponse().setBodyDelay(150,TimeUnit.MILLISECONDS).setHeader("Content-Type","application/json").setBody(JSONObject().put("data",data).put("page",JSONObject().put("nextCursor",next ?: JSONObject.NULL).put("hasMore",next!=null)).toString())
            }}
            server.start();val api=BackendApi(context,server.url("/").toString())
            val save=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            test.runOnMainSync{save.invoke(api,JSONObject().put("token","isolated-catalog-fixture").put("user",JSONObject().put("gameId","FixtureSelf").put("playerRef","fixture-self")));ShopCatalog.clear()}
            val state=LabState(scope,userName="FixtureSelf",api=api)
            var page by mutableStateOf("shop");var theme by mutableIntStateOf(1)
            host=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            val activity=host
            test.runOnMainSync{activity.setContent{LabTheme(theme,true,false){val backdrop=rememberGraphicsLayer()
                CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalOverlayBackdrop provides backdrop){
                    if (polish) {
                        val atmosphere = rememberGraphicsLayer()
                        val foreground = rememberGraphicsLayer()
                        val layer = rememberLayerBackdrop(graphicsLayer = backdrop) {
                            drawLayer(atmosphere)
                            drawLayer(foreground)
                        }
                        Box(Modifier.fillMaxSize()) {
                            GlassScene(atmosphere, backdrop, Modifier.fillMaxSize(), foreground, layer) {
                                CompositionLocalProvider(LocalGlassBackdrop provides atmosphere) {
                                    ShopPage(state, {}, WindowInsets.statusBars.asPaddingValues().calculateTopPadding(), "")
                                }
                            }
                            Box(Modifier.align(Alignment.BottomCenter).navigationBarsPadding()
                                .padding(horizontal = 18.dp, vertical = 8.dp).fillMaxWidth().height(64.dp)) {
                                MovingGlassNav(listOf("商城", "市场", "信息", "我的"),
                                    listOf(DeuteriumIcons.Shop, DeuteriumIcons.Market, DeuteriumIcons.Messages, DeuteriumIcons.Person),
                                    selected = 0, enabled = true, backdrop = layer) {}
                            }
                        }
                    } else Box(Modifier.fillMaxSize().recordGlassBackdrop(backdrop).background(MaterialTheme.colorScheme.background)){
                        if(page=="shop")ShopPage(state,{},90.dp,"")else SearchPage("Shop",state,{page="shop"},{})
                        if(page=="shop")Surface(Modifier.align(Alignment.TopCenter).fillMaxWidth().height(85.dp)){Row(Modifier.statusBarsPadding(),verticalAlignment=Alignment.CenterVertically){Text("商城隔离验收",Modifier.weight(1f));TextButton({page="search"}){Text("搜索")}}}
                    }
                }
            }}}
            fun nodes(root:AccessibilityNodeInfo?):List<AccessibilityNodeInfo> = if(root==null)emptyList() else listOf(root)+(0 until root.childCount).flatMap{nodes(root.getChild(it))}
            fun find(text:String)=nodes(test.uiAutomation.rootInActiveWindow).firstOrNull{it.text?.toString()==text||it.contentDescription?.toString()==text}
            fun waitFor(label:String,condition:()->Boolean){repeat(150){if(condition())return;Thread.sleep(80)};error("Timeout: $label")}
            fun click(text:String){waitFor(text){find(text)!=null};var n=find(text);while(n!=null&&!n.isClickable)n=n.parent;check(n?.performAction(AccessibilityNodeInfo.ACTION_CLICK)==true)}
            fun screenshot(name:String){test.waitForIdleSync();val bitmap=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("catalog-$name.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}
            waitFor("first products"){find("家居道具 0000")!=null}
            Thread.sleep(400)
            check(requests.count{it.startsWith("/api/v1/store/products?")&&!it.contains("badge=") }==1){"First screen fetched the whole catalog"}
            result.putString("first_page_only","PASS");screenshot("first-page-light")
            test.runOnMainSync{theme=2};Thread.sleep(250);screenshot("first-page-dark")
            if (polish) {
                val bounds = { label: String -> Rect().also { checkNotNull(find(label)).getBoundsInScreen(it) } }
                val search = bounds("搜索商品、品牌或分类")
                val bag = bounds("购物袋")
                val categories = bounds("更多分类")
                check(search.right <= bag.left && kotlin.math.abs(search.centerY() - bag.centerY()) <= 5)
                check(categories.top >= bag.bottom && categories.centerX() >= bag.centerX()) {
                    "The category button must remain in the second row: $bag / $categories"
                }
                result.putString("catalogPolish", "PASS baseline two-row header, bag glass and light/dark bottom bar")
                test.finish(-1, result)
                return
            }
            failNext.set(true)
            repeat(25){
                if(find("隔离测试：续页失败")==null){
                    val scroll=nodes(test.uiAutomation.rootInActiveWindow).filter{it.isScrollable}.maxByOrNull{val r=Rect();it.getBoundsInScreen(r);r.height()}
                    scroll?.performAction(AccessibilityNodeInfo.ACTION_SCROLL_FORWARD);Thread.sleep(160)
                }
            }
            waitFor("page retry"){find("重试")!=null}
            check(state.commerce.network!!.storePage.items.size==40){"Failure cleared loaded products"}
            click("重试");waitFor("next page"){state.commerce.network!!.storePage.items.size==80}
            // A slow automatic read must survive loading=true recomposition.
            repeat(25){
                if(state.commerce.network!!.storePage.items.size<120){
                    val scroll=nodes(test.uiAutomation.rootInActiveWindow).filter{it.isScrollable}.maxByOrNull{val r=Rect();it.getBoundsInScreen(r);r.height()}
                    scroll?.performAction(AccessibilityNodeInfo.ACTION_SCROLL_FORWARD);Thread.sleep(160)
                }
            }
            waitFor("delayed automatic next page"){state.commerce.network!!.storePage.items.size>=120}
            check(requests.count{it.contains("cursor=80") }==1){"Loading recomposition cancelled/restarted automatic pagination"}
            result.putString("slow_automatic_page_survives_recomposition","PASS")
            result.putString("scroll_prefetch_failure_retry","PASS");screenshot("continued-page")
            click("搜索");waitFor("search field"){nodes(test.uiAutomation.rootInActiveWindow).any{it.isEditable}}
            val field=nodes(test.uiAutomation.rootInActiveWindow).first{it.isEditable}
            check(field.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT,Bundle().apply{putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE,"3084")}))
            click("搜索")
            waitFor("full catalog search"){find("家居道具 3084")!=null}
            result.putString("search_beyond_loaded_pages","PASS");screenshot("search-last-product")
            runBlocking{withContext(Dispatchers.Main){state.commerce.network!!.refreshCart()}}
            check(state.cart["product_3070"]==2&&ShopCatalog.any{it.id=="product_3070"})
            check(state.commerce.network!!.unavailableCartProducts.isEmpty())
            check(requests.any{it.startsWith("/api/v1/store/products/product_3070?")})
            result.putString("cart_hydrates_unloaded_product","PASS")
            test.targetContext.filesDir.resolve("catalog-requests.json").writeText(JSONArray(requests).toString(2))
            test.finish(-1,result)
        }catch(failure:Throwable){
            test.targetContext.filesDir.resolve("catalog-requests.json").writeText(JSONArray(requests).toString(2))
            runCatching{val bitmap=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("catalog-failure.png").outputStream().use{bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()}
            result.putString("error",failure.stackTraceToString());test.finish(1,result)
        }
        finally{scope.cancel();host?.let{test.runOnMainSync{it.finish()}};runCatching{server.shutdown()}}
    }
}
