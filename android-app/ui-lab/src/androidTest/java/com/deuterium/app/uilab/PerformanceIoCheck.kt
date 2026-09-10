package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.ContextWrapper
import android.content.SharedPreferences
import android.os.Bundle
import android.os.Looper
import kotlinx.coroutines.*
import okhttp3.Protocol
import okhttp3.Response
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import org.json.JSONObject
import org.json.JSONArray
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicReference
import java.util.concurrent.atomic.AtomicInteger

/** Real Android preferences and an isolated HTTP interceptor. No live credentials or operations.
 * Scoped interception keeps the non-debuggable target's production TLS policy unchanged. */
object PerformanceIoCheck {
    private class Barrier {val entered=CountDownLatch(1);val release=CountDownLatch(1)}
    fun run(test:Instrumentation) {
        val result=Bundle();val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val barrier=AtomicReference<Barrier?>();val requestCount=AtomicInteger()
        val storeBarrier=AtomicReference<Barrier?>();val storeVersion=AtomicInteger(1)
        val paths=ConcurrentHashMap<String,AtomicInteger>()
        val context=object:ContextWrapper(test.targetContext) {
            override fun getSharedPreferences(name:String,mode:Int):SharedPreferences {
                val actual=super.getSharedPreferences("qa-performance-io-$name",mode)
                if(name!="backend-v2")return actual
                return object:SharedPreferences by actual {
                    override fun edit():SharedPreferences.Editor {
                        val edit=actual.edit()
                        return object:SharedPreferences.Editor by edit {
                            override fun putString(key:String?,value:String?):SharedPreferences.Editor{edit.putString(key,value);return this}
                            override fun remove(key:String?):SharedPreferences.Editor{edit.remove(key);return this}
                            override fun commit():Boolean {
                                check(Looper.myLooper()!=Looper.getMainLooper()){"Pending request committed on UI thread"}
                                val success=edit.commit()
                                barrier.get()?.let{it.entered.countDown();check(it.release.await(8,TimeUnit.SECONDS))}
                                return success
                            }
                        }
                    }
                }
            }
        }
        try {
            // Only this check's prefixed fixture state is reset.
            context.getSharedPreferences("backend-v2",0).edit().clear().apply()
            val api=BackendApi(context,"https://performance.invalid")
            val client=api.http.newBuilder().addInterceptor{chain->
                val request=chain.request();check(request.url.host=="performance.invalid")
                check(Looper.myLooper()!=Looper.getMainLooper())
                requestCount.incrementAndGet()
                val path=request.url.encodedPath
                paths.computeIfAbsent(path){AtomicInteger()}.incrementAndGet()
                if(path.endsWith("/store/categories"))storeBarrier.get()?.let{it.entered.countDown();check(it.release.await(8,TimeUnit.SECONDS))}
                val data=when {
                    path.endsWith("/store/categories")->JSONArray().put(JSONObject().put("categoryId","fixture-category").put("name","装备"))
                    path.endsWith("/store/brands")->JSONArray().put(JSONObject().put("brandId","fixture-brand").put("name","固定店铺"))
                    path.endsWith("/store/products")->JSONArray().put(JSONObject().put("productId","fixture-product").put("version",storeVersion.get()).put("content",JSONObject()
                        .put("title","同一件商品").put("subtitle","固定说明").put("price","123.45").put("categoryId","fixture-category").put("brandId","fixture-brand")
                        .put("description","固定说明").put("deliverySummary","游戏内邮箱").put("estimatedDelivery","即刻交付")))
                    path.endsWith("/store/cart")->JSONObject().put("version",1).put("items",JSONArray())
                    path.endsWith("/chat/conversations")->JSONObject().put("conversationId","fixture-conversation")
                    path.endsWith("/messages")->JSONObject().put("messageId","fixture-message").put("content","同一条消息").put("sentAt","2026-09-11T04:00:00Z")
                        .put("sender",JSONObject().put("gameId","FixtureSelf").put("playerRef","FixtureSelf"))
                    else->JSONObject().put("ok",true)
                }
                Response.Builder().request(request).protocol(Protocol.HTTP_1_1).code(200).message("OK")
                    .body(JSONObject().put("data",data).toString().toResponseBody("application/json".toMediaType())).build()
            }.build()
            BackendApi::class.java.getDeclaredField("http").apply{isAccessible=true}.set(api,client)
            val save=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            fun login(owner:String)=test.runOnMainSync{save.invoke(api,JSONObject().put("token",owner).put("user",JSONObject().put("gameId",owner).put("playerRef",owner)))}
            login("FixtureSelf")
            fun blocked()=Barrier().also{barrier.set(it)}
            fun release(value:Barrier){barrier.set(null);value.release.countDown()}
            val first=blocked()
            val writing=scope.async{runCatching{api.saveTransfer(JSONObject().put("request",JSONObject().put("clientRequestId","original-request")))}}
            check(first.entered.await(8,TimeUnit.SECONDS))
            val read=scope.async{runCatching{api.request("GET","/performance-probe")}}
            test.runOnMainSync{} // Must run while the disk completion is deliberately held.
            check(!read.isCompleted&&requestCount.get()==0)
            release(first)
            runBlocking{check(writing.await().isSuccess);check(read.await().getOrThrow().getBoolean("ok"));api.saveTransfer(null)}
            result.putString("off_main_commit_and_recovery_barrier","PASS")

            val second=blocked();val requests=requestCount.get()
            val old=scope.async{runCatching{api.savePendingDirect("friend",JSONObject().put("clientMessageId","old-owner-request"))}}
            check(second.entered.await(8,TimeUnit.SECONDS))
            val oldRead=scope.async{runCatching{api.request("GET","/performance-probe")}}
            test.runOnMainSync{};login("FixtureOther");release(second)
            runBlocking {
                check((old.await().exceptionOrNull() as ApiFailure).code=="SESSION_CHANGED")
                check((oldRead.await().exceptionOrNull() as ApiFailure).code=="SESSION_CHANGED")
            }
            check(requestCount.get()==requests);check(api.pendingDirect("friend")==null)
            result.putString("account_change_during_disk_write","PASS")

            login("FixtureSelf");runBlocking{api.savePendingDirect("friend",null)}
            val third=blocked();val before=requestCount.get()
            lateinit var state:LabState
            test.runOnMainSync{Players.clear();Players.add(PlayerProfile("Friend",playerRef="friend"));state=LabState(scope,userName="FixtureSelf",api=api)}
            val sending=scope.async{state.sendDirect("Friend","同一条消息")}
            check(third.entered.await(8,TimeUnit.SECONDS))
            runBlocking{withContext(Dispatchers.Main){check(!state.sendDirect("Friend","同一条消息"))}}
            check(requestCount.get()==before);release(third)
            runBlocking{check(sending.await())}
            check(requestCount.get()==before+2);check(state.conversation("Friend").size==1)
            check(state.directPending["Friend"]==false);check(api.pendingDirect("friend")==null)
            result.putString("double_tap_during_disk_write_submits_once","PASS")

            val held=Barrier();storeBarrier.set(held)
            val firstStore=scope.async{state.commerce.network!!.refreshStore()}
            check(held.entered.await(8,TimeUnit.SECONDS))
            val secondStore=scope.async{state.commerce.network!!.refreshStore()}
            test.runOnMainSync{};storeBarrier.set(null);held.release.countDown()
            runBlocking{firstStore.await();secondStore.await()}
            fun count(path:String)=paths["/api/v1/store/$path"]?.get() ?: 0
            check(count("categories")==1&&count("brands")==1&&count("products")==1&&count("cart")==2)
            check(ShopCatalog.single().version==1L)
            storeVersion.set(2);runBlocking{withContext(Dispatchers.Main){state.commerce.network!!.refreshStore()}}
            check(count("products")==2&&count("cart")==3&&ShopCatalog.single().version==2L)
            result.putString("overlapping_catalog_reads_coalesce_but_later_refresh_stays_fresh","PASS")
            test.finish(-1,result)
        } catch(failure:Throwable){result.putString("failure",failure.stackTraceToString());test.finish(1,result)}
        finally{barrier.getAndSet(null)?.release?.countDown();storeBarrier.getAndSet(null)?.release?.countDown();scope.cancel()}
    }
}
