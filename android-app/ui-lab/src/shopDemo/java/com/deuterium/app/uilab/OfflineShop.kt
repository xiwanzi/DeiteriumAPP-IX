package com.deuterium.app.uilab

import android.content.Context
import android.net.Uri
import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import okio.Buffer
import org.json.*

/** This transport never calls chain.proceed: all records live inside the demo APK. */
internal class OfflineShop(private val context:Context):Interceptor {
    private val data=JSONObject(context.assets.open("shop-demo/catalog.json").bufferedReader().use{it.readText()})
    private val products=data.getJSONArray("products")
    private val categories=data.getJSONArray("categories")
    private val preferences=context.getSharedPreferences("offline-shop",Context.MODE_PRIVATE)
    private val cart=JSONObject(preferences.getString("cart","{}")!!)
    private var version=1L
    private val names=(0 until categories.length()).associate{categories.getJSONObject(it).let{c->c.getString("categoryId") to c.getString("name")}}
    private val groups=(0 until categories.length()).associate{categories.getJSONObject(it).let{c->c.getString("categoryId") to c.optString("parentCategoryId")}}
    val media:List<Pair<String,String>> = (0 until data.getJSONArray("media").length()).map{data.getJSONArray("media").getJSONObject(it).let{m->m.getString("assetId") to m.getString("file")}}
    fun image(id:String)=JSONObject().put("assetId",id).put("status","READY").put("url","https://offline.invalid/media/$id").put("contentType",if(id=="demo_banner")"image/png"else"image/webp").put("urlExpiresAt","2099-01-01T00:00:00Z")
    private fun product(value:JSONObject):JSONObject=JSONObject(value.toString()).put("images",JSONArray().put(image(value.getJSONObject("content").getString("coverAssetId"))))
    private fun cartView():JSONObject=JSONObject().put("version",version).put("items",JSONArray(cart.keys().asSequence().map{JSONObject().put("productId",it).put("quantity",cart.getInt(it))}.toList()))
    @Synchronized override fun intercept(chain:Interceptor.Chain):Response {
        val request=chain.request();val path=request.url.encodedPath.removePrefix("/api/v1")
        val root=JSONObject();var code=200
        val body=request.body?.let{b->val buffer=Buffer();b.writeTo(buffer);JSONObject(buffer.readUtf8())} ?: JSONObject()
        val value:Any=when {
            path=="/account/login"->JSONObject().put("token","offline-demo-only").put("user",JSONObject().put("playerRef","offline_demo").put("gameId","DemoPlayer").put("displayName","演示玩家").put("gameVerified",true))
            path=="/store/products"->{
                val q=request.url.queryParameter("q").orEmpty();val group=request.url.queryParameter("categoryGroup").orEmpty();val badge=request.url.queryParameter("badge").orEmpty()
                val offset=request.url.queryParameter("cursor")?.toIntOrNull() ?: 0;val limit=(request.url.queryParameter("limit")?.toIntOrNull() ?: 40).coerceIn(1,100)
                val all=(0 until products.length()).map{products.getJSONObject(it)}.filter{p->val c=p.getJSONObject("content");val category=c.getString("categoryId");val groupName=names[groups[category]];val badges=c.getJSONArray("badges").toString()
                    (group.isBlank()||groupName==group)&&(badge.isBlank()||badges.contains("\"$badge\""))&&(q.isBlank()||c.getString("title").contains(q,true)||names[category]?.contains(q,true)==true||groupName?.contains(q,true)==true||badges.contains(q)||q.equals("IKEA",true))}
                val selected=all.drop(offset).take(limit);val more=offset+selected.size<all.size
                root.put("page",JSONObject().put("hasMore",more).put("nextCursor",if(more)(offset+selected.size).toString()else JSONObject.NULL))
                JSONArray(selected.map(::product))
            }
            path.startsWith("/store/products/")->(0 until products.length()).map{products.getJSONObject(it)}.firstOrNull{it.getString("productId")==path.substringAfterLast('/')}?.let(::product) ?: JSONObject()
            path=="/store/categories"->JSONArray((0 until categories.length()).map{JSONObject(categories.getJSONObject(it).toString()).apply{optString("imageAssetId").takeIf{it.isNotBlank()}?.let{put("image",image(it))}}})
            path=="/store/brands"->JSONArray().put(JSONObject().put("brandId","demo_brand").put("name","家居精选"))
            path=="/store/stores"->JSONArray().put(JSONObject().put("storeId","demo_store").put("name","演示商城"))
            path=="/store/stores/demo_store/homepage"->JSONObject().put("banners",JSONArray().put(JSONObject().put("sectionId","demo_activity").put("title","建造你的理想生活").put("image",image("demo_banner"))))
            path.startsWith("/store/cart/items/")-> {
                val id=Uri.decode(path.removePrefix("/store/cart/items/").removeSuffix("/remove"));val quantity=body.optInt("quantity",0).coerceIn(0,64)
                if(quantity==0)cart.remove(id)else cart.put(id,quantity)
                version++;preferences.edit().putString("cart",cart.toString()).apply();cartView()
            }
            path=="/store/cart"->cartView()
            path.startsWith("/store/coupons")||path=="/orders"||path.startsWith("/market/")->JSONArray()
            else->{code=422;root.put("error",JSONObject().put("code","OFFLINE_DEMO").put("message","这是离线商城演示，不创建订单或执行真实交易。"));JSONObject()}
        }
        if(code==200)root.put("data",value)
        return Response.Builder().request(request).protocol(Protocol.HTTP_1_1).code(code).message(if(code==200)"OK"else"Offline demo").body(root.toString().toResponseBody("application/json".toMediaType())).build()
    }
}
