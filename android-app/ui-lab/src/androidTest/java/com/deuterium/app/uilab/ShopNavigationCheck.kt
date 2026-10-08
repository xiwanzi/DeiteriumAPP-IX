package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.*
import android.graphics.Bitmap
import android.graphics.Rect
import android.os.Bundle
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveableStateHolder
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.*
import okhttp3.mockwebserver.*
import org.json.*
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.TimeUnit
import java.time.Instant

/** Production screens with isolated metadata/paging. No real account or purchase. */
object ShopNavigationCheck {
 fun run(test:Instrumentation) {
  val result=Bundle();val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
  val server=MockWebServer();var host:DeuteriumActivity?=null
  val context=object:ContextWrapper(test.targetContext){override fun getSharedPreferences(name:String,mode:Int)=super.getSharedPreferences("qa-shop-navigation-$name",mode)}
  val requests=CopyOnWriteArrayList<String>()
  val samples=JSONArray(test.targetContext.filesDir.resolve("qa-shop-products.json").readText())
  val categoryRows=JSONArray(test.targetContext.filesDir.resolve("qa-shop-categories.json").readText())
  val categoryNames=(0 until categoryRows.length()).associate{categoryRows.getJSONObject(it).let{x->x.getString("categoryId") to x.getString("name")}}
  val categoryGroups=(0 until samples.length()).associate{samples.getJSONObject(it).let{x->x.getString("categoryId") to x.getString("categoryGroup")}}
  val destinations=Class.forName("com.deuterium.app.uilab.Destination")
  val titleMethod=destinations.getDeclaredMethod("getTitle").apply{isAccessible=true}
  val iconMethod=destinations.getDeclaredMethod("getIcon").apply{isAccessible=true}
  val navLabels=destinations.enumConstants!!.map{titleMethod.invoke(it) as String}
  val navIcons=destinations.enumConstants!!.map{iconMethod.invoke(it) as androidx.compose.ui.graphics.vector.ImageVector}
  val rootNames=listOf("家具家居","灯具照明","绿植庭院","建筑建材","装饰摆件","其他")

  fun image(id:String)=JSONObject().put("assetId",id).put("url","https://fixture.example.invalid/$id").put("status","READY").put("contentType",if(id=="qa-banner")"image/png"else"image/webp").put("urlExpiresAt",Instant.now().plusSeconds(3600).toString())
  fun product(i:Int):JSONObject {
   val sample=if(i==3084)(0 until samples.length()).map{samples.getJSONObject(it)}.first{it.getString("categoryGroup")=="灯具照明"}else samples.getJSONObject(i%samples.length());val asset=sample.getString("assetId");val child=sample.getString("categoryId")
   return JSONObject().put("productId","product_$i").put("storeId","store_fixture").put("version",2).put("availableStock",JSONObject.NULL).put("images",JSONArray().put(image(asset)))
    .put("content",JSONObject().put("title",if(i==3084)"最后一件蓝色灯"else sample.getString("name")+if(i>=samples.length())" ${i+1}"else"").put("subtitle","隔离目录 · 页面布局验收").put("price","100.00").put("inventoryPolicy","UNLIMITED").put("stock",0).put("limitPerOrder",64).put("categoryId",child).put("brandId","brand").put("description","本机隔离商品，用于验证导航与连续浏览。").put("deliverySummary","游戏邮箱").put("estimatedDelivery","付款后投递").put("posterTone","LIGHT").put("galleryAssetIds",JSONArray().put(asset)).put("includedItems",JSONArray()).put("badges",if(i%7==0)JSONArray().put(NEW_PRODUCT_BADGE)else JSONArray()))
  }
  try {
   server.dispatcher=object:Dispatcher(){override fun dispatch(request:RecordedRequest):MockResponse {
    val u=request.requestUrl!!;val path=u.encodedPath.removePrefix("/api/v1");requests.add(u.toString());var next:String?=null
    val data:Any=when(path) {
     "/store/products"->{val offset=u.queryParameter("cursor")?.toInt() ?: 0;val q=u.queryParameter("q").orEmpty();val group=u.queryParameter("categoryGroup").orEmpty();val badge=u.queryParameter("badge").orEmpty()
      val ids=(0 until 3085).filter{i->val p=product(i).getJSONObject("content");val root=categoryGroups[p.getString("categoryId")];val child=categoryNames[p.getString("categoryId")];(group.isBlank()||root==group)&&(q.isBlank()||p.getString("title").contains(q)||q==root||q==child||(q==NEW_PRODUCT_BADGE&&i%7==0))&&(badge.isBlank()||(badge==NEW_PRODUCT_BADGE&&i%7==0))}.drop(offset).take(40)
      if(ids.size==40)next=(offset+40).toString();JSONArray(ids.map(::product))}
     "/store/categories"->{val a=JSONArray();rootNames.forEachIndexed{i,n->a.put(JSONObject().put("categoryId","root_$i").put("name",n).put("sortOrder",i))};for(i in 0 until categoryRows.length()){val c=JSONObject(categoryRows.getJSONObject(i).toString());c.put("image",image(c.getString("imageAssetId")));a.put(c)};a}
     "/store/brands"->JSONArray().put(JSONObject().put("brandId","brand").put("name","布局验收"))
     "/store/stores"->JSONArray().put(JSONObject().put("storeId","store_fixture"))
     "/store/stores/store_fixture/homepage"->JSONObject().put("banners",JSONArray().put(JSONObject().put("sectionId","activity").put("title","建造你的理想生活").put("image",image("qa-banner"))))
     else->if(path.startsWith("/store/products/product_"))product(path.substringAfterLast('_').toInt())else return MockResponse().setResponseCode(404).setBody("{}")
    }
    return MockResponse().setBodyDelay(90,TimeUnit.MILLISECONDS).setHeader("Content-Type","application/json").setBody(JSONObject().put("data",data).put("page",JSONObject().put("nextCursor",next ?: JSONObject.NULL).put("hasMore",next!=null)).toString())
   }}
   server.start();val api=BackendApi(context,server.url("/").toString())
   val save=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
   test.runOnMainSync{save.invoke(api,JSONObject().put("token","isolated-shop-navigation").put("user",JSONObject().put("gameId","ShopFixture").put("playerRef","fixture-shop")));ShopCatalog.clear()}
   val state=LabState(scope,userName="ShopFixture",api=api)
   host=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
   val imageScope=BackendApi.get(test.targetContext).financialScope();val images=AppImages.get(test.targetContext)
   for((id,file) in (0 until samples.length()).map{samples.getJSONObject(it).let{x->x.getString("assetId") to x.getString("imageFile")}}+listOf("qa-banner" to "qa-shop-banner.png")) {
    val bytes=test.targetContext.filesDir.resolve(file).readBytes();runBlocking{images.store.seed(ImageCacheStore.key(imageScope,"asset:$id"),bytes,ImageAccess("https://fixture.example.invalid/$id",ImageCacheStore.hash(bytes),if(id=="qa-banner")"image/png"else"image/webp"))}
    test.runOnMainSync{RemoteImageUrls.remember(image(id),imageScope)}
   }
   var stack by mutableStateOf(listOf("Shop"));var theme by mutableIntStateOf(1)
   fun open(route:String){stack=stack+route};fun back(){stack=stack.dropLast(1)}
   test.runOnMainSync{host!!.setContent{LabTheme(theme,true,true,GlassMaterials().overlay){val backdrop=rememberGraphicsLayer();val atmosphere=rememberGraphicsLayer();val saved=rememberSaveableStateHolder()
    val dark=theme==2
    SideEffect{androidx.core.view.WindowCompat.getInsetsController(host!!.window,host!!.window.decorView).apply{isAppearanceLightStatusBars=!dark;isAppearanceLightNavigationBars=!dark}}
    CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface,LocalOverlayBackdrop provides backdrop,LocalGlassBackdrop provides atmosphere,LocalHeaderGlassParameters provides GlassMaterials().header){
     Box(Modifier.fillMaxSize()){
      val route=stack.last()
      GlassScene(atmosphere,backdrop,Modifier.fillMaxSize()){saved.SaveableStateProvider(route){when {
       route=="Shop"->ShopPage(state,{open("product:$it")},WindowInsets.statusBars.asPaddingValues().calculateTopPadding(),"",::open)
       route.startsWith("shop-categories:")->ShopCategoriesPage(state,route.substringAfter(':'),::back,::open)
       route.startsWith("search:Shop:")->ShopSearchPage(state,android.net.Uri.decode(route.removePrefix("search:Shop:")),::back,::open,true)
       route=="search:Shop"->SearchPage("Shop",state,::back,::open)
       route.startsWith("product:")->{ProductPage(ShopCatalog.first{it.id==route.removePrefix("product:")},state,90.dp,{}, {_,_->});TextButton(::back,Modifier.statusBarsPadding()){Text("返回")}}
      }}}
      if(route=="Shop")LiquidGlass(Modifier.align(Alignment.BottomCenter).navigationBarsPadding().padding(horizontal=18.dp,vertical=9.dp).fillMaxWidth().height(70.dp),radius=30.dp,backdrop=backdrop,enabled=true,parameters=GlassMaterials().bottomBar){MovingGlassNav(navLabels,navIcons,0,true,emptySet()) {}}

     }
    }
   }}}
   fun nodes(n:AccessibilityNodeInfo?):List<AccessibilityNodeInfo> = if(n==null)emptyList()else listOf(n)+(0 until n.childCount).flatMap{nodes(n.getChild(it))}
   fun find(s:String):AccessibilityNodeInfo? {test.uiAutomation.clearCache();return nodes(test.uiAutomation.rootInActiveWindow).firstOrNull{it.text?.toString()==s||it.contentDescription?.toString()==s}}
   fun waitFor(label:String,condition:()->Boolean){repeat(160){if(condition())return;Thread.sleep(80)};error("Timeout: $label")}
   fun click(s:String){
    waitFor(s){find(s)!=null};val bounds=Rect();find(s)!!.getBoundsInScreen(bounds)
    val time=android.os.SystemClock.uptimeMillis()
    for(action in listOf(android.view.MotionEvent.ACTION_DOWN,android.view.MotionEvent.ACTION_UP)){
     val event=android.view.MotionEvent.obtain(time,android.os.SystemClock.uptimeMillis(),action,bounds.exactCenterX(),bounds.exactCenterY(),0)
     test.uiAutomation.injectInputEvent(event,true);event.recycle()
    }
    test.waitForIdleSync();Thread.sleep(300)
   }
   fun shot(name:String){Thread.sleep(400);test.waitForIdleSync();val b=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("shop-$name.png").outputStream().use{b.compress(Bitmap.CompressFormat.PNG,100,it)};b.recycle()}
   fun scroll(){val n=nodes(test.uiAutomation.rootInActiveWindow).filter{it.isScrollable}.maxByOrNull{val b=Rect();it.getBoundsInScreen(b);b.height()};n?.performAction(AccessibilityNodeInfo.ACTION_SCROLL_FORWARD);Thread.sleep(180)}
   waitFor("home"){find("查看更多")!=null&&state.commerce.network!!.storePage.items.size==40}
   check(state.commerce.network!!.recommendedPage.items.all{NEW_PRODUCT_BADGE in it.badges})
   shot("home-light");click("查看更多");waitFor("recommendation search"){state.commerce.network!!.searchPage.items.isNotEmpty()};check(requests.any{it.contains("badge=")&&it.contains("q=")});shot("recommendations");click("返回")
   click("家具家居");waitFor("children"){find("床与寝具")!=null};shot("furniture")
   test.runOnMainSync{repeat(10){state.commerce.network!!.categoryTree.add(ShopCategory("overflow_$it","额外分类 $it","root_0",100+it))}}
   click("更多");waitFor("directory"){find("分类")!=null}
   test.runOnMainSync{state.commerce.network!!.categoryTree.removeAll{it.id.startsWith("overflow_")}}
   shot("categories");click("座椅沙发")
   waitFor("keyword result"){state.commerce.network!!.searchPage.query=="座椅沙发"&&state.commerce.network!!.searchPage.items.isNotEmpty()};shot("category-search")
   val wanted=state.commerce.network!!.searchPage.items.first().name;click(wanted);waitFor("detail"){find("购买 $wanted")!=null};click("返回");waitFor("back to search"){stack.last().startsWith("search:")&&find(wanted)!=null}
   click("返回");waitFor("directory back"){stack.last().startsWith("shop-categories:")};click("返回");waitFor("shop back"){stack.last()=="Shop"};click("灯具照明");waitFor("short children"){state.commerce.network!!.storePage.items.firstOrNull()?.categoryId?.let{categoryGroups[it]}=="灯具照明"};waitFor("no extra category button"){find("更多")==null};shot("lighting")
   click("首页");waitFor("homepage again"){find("查看更多")!=null};repeat(16){scroll()};waitFor("pagination"){state.commerce.network!!.storePage.items.size>=80};check(find("更多分类")!=null){"Pinned header disappeared"};shot("feed")
   click("回到商城顶部");waitFor("scroll top"){find("查看更多")!=null}
   test.runOnMainSync{theme=2};shot("home-dark");test.runOnMainSync{theme=1}
   click("搜索商品、品牌或分类");waitFor("input"){nodes(test.uiAutomation.rootInActiveWindow).any{it.isEditable}}
   val field=nodes(test.uiAutomation.rootInActiveWindow).first{it.isEditable};field.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT,Bundle().apply{putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE,"最后一件")});click("搜索")
   waitFor("full directory search"){find("最后一件蓝色灯")!=null};shot("keyword-search")
   test.targetContext.filesDir.resolve("shop-requests.json").writeText(JSONArray(requests).toString(2));result.putString("home_category_directory_search_back_paging_pinned_header_themes","PASS");test.finish(-1,result)
  } catch(f:Throwable){runCatching{val b=test.uiAutomation.takeScreenshot();test.targetContext.filesDir.resolve("shop-failure.png").outputStream().use{b.compress(Bitmap.CompressFormat.PNG,100,it)};b.recycle()};test.targetContext.filesDir.resolve("shop-requests.json").writeText(JSONArray(requests).toString(2));result.putString("error",f.stackTraceToString());test.finish(1,result)}
  finally {test.targetContext.filesDir.resolve("shop-requests.json").writeText(JSONArray(requests).toString(2));scope.cancel();host?.let{test.runOnMainSync{it.finish()}};runCatching{server.shutdown()}}
 }
}
