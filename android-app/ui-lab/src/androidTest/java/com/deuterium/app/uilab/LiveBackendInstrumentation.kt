package com.deuterium.app.uilab

import android.app.Instrumentation
import android.os.Bundle
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.*
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.MediaType.Companion.toMediaType
import org.json.JSONObject
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

/** Separate QA APK only. Never packaged into the user-facing APK. */
class LiveBackendInstrumentation : Instrumentation() {
    private var options=Bundle()
    override fun onCreate(arguments:Bundle?) { super.onCreate(arguments);options=arguments ?: Bundle();start() }
    override fun onStart() {
        if(options.getString("accountErasure")=="true"){AccountErasureCheck.run(this);return}
        if(options.getString("authRegistration")=="true"){AuthRegistrationCheck.run(this);return}
        options.getString("overdrawPageCapture")?.let{OverdrawPageCapture.run(this,it);return}
        options.getString("overdrawScrollBenchmark")?.let{OverdrawPageCapture.run(this,it,benchmark=true);return}
        options.getString("overdrawDiagnostics")?.let{OverdrawDiagnostics.run(this,experiments=it=="experiments");return}
        options.getString("performanceVisual")?.let{PerformanceVisualCheck.run(this,it);return}
        if(options.getString("performanceIO")=="true"){PerformanceIoCheck.run(this);return}
        if(options.getString("couponArrivals")=="true"){CouponArrivalCheck.run(this);return}
        if(options.getString("commercePromotions")=="true"){CommercePromotionsCheck.run(this);return}
        options.getString("launcherIconSelect")?.let{LauncherIconCheck.select(this,it);return}
        if(options.getString("launcherIcons")=="true"){LauncherIconCheck.run(this);return}
        if(options.getString("sakiLayout")=="true"){SakiLayoutCheck.run(this);return}
        if(options.getString("historyLayout")=="true"){HistoryLayoutCheck.run(this);return}
        if(options.getString("installerHandoff")=="true"){InstallerHandoffCheck.run(this);return}
        if(options.getString("releaseV204")=="true"){ReleaseV204Check.run(this);return}
        if(options.getString("releaseV203")=="true"){ReleaseV203Check.run(this);return}
        if(options.getString("paymentTiming")=="true"){PaymentTimingCheck.run(this);return}
        if(options.getString("contactsLayout")=="true"){ContactsLayoutCheck.run(this);return}
        if(options.getString("imageCache")=="true"){ImageCacheInstrumentation.run(this);return}
        if(options.getString("storageLayout")=="true"){StorageLayoutCheck.run(this,options.getString("theme")?.toIntOrNull() ?: 1,options.getString("clear")=="true");return}
        if(options.getString("avatarLayout")=="true"){AvatarLayoutCheck.run(this);return}
        val result=Bundle()
        val fixtureFile=targetContext.filesDir.resolve("qa-session.json")
        var api:BackendApi?=null
        try {
            check(fixtureFile.isFile){"QA session fixture is missing"}
            val fixture=JSONObject(fixtureFile.readText())
            fixtureFile.delete()
            val service=BackendApi.get(targetContext);api=service
            // The issued short-lived session is inserted only from this separately installed test APK.
            // Production login code and its validation remain unchanged.
            val save=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            runOnMainSync{save.invoke(service,fixture)}
            runBlocking {
                service.verifySession()
                val user=service.request("GET","/account/me").getJSONObject("user")
                check(user.getString("playerRef")==fixture.getJSONObject("user").getString("playerRef")){"Authenticated player mismatch"}
                val publicMessages=service.request("GET","/chat/messages?limit=20").getJSONArray("messages")
                options.getString("publicMessageId")?.let { expected ->
                    check((0 until publicMessages.length()).any { publicMessages.getJSONObject(it).optString("messageId")==expected }){"Authorized public message was not received"}
                    result.putString("authorized_existing_public_message","PASS")
                }
                val opened=CountDownLatch(1)
                val socket=BackendChat(service.http,service.baseUrl,service.token!!,{}, { _,_ -> },{opened.countDown()})
                try{socket.connect();check(opened.await(15,TimeUnit.SECONDS)){"Authenticated chat socket did not open"}}finally{socket.close()}
                result.putString("account","PASS")
                result.putString("public_chat_history_and_socket","PASS")
                if(options.getString("wallet")=="true") {
                    val balance=service.request("GET","/wallet/balance").getJSONObject("balance")
                    apiCents(balance.getString("amount"))
                    service.request("GET","/wallet/records?limit=20").getJSONArray("records")
                    val recipients=service.request("GET","/wallet/recipients/search?query=luoyinwuchen1&type=auto").getJSONArray("candidates")
                    check(recipients.length()>0){"Authorized recipient was not resolved"}
                    result.putString("wallet_read_and_recipient_resolution","PASS")
                    options.getString("transferId")?.let { transferId ->
                        val transfer=service.request("GET","/wallet/transfers/$transferId").getJSONObject("transfer")
                        check(transfer.getString("transferId")==transferId){"Transfer identity differs"}
                        check(transfer.getString("amount")=="1.00"){"Authorized transfer amount differs"}
                        check(transfer.getString("status")=="success"){"Authorized transfer is not confirmed successful"}
                        targetContext.filesDir.resolve("qa-wallet-result.json").writeText(transfer.toString(2))
                        result.putString("authorized_existing_transfer_read","PASS")
                    }
                }
                if(options.getString("social")=="true") {
                    service.request("GET","/chat/player-directory").getJSONArray("players")
                    service.request("GET","/chat/follows").getJSONArray("players")
                    service.request("GET","/chat/conversations").getJSONArray("items")
                    service.request("GET","/announcements").getJSONArray("items")
                    service.request("GET","/notifications").getJSONArray("items")
                    service.request("GET","/notifications/preferences").getLong("version")
                    result.putString("social_read_contracts","PASS")
                }
                if(options.getString("catalog")=="true") {
                    service.request("GET","/store/products?limit=20").getJSONArray("items")
                    service.request("GET","/market/listings?limit=20").getJSONArray("items")
                    service.request("GET","/market/me/listings?limit=20").getJSONArray("items")
                    service.request("GET","/store/cart").getLong("version")
                    result.putString("catalog_read_contracts","PASS")
                }
                if(options.getString("financeLists")=="true") {
                    val installedVersion=targetContext.packageManager.getPackageInfo(targetContext.packageName,0).longVersionCode
                    check(installedVersion>=20003L){"Unexpected installed APK version"}
                    val counts=JSONObject();val routes=org.json.JSONArray()
                    for(path in listOf("/orders","/commissions","/commissions/me","/store/products","/market/listings","/market/me/listings")) {
                        val first=service.request("GET","$path?limit=100")
                        val items=first.getJSONArray("items")
                        val all=service.listAll(path)
                        val page=first.optJSONObject("_page")
                        if(page?.optBoolean("hasMore")!=true)check(all.size==items.length()){"Page count differs"}
                        counts.put(path,all.size)
                        routes.put(JSONObject().put("method","GET").put("path",path).put("firstPageItems",items.length())
                            .put("allItems",all.size).put("page",page ?: JSONObject.NULL).put("parsedBy","installed BackendApi.listAll"))
                    }
                    val evidence=JSONObject().put("timestamp",java.time.Instant.now().toString()).put("versionCode",installedVersion)
                        .put("apiBaseUrl",service.baseUrl).put("serverRevisionReported",options.getString("serverRevision"))
                        .put("businessWrites",0).put("routes",routes)
                    targetContext.filesDir.resolve("qa-financial-lists.json").writeText(evidence.toString(2))
                    result.putString("financial_lists_and_pagination","PASS")
                    result.putString("financial_list_counts",counts.toString())
                }
                if(options.getString("mediaRecovery")=="true") {
                    val transferId="transfer_732103cd41d23562ef51c6ebae75edfde4995431"
                    val transfer=service.request("GET","/wallet/transfers/$transferId").getJSONObject("transfer")
                    check(transfer.getString("status")=="success"){"Original transfer has not recovered on the backend"}
                    val listing=service.request("GET","/market/listings?limit=1").getJSONArray("items").getJSONObject(0)
                    val photoId=listing.getJSONArray("photoAssetIds").getString(0)
                    val profile=service.request("GET","/players/${transfer.getJSONObject("recipient").getString("playerRef")}")
                    val avatarId=profile.getJSONObject("avatar").getString("assetId")
                    val photo=BackendAssets.bitmap(targetContext,"asset:$photoId",1000)
                    val avatar=BackendAssets.bitmap(targetContext,"asset:$avatarId",640)
                    val evidence=JSONObject().put("listingId",listing.getString("listingId")).put("photoAssetId",photoId)
                        .put("photoWidth",photo.width).put("photoHeight",photo.height).put("avatarAssetId",avatarId)
                        .put("avatarWidth",avatar.width).put("avatarHeight",avatar.height)
                    check(photo.width>0&&avatar.width>0);photo.recycle();avatar.recycle()
                    check(service.pendingTransfer()==null){"Do not replace an existing local pending request"}
                    val original=JSONObject().put("clientRequestId",transfer.getString("clientRequestId"))
                        .put("recipientPlayerRef",transfer.getJSONObject("recipient").getString("playerRef"))
                        .put("amount",transfer.getString("amount")).put("note",transfer.optString("note"))
                    service.saveTransfer(JSONObject().put("request",original).put("transferId",transferId))
                    val testScope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
                    val state=withContext(Dispatchers.Main){LabState(testScope,userName=service.userName,api=service)}
                    try {
                        check(state.transferPending)
                        check(withContext(Dispatchers.Main){state.recoverTransfer()}){"Original pending request did not resolve"}
                        check(service.pendingTransfer()==null&&!state.transferPending&&state.transferSucceeded){"Next transfer remained blocked"}
                        check(state.completedTransferKey==original.getString("clientRequestId"))
                        evidence.put("transferId",transferId).put("operationId",transfer.getString("operationId"))
                            .put("pendingCleared",true).put("newTransferUnlocked",true).put("businessWrites",0)
                        targetContext.filesDir.resolve("qa-media-recovery.json").writeText(evidence.toString(2))
                        result.putString("real_listing_and_recipient_avatar_decode","PASS")
                        result.putString("original_pending_transfer_recovers_and_unlocks","PASS")
                    }finally{state.close();testScope.cancel()}
                }
                if(options.getString("assets")=="true") {
                    val imageFile=targetContext.cacheDir.resolve("qa-asset.png")
                    val bitmap=android.graphics.Bitmap.createBitmap(16,16,android.graphics.Bitmap.Config.ARGB_8888)
                    bitmap.eraseColor(android.graphics.Color.rgb(kotlin.random.Random.nextInt(256),kotlin.random.Random.nextInt(256),kotlin.random.Random.nextInt(256)))
                    imageFile.outputStream().use{check(bitmap.compress(android.graphics.Bitmap.CompressFormat.PNG,100,it))};bitmap.recycle()
                    var asset:String?=null
                    try {
                        asset=BackendAssets.upload(targetContext,android.net.Uri.fromFile(imageFile).toString(),"AVATAR","PROFILE")
                        val downloaded=BackendAssets.bitmap(targetContext,asset,32)
                        check(downloaded.width==16&&downloaded.height==16){"Asset dimensions differ"};downloaded.recycle()
                        result.putString("rainyun_signed_upload_verification_and_download","PASS")
                    }finally{
                        imageFile.delete()
                        asset?.let{service.request("POST","/assets/${it.removePrefix("asset:")}/remove",JSONObject().put("clientRequestId",java.util.UUID.randomUUID().toString()))}
                    }
                }
                if(options.getString("ai")=="true") {
                    val aiScope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
                    val state=withContext(Dispatchers.Main){LabState(aiScope,userName=service.userName,api=service)}
                    try{
                        val ai=state.ai!!
                        withContext(Dispatchers.Main){ai.refresh()}
                        val policy=service.request("GET","/ai/me")
                        check(policy.getJSONObject("quota").getBoolean("unlimited")){"Authorized administrator exemption missing"}
                        val prompt="请联网查阅 DeepSeek 官方 Responses API 文档，说明流式响应以哪些事件结束，并附上官方文档链接。回答控制在150字以内。"
                        check(withContext(Dispatchers.Main){ai.send(prompt)} ){"Real AI stream did not complete"}
                        val message=withContext(Dispatchers.Main){state.conversation("AI 助手").last{!it.mine}}
                        check(message.aiStatus=="completed"&&message.text.isNotBlank()&&message.searchUsed){"Search answer not confirmed"}
                        check(message.sources.any{it.url.startsWith("https://api-docs.deepseek.com/")}){"No actual official source link returned"}
                        val originalKey=ai.lastRequestId!!
                        val request=JSONObject().put("clientMessageId",originalKey).put("content",prompt)
                        var replayedId:String?=null
                        withContext(Dispatchers.IO){
                            service.http.newCall(Request.Builder().url(service.baseUrl+"/api/v1/ai/chat/stream").header("Authorization","Bearer "+service.token)
                                .post(request.toString().toRequestBody("application/json".toMediaType())).build()).execute().use{response->check(response.isSuccessful);readAiEvents(response.body!!.charStream()){event,value->if(event=="done")replayedId=value.getJSONObject("message").getString("messageId")}}
                        }
                        check(replayedId==message.remoteId){"Original request did not recover the same reply"}
                        val evidence=JSONObject().put("clientRequestId",originalKey).put("assistantMessageId",message.remoteId).put("searchUsed",message.searchUsed).put("contentChars",message.text.length)
                            .put("sources",org.json.JSONArray(message.sources.map{JSONObject().put("title",it.title).put("url",it.url).put("origin",it.origin)})).put("sameIdReplay",true).put("unlimited",true)
                        targetContext.filesDir.resolve("qa-ai-result.json").writeText(evidence.toString(2))
                        result.putString("real_ai_search_stream_and_same_id_recovery","PASS")
                    }finally{state.close();aiScope.cancel()}
                }
            }
            result.putString("stream","Live backend checks passed; no balances or secrets logged.\n")
            finish(android.app.Activity.RESULT_OK,result)
        } catch(error:Throwable) {
            if(error is NoSuchMethodError) result.putString("abi_error",error.message+"\n"+error.stackTrace.take(5).joinToString("\n"))
            result.putString("stream","Live backend check failed: ${error.javaClass.simpleName}${(error as? ApiFailure)?.let{": "+it.code}.orEmpty()}.\n")
            finish(android.app.Activity.RESULT_CANCELED,result)
        } finally {
            fixtureFile.delete()
            if(options.getString("keepUiSession")!="true")api?.let{service->runOnMainSync{service.forgetSession()}}
        }
    }
}
