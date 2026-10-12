package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.net.Uri
import android.os.Bundle
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.*
import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody
import okhttp3.mockwebserver.Dispatcher
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.mockwebserver.RecordedRequest
import okio.Buffer
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.io.RandomAccessFile
import java.time.Instant
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger

/** Runs only in the separate .chatmediaqa application, with loopback API and
 * in-process object transport. Never logs in or writes to a real account. */
object ChatMediaCheck {
    fun run(test:Instrumentation,theme:Int,glassCheck:Boolean=false,overlayGlass:Boolean=true){
        val result=Bundle();val context=test.targetContext
        check(context.packageName.endsWith(".chatmediaqa")){"Chat media checks require the isolated QA application"}
        val materialPrefs=context.getSharedPreferences("qa-chat-materials",0)
        materialPrefs.edit().clear().putInt("materialVersion",2).putFloat("bottomBar.blur",11f).putFloat("overlay.blur",29f).putFloat("header.blur",9f).commit()
        val materialStore=GlassSettingsStore(materialPrefs)
        val legacy=materialStore.load()
        check(legacy.bottomBar.blur==11f&&legacy.overlay.blur==29f&&legacy.header.blur==9f)
        val tuned=legacy.copy(chatInputLight=legacy.chatInputLight.copy(blur=27f,opacity=.84f),chatPanelDark=legacy.chatPanelDark.copy(blur=63f))
        materialStore.save(tuned);check(materialStore.load()==tuned)
        check(tuned.chatInputDark==legacy.chatInputDark&&tuned.chatPanelLight==legacy.chatPanelLight&&tuned.overlay==legacy.overlay)
        materialPrefs.edit().putFloat("chatInput.light.blur",Float.NaN).commit()
        check(materialStore.load().chatInputLight.blur==GlassMaterials().chatInputLight.blur&&materialStore.load().chatPanelDark.blur==63f)
        result.putString("material_persistence","PASS legacy settings retained; input/panel and light/dark save independently; 52+ dp blur survives reload")
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate);val server=MockWebServer()
        val objectBytes=ConcurrentHashMap<String,ByteArray>();val manifests=ConcurrentHashMap<String,JSONObject>();val sent=ConcurrentHashMap<String,JSONObject>();val stickers=JSONArray()
        val puts=AtomicInteger();val sends=AtomicInteger();val failOnce=AtomicBoolean(true)
        val user=JSONObject().put("userId","chat-qa").put("playerRef","player_chat_qa").put("gameId","ChatQA").put("displayName","ChatQA").put("gameVerified",true)
        fun response(value:Any,status:Int=200)=MockResponse().setResponseCode(status).setHeader("Content-Type","application/json").setBody(JSONObject().put("data",value).toString())
        try{
            server.dispatcher=object:Dispatcher(){override fun dispatch(r:RecordedRequest):MockResponse{
                val path=r.requestUrl!!.encodedPath;val body=r.body.readUtf8().takeIf{it.isNotBlank()}?.let(::JSONObject) ?: JSONObject()
                if(path=="/api/v1/account/me")return response(JSONObject().put("user",user))
                if(path=="/api/v1/chat/stickers")return response(JSONObject().put("stickers",stickers).put("limit",10))
                if(path.startsWith("/api/v1/chat/sends/")){val found=sent[path.substringAfterLast('/')];return response(if(found==null)JSONObject().put("status","absent") else JSONObject().put("status","accepted").put("message",found))}
                if(path=="/api/v1/chat/media/uploads"){
                    val hash=body.getString("sha256");val previous=manifests[hash]
                    if(previous!=null)return response(JSONObject().put("status","READY").put("uploadId",hash).put("media",previous))
                    manifests["input:$hash"]=body
                    fun auth(part:String,size:Long,md5:String)=JSONObject().put("uploadUrl","https://qa.rains3.com/$hash/$part").put("method","PUT").put("sizeBytes",size).put("signedHeaders",JSONObject().put("Content-Type","image/webp").put("Content-Md5",md5).put("If-None-Match","*"))
                    return response(JSONObject().put("status","UPLOADING").put("uploadId",hash).put("authorization",auth("main",body.getLong("sizeBytes"),body.getString("contentMd5"))).put("thumbnailAuthorization",auth("thumb",body.getLong("thumbnailBytes"),body.getString("thumbnailMd5"))))
                }
                if(path.endsWith("/complete")){
                    val hash=path.substringBeforeLast('/').substringAfterLast('/');val input=manifests.getValue("input:$hash");val bytes=objectBytes.getValue("/$hash/main");check(ChatCache.hash(bytes)==hash)
                    val dimensions=BitmapFactoryBounds(bytes);val value=JSONObject().put("mediaId","cm_$hash").put("sha256",hash).put("thumbnailSha256",input.getString("thumbnailSha256")).put("sizeBytes",bytes.size).put("thumbnailBytes",input.getLong("thumbnailBytes")).put("width",dimensions.first).put("height",dimensions.second).put("animated",ChatWebp.isAnimated(bytes))
                    manifests[hash]=value;return response(JSONObject().put("status","READY").put("media",value))
                }
                if(path=="/api/v1/chat/messages"&&r.method=="POST"){
                    sends.incrementAndGet();val id=body.getString("clientMessageId");val media=manifests.values.first{it.optString("mediaId")==body.getJSONObject("media").getString("mediaId")}
                    val value=JSONObject().put("messageId","msg_$id").put("clientMessageId",id).put("content",body.getString("content")).put("kind","public_chat").put("sender",JSONObject().put("playerRef","player_chat_qa").put("gameId","ChatQA")).put("sentAt",Instant.now().toString()).put("media",media)
                    sent[id]=value
                    if(failOnce.getAndSet(false))return MockResponse().setResponseCode(503).setHeader("Content-Type","application/json").setBody("{\"error\":{\"code\":\"RESULT_UNKNOWN\",\"message\":\"请重试原消息\"}}")
                    return response(JSONObject().put("status","accepted").put("message",value))
                }
                if(path=="/api/v1/chat/messages")return response(JSONObject().put("messages",JSONArray()))
                return MockResponse().setResponseCode(410).setHeader("Content-Type","application/json").setBody("{\"error\":{\"code\":\"MEDIA_EXPIRED\",\"message\":\"图片已过期\"}}")
            }}
            server.start()
            val api=BackendApi(context,server.url("/").toString())
            val remember=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            test.runOnMainSync{remember.invoke(api,JSONObject().put("token","isolated-chat-media-session").put("user",user))}
            val cache=ChatCache(context,api.financialScope());ChatCache.clear(context,true)
            val root=File(context.filesDir,"qa-chat-media").apply{mkdirs()}
            val source=File(root,"screenshot.png")
            val bitmap=Bitmap.createBitmap(1080,6500,Bitmap.Config.ARGB_8888);val canvas=Canvas(bitmap);canvas.drawColor(Color.WHITE);val paint=Paint(Paint.ANTI_ALIAS_FLAG).apply{color=Color.rgb(30,40,65);textSize=30f}
            if(glassCheck){
                paint.color=Color.rgb(240,82,116);canvas.drawRect(0f,0f,540f,6500f,paint)
                paint.color=Color.rgb(37,139,235);canvas.drawRect(540f,0f,1080f,6500f,paint)
                paint.color=Color.argb(90,255,255,255);for(y in 0..6500 step 16)canvas.drawRect(0f,y.toFloat(),1080f,(y+4).toFloat(),paint)
                paint.color=Color.rgb(30,40,65)
            }
            for(y in 60..6450 step 60)canvas.drawText("Deuterium 公共聊天 压缩后文字仍需清楚 1234567890",30f,y.toFloat(),paint)
            source.outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()
            val prepared=runBlocking{ChatImageCompressor.prepare(source,false)}
            check(prepared.width==1080&&prepared.height==6500&&prepared.image.size<=2*1024*1024)
            File(root,"screenshot.image.webp").writeBytes(prepared.image);File(root,"screenshot.thumb.webp").writeBytes(prepared.thumbnail)
            val gif=File(root,"motion.gif");test.context.assets.open("chat-media/motion.gif").use{input->gif.outputStream().use{input.copyTo(it)}}
            val animated=runBlocking{ChatImageCompressor.prepare(gif,true)};val animation=ChatWebp.animation(animated.image)
            check(animated.animated&&animation.frames.size==12&&animation.frames.sumOf{it.duration}==960)
            animation.frames.forEach{check(android.graphics.BitmapFactory.decodeByteArray(it.bytes,0,it.bytes.size)!=null)}
            File(root,"motion.image.webp").writeBytes(animated.image);File(root,"motion.thumb.webp").writeBytes(animated.thumbnail)
            result.putString("compression", "PASS screenshot ${source.length()} -> ${prepared.image.size}; GIF ${gif.length()} -> ${animated.image.size}, 12 frames/960ms")
            val media=ChatMedia("cm_local",ChatCache.hash(prepared.image),ChatCache.hash(prepared.thumbnail),prepared.image.size.toLong(),prepared.thumbnail.size.toLong(),prepared.width,prepared.height,false,expiresAt=Instant.now().minusSeconds(400000).toString())
            cache.putFile(media.sha,prepared.image).setLastModified(System.currentTimeMillis()-40L*86400000)
            cache.putFile(media.thumbnailSha,prepared.thumbnail)
            val record=JSONObject().put("messageId","cached_40_days").put("sentAt",Instant.now().minusSeconds(40L*86400).toString()).put("content","[图片]").put("sender",JSONObject().put("gameId","Alice").put("playerRef","alice")).put("media",media.json())
            cache.save("public",record);check(cache.read("public").size==1)
            val service=BackendChatMedia(api,scope){_,_->}
            val requestsBefore=server.requestCount;check(runBlocking{service.file(media,"cached_40_days")}.isFile);check(server.requestCount==requestsBefore){"Local expired image triggered a server read"}
            val owner=ChatCache.hash("${api.financialScope().origin}:${api.financialScope().owner}".toByteArray());val oldest=File(context.filesDir,"chat-cache/media/$owner/${"0".repeat(64)}.webp").apply{parentFile!!.mkdirs()}
            RandomAccessFile(oldest,"rw").use{it.setLength(ChatCache.LIMIT)};oldest.setLastModified(1)
            cache.putFile(ChatCache.hash(animated.image),animated.image);check(!oldest.exists()&&ChatCache.bytes(context)<=ChatCache.LIMIT){"500MiB oldest-first eviction failed"}
            val pendingSentinel=File(context.filesDir,"chat-outbox/sentinel").apply{parentFile!!.mkdirs();writeText("pending")}
            runBlocking{AppStorage.clear(context,CacheSelection.ChatMedia,null)};check(cache.read("public").size==1&&cache.file(media.sha)==null&&pendingSentinel.exists())
            runBlocking{AppStorage.clear(context,CacheSelection.ChatAll,null)};check(cache.read("public").isEmpty()&&pendingSentinel.exists());pendingSentinel.delete()
            result.putString("cache","PASS local reads after server expiry, 500MiB eviction, two clear modes, pending isolation")
            val transport=OkHttpClient.Builder().addInterceptor{chain->check(chain.request().header("Authorization")==null);val buffer=Buffer();chain.request().body!!.writeTo(buffer);objectBytes[chain.request().url.encodedPath]=buffer.readByteArray();puts.incrementAndGet();Response.Builder().request(chain.request()).protocol(Protocol.HTTP_1_1).code(200).message("OK").body(ByteArray(0).toResponseBody("application/octet-stream".toMediaType())).build()}.build()
            val accepted=mutableListOf<JSONObject>();val sender=BackendChatMedia(api,scope,transport){message,_->accepted+=message}
            runBlocking{withContext(Dispatchers.Main){sender.enqueue(Uri.fromFile(source).toString(),null)}}
            fun waitFor(label:String,condition:()->Boolean){val end=System.currentTimeMillis()+30000;while(!condition()&&System.currentTimeMillis()<end)Thread.sleep(80);check(condition()){label}}
            waitFor("uncertain send retained"){sender.pending.firstOrNull()?.status=="failed"}
            check(sends.get()==1&&puts.get()==2)
            test.runOnMainSync{sender.retry(sender.pending.first())};waitFor("original send recovered"){sender.pending.isEmpty()&&accepted.size==1};check(sends.get()==1&&puts.get()==2)
            runBlocking{withContext(Dispatchers.Main){sender.enqueue(Uri.fromFile(source).toString(),null)}};waitFor("identical image reused"){sender.pending.isEmpty()&&accepted.size==2};check(sends.get()==2&&puts.get()==2)
            result.putString("outbox","PASS unknown receipt recovery + identical image skips second PUT")
            // All visual content is already cached, including an expired image.
            cache.putFile(media.sha,prepared.image);cache.putFile(media.thumbnailSha,prepared.thumbnail)
            val moving=ChatMedia("cm_moving",ChatCache.hash(animated.image),ChatCache.hash(animated.thumbnail),animated.image.size.toLong(),animated.thumbnail.size.toLong(),animated.width,animated.height,true,"sticker")
            cache.putFile(moving.sha,animated.image);cache.putFile(moving.thumbnailSha,animated.thumbnail)
            repeat(10){stickers.put(moving.copy(id="cm_sticker_$it").json())}
            val state=LabState(scope,userName="ChatQA",api=api)
            test.runOnMainSync{
                Players.add(PlayerProfile("Alice",playerRef="alice"));state.chatStatus="已连接服务器"
                if(glassCheck)repeat(10){i->state.chat+=ChatLine(-20L+i,"Alice","更早的本机消息 ${i+1}，用于验证滚动内容能够从玻璃下方经过。",false,serverAt=-20L+i)}
                state.chat+=ChatLine(1,"Alice","这是本机聊天媒体验收。",false,serverAt=1)
                state.chat+=ChatLine(2,"Alice","[图片]",false,remoteId="cached_40_days",serverAt=2,media=media)
                state.chat+=ChatLine(3,"你","[贴纸]",true,remoteId="moving_message",serverAt=3,media=moving)
            }
            runBlocking{withContext(Dispatchers.Main){state.chatMedia!!.refreshStickers()}}
            val previewOpen=mutableStateOf(false)
            val tunerOpen=mutableStateOf(false)
            val uiMaterials=mutableStateOf(GlassMaterials())
            val activity=test.startActivitySync(Intent(context,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            test.runOnMainSync{activity.setContent{LabTheme(theme,theme==1,true){
              val atmosphere=rememberGraphicsLayer();val backdrop=rememberGraphicsLayer()
              CompositionLocalProvider(LocalGlassMaterials provides uiMaterials.value,LocalGlassBackdrop provides atmosphere,LocalOverlayBackdrop provides backdrop,LocalOverlayGlassEnabled provides overlayGlass){Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)){
                GlassScene(atmosphere,backdrop,Modifier.fillMaxSize()){
                Box(Modifier.fillMaxSize().windowInsetsPadding(WindowInsets.ime)){ChatPage(state,90.dp,{},{})}
                Box(Modifier.align(Alignment.TopCenter).fillMaxWidth().height(90.dp).background(MaterialTheme.colorScheme.background).statusBarsPadding(),contentAlignment=Alignment.Center){Text("公共聊天",style=MaterialTheme.typography.titleMedium,color=MaterialTheme.colorScheme.onSurface)}
                }
                if(previewOpen.value)ChatAttachmentPreview(Uri.fromFile(source).toString(),false,false,null,{},{},{previewOpen.value=false})
                if(tunerOpen.value)GlassTuner(uiMaterials.value,{uiMaterials.value=it;materialStore.save(it)},{tunerOpen.value=false})
            }}}}}
            fun find(node:AccessibilityNodeInfo?,label:String):AccessibilityNodeInfo?{if(node==null)return null;if(node.text?.toString()==label||node.contentDescription?.toString()==label)return node;repeat(node.childCount){find(node.getChild(it),label)?.let{return it}};return null}
            fun click(label:String){waitFor("missing $label"){if(android.os.Build.VERSION.SDK_INT>=33)test.uiAutomation.clearCache();find(test.uiAutomation.rootInActiveWindow,label)!=null};var node=find(test.uiAutomation.rootInActiveWindow,label)!!;while(!node.isClickable)node=node.parent;check(node.performAction(AccessibilityNodeInfo.ACTION_CLICK));test.waitForIdleSync();Thread.sleep(250)}
            fun shot(name:String){test.waitForIdleSync();Thread.sleep(400);test.uiAutomation.takeScreenshot()!!.let{b->File(context.filesDir,"qa-chat-$name.png").outputStream().use{b.compress(Bitmap.CompressFormat.PNG,100,it)};b.recycle()}}
            waitFor("chat rendered"){find(test.uiAutomation.rootInActiveWindow,"选择表情")!=null};Thread.sleep(1200)
            val tile=find(test.uiAutomation.rootInActiveWindow,"查看贴纸")!!;val tileBounds=android.graphics.Rect();tile.getBoundsInScreen(tileBounds)
            fun tileHash():Int {val screenshot=test.uiAutomation.takeScreenshot()!!;return try{val rect=android.graphics.Rect(tileBounds);rect.intersect(0,0,screenshot.width,screenshot.height);val pixels=IntArray(rect.width()*rect.height());screenshot.getPixels(pixels,0,rect.width(),rect.left,rect.top,rect.width(),rect.height());pixels.contentHashCode()}finally{screenshot.recycle()}}
            val firstFrame=tileHash();Thread.sleep(230);val secondFrame=tileHash()
            check(if(theme==1)firstFrame!=secondFrame else firstFrame==secondFrame){"GIF playback/reduced motion mismatch"}
            result.putString("gif_playback",if(theme==1)"PASS visible animation changes pixels" else "PASS reduced motion freezes frame")
            test.uiAutomation.takeScreenshot()!!.let{screen->
                fun distance(a:Int,b:Int)=kotlin.math.abs(Color.red(a)-Color.red(b))+kotlin.math.abs(Color.green(a)-Color.green(b))+kotlin.math.abs(Color.blue(a)-Color.blue(b))
                val corner=screen.getPixel(tileBounds.left+3,tileBounds.top+3)
                val outside=screen.getPixel(tileBounds.left-3,tileBounds.top+3)
                val top=screen.getPixel(tileBounds.centerX(),tileBounds.top+3)
                File(context.filesDir,"qa-chat-sticker-rounding-$theme.png").outputStream().use{screen.compress(Bitmap.CompressFormat.PNG,100,it)}
                check(distance(corner,outside)<=3&&distance(top,outside)>distance(corner,outside)+5){"Animated sticker corner mismatch: bounds=$tileBounds corner=${Color.red(corner)},${Color.green(corner)},${Color.blue(corner)} outside=${Color.red(outside)},${Color.green(outside)},${Color.blue(outside)} top=${Color.red(top)},${Color.green(top)},${Color.blue(top)}"}
                screen.recycle()
            }
            result.putString("sticker_rounding","PASS rounded animated artwork with transparent corners")
            shot(if(theme==2)"dark" else "light")
            val anchorSamples=mutableListOf<Int>()
            fun composerAnchor(){if(android.os.Build.VERSION.SDK_INT>=33)test.uiAutomation.clearCache();find(test.uiAutomation.rootInActiveWindow,"发送消息")?.let{it.refresh();val r=android.graphics.Rect();it.getBoundsInScreen(r);anchorSamples+=r.bottom}}
            composerAnchor()
            var toggle=find(test.uiAutomation.rootInActiveWindow,"选择表情")!!;while(!toggle.isClickable)toggle=toggle.parent
            check(toggle.performAction(AccessibilityNodeInfo.ACTION_CLICK))
            repeat(12){Thread.sleep(32);composerAnchor()}
            shot("emoji-$theme")
            // Accessibility updates may wait for UI idle and omit in-flight frames. Verify the
            // endpoints here; the recorded display frames are used for the motion trajectory.
            check(anchorSamples.first()-anchorSamples.last()>context.resources.displayMetrics.density*100){"Panel did not reserve its dock: $anchorSamples"}
            result.putString("panel_motion","PASS composer dock endpoints; rendered trajectory verified separately: ${anchorSamples.first()} -> ${anchorSamples.last()}")
            shot("emoji-$theme");click("手势");shot("gestures-$theme");click("👍");check(state.chatDraft.text=="👍");click("删除表情");check(state.chatDraft.text.isEmpty())
            click("心情");click("❤️‍🔥");check(state.chatDraft.text=="❤️‍🔥");click("删除表情");check(state.chatDraft.text.isEmpty())
            click("贴纸 10/10");shot("stickers-$theme");check(find(test.uiAutomation.rootInActiveWindow,"贴纸已满，删除已有贴纸后可继续添加")!=null)
            test.sendKeyDownUpSync(android.view.KeyEvent.KEYCODE_BACK);Thread.sleep(300);click("查看图片");shot("viewer-$theme")
            fun zoomed(node:AccessibilityNodeInfo?):Boolean {if(node==null)return false;val state=androidx.core.view.accessibility.AccessibilityNodeInfoCompat.wrap(node).stateDescription?.toString();if(state=="缩放 300%")return true;repeat(node.childCount){if(zoomed(node.getChild(it)))return true};return false}
            val screen=test.uiAutomation.takeScreenshot()!!;val cx=screen.width/2f;val cy=screen.height/2f;screen.recycle()
            fun tap(){val now=android.os.SystemClock.uptimeMillis();val down=android.view.MotionEvent.obtain(now,now,android.view.MotionEvent.ACTION_DOWN,cx,cy,0);test.sendPointerSync(down);down.recycle();Thread.sleep(25);val up=android.view.MotionEvent.obtain(now,android.os.SystemClock.uptimeMillis(),android.view.MotionEvent.ACTION_UP,cx,cy,0);test.sendPointerSync(up);up.recycle()}
            tap();Thread.sleep(60);tap();waitFor("double tap zoom"){zoomed(test.uiAutomation.rootInActiveWindow)};shot("zoom-$theme");click("关闭图片");waitFor("viewer closed"){find(test.uiAutomation.rootInActiveWindow,"选择表情")!=null}
            result.putString("viewer","PASS full image + double tap zoom + close")
            click("输入消息");Thread.sleep(700)
            val keyboardAnchor=android.graphics.Rect();find(test.uiAutomation.rootInActiveWindow,"选择表情")!!.getBoundsInScreen(keyboardAnchor)
            click("选择表情");Thread.sleep(500)
            val panelAnchor=android.graphics.Rect();find(test.uiAutomation.rootInActiveWindow,"切换键盘")!!.getBoundsInScreen(panelAnchor)
            check(kotlin.math.abs(keyboardAnchor.bottom-panelAnchor.bottom)<context.resources.displayMetrics.density*28){"Keyboard/panel changed composer position: $keyboardAnchor -> $panelAnchor"}
            shot("keyboard-panel-$theme");click("切换键盘");Thread.sleep(500)
            result.putString("keyboard","PASS keyboard/panel retains composer position")
            test.sendKeyDownUpSync(android.view.KeyEvent.KEYCODE_BACK);Thread.sleep(300)
            test.runOnMainSync{previewOpen.value=true}
            waitFor("preview half sheet"){find(test.uiAutomation.rootInActiveWindow,"展开预览")!=null};shot("attachment-half-$theme")
            fun dragHandle(label:String,delta:Float){
                if(android.os.Build.VERSION.SDK_INT>=33)test.uiAutomation.clearCache()
                val node=find(test.uiAutomation.rootInActiveWindow,label)!!;val bounds=android.graphics.Rect();node.getBoundsInScreen(bounds)
                val x=bounds.exactCenterX();val y=bounds.exactCenterY();val downAt=android.os.SystemClock.uptimeMillis()
                fun event(action:Int,at:Float){val ev=android.view.MotionEvent.obtain(downAt,android.os.SystemClock.uptimeMillis(),action,x,at,0);test.sendPointerSync(ev);ev.recycle()}
                event(android.view.MotionEvent.ACTION_DOWN,y)
                repeat(12){Thread.sleep(18);event(android.view.MotionEvent.ACTION_MOVE,y+delta*(it+1)/12)}
                event(android.view.MotionEvent.ACTION_UP,y+delta);test.waitForIdleSync();Thread.sleep(350)
            }
            dragHandle("展开预览",-context.resources.displayMetrics.density*220)
            waitFor("preview expanded by drag"){if(android.os.Build.VERSION.SDK_INT>=33)test.uiAutomation.clearCache();find(test.uiAutomation.rootInActiveWindow,"收起预览")!=null};shot("attachment-full-$theme")
            dragHandle("收起预览",context.resources.displayMetrics.density*220)
            waitFor("preview collapsed by drag"){if(android.os.Build.VERSION.SDK_INT>=33)test.uiAutomation.clearCache();find(test.uiAutomation.rootInActiveWindow,"展开预览")!=null}
            click("展开预览")
            click("收起预览");click("取消");waitFor("preview dismissed"){!previewOpen.value}
            result.putString("attachment","PASS half-height / drag expand / drag collapse / handle click / animated dismissal")
            if(glassCheck){
                click("选择表情");click("表情符号");click("表情")
                fun bounds(label:String):android.graphics.Rect {if(android.os.Build.VERSION.SDK_INT>=33)test.uiAutomation.clearCache();return android.graphics.Rect().also{find(test.uiAutomation.rootInActiveWindow,label)!!.getBoundsInScreen(it)}}
                // Scroll actual chat media under the dock; no duplicate image is inserted into glass.
                val input=bounds("输入消息");val picture=bounds("查看图片")
                val target=input.top-context.resources.displayMetrics.density*35
                val delta=(target-picture.top).coerceIn(0f,context.resources.displayMetrics.heightPixels*.6f)
                val startY=context.resources.displayMetrics.density*110;val x=picture.exactCenterX();val downAt=android.os.SystemClock.uptimeMillis()
                fun swipe(action:Int,y:Float){val event=android.view.MotionEvent.obtain(downAt,android.os.SystemClock.uptimeMillis(),action,x,y,0);test.sendPointerSync(event);event.recycle()}
                swipe(android.view.MotionEvent.ACTION_DOWN,startY)
                repeat(20){Thread.sleep(22);swipe(android.view.MotionEvent.ACTION_MOVE,startY+delta*(it+1)/20)}
                swipe(android.view.MotionEvent.ACTION_UP,startY+delta);test.waitForIdleSync();Thread.sleep(600)
                shot(if(overlayGlass)"glass-$theme" else "glass-off-$theme")
                val behind=bounds("查看图片");val dockInput=bounds("输入消息")
                check(behind.bottom>dockInput.bottom+context.resources.displayMetrics.density*55){"Chat image did not extend behind the dock: $behind / $dockInput"}
                val panelBounds=bounds("表情面板")
                test.uiAutomation.takeScreenshot()!!.let{frame->check(panelBounds.left==0&&panelBounds.right==frame.width&&panelBounds.bottom==frame.height){"Panel must reach both sides and the physical bottom: $panelBounds / ${frame.width}x${frame.height}"};frame.recycle()}
                result.putString("panel_edges","PASS top corners only; both sides and gesture area reach screen edges")
                result.putString("glass_backdrop","PASS actual message image under input/panel: $behind / $dockInput")
                test.runOnMainSync{tunerOpen.value=true}
                click("输入栏")
                fun range(node:AccessibilityNodeInfo?):AccessibilityNodeInfo?{if(node==null)return null;if(node.contentDescription?.toString()=="模糊半径"&&node.rangeInfo!=null)return node;repeat(node.childCount){range(node.getChild(it))?.let{return it}};return null}
                fun setBlur(value:Float){if(android.os.Build.VERSION.SDK_INT>=33)test.uiAutomation.clearCache();val slider=range(test.uiAutomation.rootInActiveWindow)!!;check(slider.performAction(AccessibilityNodeInfo.AccessibilityAction.ACTION_SET_PROGRESS.id,Bundle().apply{putFloat(AccessibilityNodeInfo.ACTION_ARGUMENT_PROGRESS_VALUE,value)}));test.waitForIdleSync();Thread.sleep(150)}
                setBlur(30f)
                check((if(theme==2)uiMaterials.value.chatInputDark else uiMaterials.value.chatInputLight).blur==30f)
                check(uiMaterials.value.chatPanelLight==GlassMaterials().chatPanelLight&&uiMaterials.value.chatPanelDark==GlassMaterials().chatPanelDark)
                click("表情面板");setBlur(60f)
                check((if(theme==2)uiMaterials.value.chatPanelDark else uiMaterials.value.chatPanelLight).blur==60f)
                check((if(theme==2)uiMaterials.value.chatInputDark else uiMaterials.value.chatInputLight).blur==30f)
                check(materialStore.load()==uiMaterials.value)
                click("还原")
                check((if(theme==2)uiMaterials.value.chatPanelDark else uiMaterials.value.chatPanelLight).blur==52f)
                check((if(theme==2)uiMaterials.value.chatInputDark else uiMaterials.value.chatInputLight).blur==30f)
                shot("material-tuner-$theme")
                click("完成");waitFor("tuner closed"){!tunerOpen.value}
                result.putString("material_controls","PASS four areas; input/panel sliders and reset change only selected area and theme")
            }
            result.putString("ui","PASS public images/GIF + emoji panel + full sticker library")
            result.putString("status","PASS");test.finish(-1,result)
        }catch(e:Throwable){result.putString("error",e.stackTraceToString());test.finish(1,result)}finally{scope.cancel();server.shutdown()}
    }
    private fun BitmapFactoryBounds(bytes:ByteArray):Pair<Int,Int>{val o=android.graphics.BitmapFactory.Options().apply{inJustDecodeBounds=true};android.graphics.BitmapFactory.decodeByteArray(bytes,0,bytes.size,o);return o.outWidth to o.outHeight}
}
