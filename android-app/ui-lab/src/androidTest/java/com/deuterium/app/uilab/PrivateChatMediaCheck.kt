package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.ContentValues
import android.content.Intent
import android.graphics.Bitmap
import android.net.Uri
import android.os.Bundle
import android.provider.MediaStore
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.*
import okhttp3.mockwebserver.*
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.time.Instant
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.atomic.AtomicBoolean

/** Isolated loopback service: never sends to a real player or object store. */
object PrivateChatMediaCheck {
    fun run(test:Instrumentation,theme:Int){
        val context=test.targetContext;val result=Bundle()
        check(context.packageName.endsWith(".chatmediaqa"))
        test.uiAutomation.serviceInfo=test.uiAutomation.serviceInfo.apply{flags=flags or android.accessibilityservice.AccessibilityServiceInfo.FLAG_REPORT_VIEW_IDS}
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        var senderScope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val server=MockWebServer();var galleryUri:Uri?=null
        try{
            val root=File(context.filesDir,"qa-private-media").apply{mkdirs()}
            val photo=File(root,"photo.png")
            Bitmap.createBitmap(420,300,Bitmap.Config.ARGB_8888).let{image->
                image.eraseColor(android.graphics.Color.rgb(57,142,211))
                android.graphics.Canvas(image).drawText("PRIVATE CHAT",24f,140f,android.graphics.Paint().apply{color=android.graphics.Color.WHITE;textSize=36f})
                photo.outputStream().use{image.compress(Bitmap.CompressFormat.PNG,100,it)};image.recycle()
            }
            val gif=File(root,"motion.gif");test.context.assets.open("chat-media/motion.gif").use{input->gif.outputStream().use{input.copyTo(it)}}
            val photoBytes=runBlocking{ChatImageCompressor.prepare(photo,false)}
            val gifBytes=runBlocking{ChatImageCompressor.prepare(gif,true)}
            fun media(id:String,p:PreparedChatImage,type:String)=ChatMedia(id,ChatCache.hash(p.image),ChatCache.hash(p.thumbnail),p.image.size.toLong(),p.thumbnail.size.toLong(),p.width,p.height,p.animated,type)
            val picture=media("cm_private_photo",photoBytes,"image");val moving=media("cm_private_gif",gifBytes,"sticker")
            val user=JSONObject().put("userId","private-qa").put("playerRef","player_private_qa").put("gameId","PrivateQA").put("gameVerified",true)
            val journal=ConcurrentHashMap<String,JSONObject>();val posts=CopyOnWriteArrayList<Pair<String,String>>()
            val lostResponse=AtomicBoolean(true);val rejectBob=AtomicBoolean(true);val forceHistoryPage=AtomicBoolean(true)
            fun response(value:Any)=MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",value).toString())
            fun failure()=MockResponse().setResponseCode(503).setHeader("Content-Type","application/json").setBody("{\"error\":{\"code\":\"RESULT_UNKNOWN\",\"message\":\"私聊验收重试\"}}")
            server.dispatcher=object:Dispatcher(){override fun dispatch(r:RecordedRequest):MockResponse{
                val path=r.requestUrl!!.encodedPath;val body=r.body.readUtf8().takeIf{it.isNotBlank()}?.let(::JSONObject) ?: JSONObject()
                if(path=="/api/v1/account/me")return response(JSONObject().put("user",user))
                if(path=="/api/v1/chat/stickers")return response(JSONObject().put("stickers",JSONArray().put(moving.json())).put("limit",10))
                if(path=="/api/v1/chat/media/uploads"){
                    val m=listOf(picture,moving).first{it.sha==body.getString("sha256")}
                    return response(JSONObject().put("status","READY").put("uploadId",m.id).put("media",m.json()))
                }
                if(path=="/api/v1/chat/conversations"&&r.method=="POST")return response(JSONObject().put("conversationId","conv_"+body.getString("otherPlayerRef").removePrefix("player_")))
                if(path.endsWith("/messages")&&r.method=="POST"){
                    check(path.startsWith("/api/v1/chat/conversations/")){"Private media escaped to public chat"}
                    val conversation=path.split('/')[5];val id=body.getString("clientMessageId")
                    posts+=conversation to body.toString()
                    journal[id]?.let{return response(it)}
                    if(conversation=="conv_Bob"&&rejectBob.get())return failure()
                    val message=JSONObject().put("messageId","msg_$id").put("clientMessageId",id).put("conversationId",conversation)
                        .put("kind","private_chat").put("content",body.getString("content")).put("sender",JSONObject().put("gameId","PrivateQA").put("playerRef","player_private_qa")).put("sentAt",Instant.now().toString())
                    body.optJSONObject("media")?.let{m->message.put("media",(if(m.getString("mediaId")==moving.id)moving else picture).copy(type=m.getString("type")).json())}
                    journal[id]=message
                    if(body.has("media")&&lostResponse.getAndSet(false))return failure()
                    return response(message)
                }
                if(path.endsWith("/messages")){
                    if(forceHistoryPage.getAndSet(false))return MockResponse().setHeader("Content-Type","application/json").setBody("{\"data\":{\"items\":[]},\"page\":{\"hasMore\":true,\"nextCursor\":\"next-page\"}}")
                    return response(JSONObject().put("items",JSONArray(journal.values.filter{it.getString("conversationId")==path.split('/')[5]})).put("hasMore",false))
                }
                if(path.endsWith("/read"))return response(JSONObject().put("unreadCount",0))
                if(path.contains("/chat/media/files/"))return MockResponse().setResponseCode(410).setBody("{\"error\":{\"code\":\"MEDIA_EXPIRED\",\"message\":\"图片已过期\"}}")
                return response(JSONObject().put("items",JSONArray()).put("hasMore",false))
            }}
            server.start();val api=BackendApi(context,server.url("/").toString())
            val remember=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            test.runOnMainSync{remember.invoke(api,JSONObject().put("token","private-local-qa").put("user",user))}
            val cache=ChatCache(context,api.financialScope())
            for((m,p) in listOf(picture to photoBytes,moving to gifBytes)){cache.putFile(m.sha,p.image);cache.putFile(m.thumbnailSha,p.thumbnail)}
            val targetA=DirectMediaTarget("conv_Alice","player_Alice","Alice");val targetB=DirectMediaTarget("conv_Bob","player_Bob","Bob")
            var sender=BackendChatMedia(api,senderScope){_,_->}
            runBlocking{withContext(Dispatchers.Main){sender.enqueue(Uri.fromFile(photo).toString(),ChatReply(1,"Alice","引用","reply_private"),target=targetA)}}
            waitFor("lost private response retained"){sender.pending.firstOrNull()?.status=="failed"}
            val pending=sender.pending.single();val original=posts.single()
            val record=JSONObject(File(pending.directory,"pending.json").readText())
            check(record.getJSONObject("target").getString("conversationId")==targetA.conversationId)
            // A stale local file must not prevent recovering an already committed message.
            record.put("createdAt",System.currentTimeMillis()-4*86400000L)
            File(pending.directory,"pending.json").writeText(record.toString())
            senderScope.cancel();senderScope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
            val accepted=CopyOnWriteArrayList<Pair<JSONObject,DirectMediaTarget?>>()
            sender=BackendChatMedia(api,senderScope){message,target->accepted+=message to target}
            runBlocking{withContext(Dispatchers.Main){sender.start()}}
            waitFor("private recovery after restart"){sender.pending.isEmpty()&&accepted.size==1}
            check(posts.size==1&&accepted.single().second==targetA&&journal.size==1)
            check(JSONObject(original.second).getString("replyToMessageId")=="reply_private")
            runBlocking{withContext(Dispatchers.Main){sender.enqueue(null,null,moving,target=targetB)}}
            waitFor("Bob failed queue"){sender.pending.singleOrNull()?.status=="failed"}
            val expired=sender.pending.single();val expiredRecord=JSONObject(File(expired.directory,"pending.json").readText()).put("createdAt",System.currentTimeMillis()-4*86400000L)
            File(expired.directory,"pending.json").writeText(expiredRecord.toString())
            senderScope.cancel();senderScope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
            val beforeExpired=posts.size;sender=BackendChatMedia(api,senderScope){_,_->}
            runBlocking{withContext(Dispatchers.Main){sender.start()}}
            waitFor("expired unaccepted sticker refused"){sender.pending.singleOrNull()?.status=="failed"}
            check(posts.size==beforeExpired&&sender.pending.single().error!!.contains("超过 3 天"))
            runBlocking{withContext(Dispatchers.Main){sender.dismiss(sender.pending.single());sender.enqueue(null,null,moving,target=targetB)}}
            waitFor("fresh Bob failed queue"){sender.pending.singleOrNull()?.status=="failed"}
            val failedBob=sender.pending.single();val beforeRetry=posts.last()
            test.runOnMainSync{sender.retry(failedBob)};waitFor("Bob stable retry"){failedBob.status=="failed"&&posts.size==beforeExpired+2}
            check(posts.last()==beforeRetry)
            senderScope.cancel()
            result.putString("durable_send","PASS original recipient/request/reply after restart; expired accepted send recovered read-only; expired unaccepted sticker never sent; stable retry body")

            val state=LabState(scope,userName="PrivateQA",api=api)
            test.runOnMainSync{Players.removeAll{it.name in setOf("Alice","Bob")};Players.add(PlayerProfile("Alice",playerRef="player_Alice"));Players.add(PlayerProfile("Bob",playerRef="player_Bob"));state.chatDraft=TextFieldValue("public draft untouched")}
            runBlocking{withContext(Dispatchers.Main){state.directMediaTarget("Alice");state.directMediaTarget("Bob");state.chatMedia!!.start()}}
            waitFor("restored Bob failure"){state.chatMedia!!.pending.singleOrNull()?.status=="failed"}
            test.runOnMainSync{state.conversation("Alice")+=ChatLine(1,"Alice","[贴纸]",false,remoteId="incoming_gif",media=moving,serverAt=1)}
            val recipient=mutableStateOf("Alice")
            val activity=test.startActivitySync(Intent(context,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            test.waitForIdleSync();Thread.sleep(500)
            test.runOnMainSync{Players.removeAll{it.name in setOf("Alice","Bob")};Players.add(PlayerProfile("Alice",playerRef="player_Alice"));Players.add(PlayerProfile("Bob",playerRef="player_Bob"))}
            test.runOnMainSync{activity.setContent{LabTheme(theme,theme==1,true){
                val atmosphere=rememberGraphicsLayer();val backdrop=rememberGraphicsLayer()
                CompositionLocalProvider(LocalGlassMaterials provides GlassMaterials(),LocalGlassBackdrop provides atmosphere,LocalOverlayBackdrop provides backdrop,LocalOverlayGlassEnabled provides true){
                    Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)){GlassScene(atmosphere,backdrop,Modifier.fillMaxSize()){
                        Box(Modifier.fillMaxSize().windowInsetsPadding(WindowInsets.ime)){DirectChatPage(state,recipient.value,70.dp){}}
                    }}
                }
            }}}
            fun root():AccessibilityNodeInfo?{if(android.os.Build.VERSION.SDK_INT>=33)test.uiAutomation.clearCache();return test.uiAutomation.rootInActiveWindow}
            fun click(label:String,within:String?=null){
                waitFor("missing $label"){find(if(within==null)root() else find(root(),within),label)!=null}
                var node=find(if(within==null)root() else find(root(),within),label)!!
                while(!node.isClickable)node=node.parent
                check(node.performAction(AccessibilityNodeInfo.ACTION_CLICK));test.waitForIdleSync();Thread.sleep(280)
            }
            fun screenshot(name:String){Thread.sleep(400);test.uiAutomation.takeScreenshot()!!.let{frame->File(context.filesDir,"qa-private-$name-$theme.png").outputStream().use{frame.compress(Bitmap.CompressFormat.PNG,100,it)};frame.recycle()}}
            waitFor("private composer"){find(root(),"选择表情")!=null}
            check(find(root(),"私聊验收重试")==null){"Bob pending image visible in Alice"}
            click("选择表情");click("😀")
            check(editable(root())!!.text.toString()=="😀"&&state.chatDraft.text=="public draft untouched")
            click("发送消息");waitFor("emoji sent only to Alice"){state.conversation("Alice").any{it.text=="😀"}}
            val panel=find(root(),"表情面板")!!;val bounds=android.graphics.Rect();panel.getBoundsInScreen(bounds)
            test.uiAutomation.takeScreenshot()!!.let{frame->check(bounds.left==0&&bounds.right==frame.width&&bounds.bottom==frame.height);frame.recycle()}
            click("贴纸 1/10");click("查看贴纸","表情面板")
            waitFor("sticker sent to Alice"){state.conversation("Alice").any{it.mine&&it.media?.id==moving.id}}
            check(state.chat.isEmpty()&&state.chatDraft.text=="public draft untouched")
            screenshot("stickers")
            test.runOnMainSync{recipient.value="Bob"};waitFor("Bob pending image visible"){find(root(),"私聊验收重试")!=null}
            check(editable(root())!!.text.isNullOrBlank())
            test.runOnMainSync{recipient.value="Alice"};waitFor("Alice queue isolated"){find(root(),"私聊验收重试")==null&&find(root(),"选择表情")!=null}
            click("查看图片");waitFor("private image viewer"){find(root(),"关闭图片")!=null};screenshot("viewer");click("关闭图片")
            result.putString("private_ui","PASS Emoji/sticker sends, received media viewer, shared personal stickers, recipient-only queue, public draft isolation, edge-to-edge panel")

            galleryUri=context.contentResolver.insert(MediaStore.Images.Media.EXTERNAL_CONTENT_URI,ContentValues().apply{
                put(MediaStore.Images.Media.DISPLAY_NAME,"Deuterium_private_QA_$theme.png");put(MediaStore.Images.Media.MIME_TYPE,"image/png");put(MediaStore.Images.Media.RELATIVE_PATH,"Pictures/DeuteriumQA")
            })!!
            context.contentResolver.openOutputStream(galleryUri!!).use{out->photo.inputStream().use{it.copyTo(out!!)}}
            click("发送图片")
            // The platform picker labels thumbnails by date instead of by filename.
            waitFor("system photo picker"){findPhoto(root())!=null}
            var pick=findPhoto(root())!!;while(!pick.isClickable)pick=pick.parent
            check(pick.performAction(AccessibilityNodeInfo.ACTION_CLICK))
            waitFor("image confirmation"){find(root(),"展开预览")!=null};screenshot("attachment")
            val before=state.conversation("Alice").count{it.media?.id==picture.id}
            click("发送")
            waitFor("selected image sent to Alice"){state.conversation("Alice").count{it.media?.id==picture.id}>before}
            check(state.chat.isEmpty()&&state.chatDraft.text=="public draft untouched")
            rejectBob.set(false);test.runOnMainSync{state.chatMedia!!.retry(state.chatMedia.pending.single{it.target==targetB})}
            waitFor("Bob retry routed to Bob"){state.chatMedia!!.pending.isEmpty()&&state.conversation("Bob").any{it.media?.id==moving.id}}
            check(posts.all{it.first in setOf("conv_Alice","conv_Bob")})
            screenshot("complete")
            result.putString("photo_picker","PASS actual system picker → preview → private image; Bob retry still targets Bob")
            result.putString("status","PASS");test.finish(-1,result)
        }catch(e:Throwable){
            test.uiAutomation.takeScreenshot()?.let{frame->File(context.filesDir,"qa-private-failure-$theme.png").outputStream().use{frame.compress(Bitmap.CompressFormat.PNG,100,it)};frame.recycle()}
            fun dump(node:AccessibilityNodeInfo?):String=if(node==null)"" else "${node.text} | ${node.contentDescription} | ${node.viewIdResourceName}\n"+(0 until node.childCount).joinToString(""){dump(node.getChild(it))}
            File(context.filesDir,"qa-private-failure.txt").writeText(dump(test.uiAutomation.rootInActiveWindow))
            result.putString("error",e.stackTraceToString());test.finish(1,result)
        }
        finally{galleryUri?.let{context.contentResolver.delete(it,null,null)};scope.cancel();senderScope.cancel();server.shutdown()}
    }
    private fun waitFor(label:String,test:()->Boolean){val until=System.currentTimeMillis()+30000;while(!test()&&System.currentTimeMillis()<until)Thread.sleep(100);check(test()){label}}
    private fun find(node:AccessibilityNodeInfo?,label:String):AccessibilityNodeInfo?{if(node==null)return null;if(node.text?.toString()==label||node.contentDescription?.toString()==label)return node;repeat(node.childCount){find(node.getChild(it),label)?.let{return it}};return null}
    private fun editable(node:AccessibilityNodeInfo?):AccessibilityNodeInfo?{if(node==null)return null;if(node.isEditable)return node;repeat(node.childCount){editable(node.getChild(it))?.let{return it}};return null}
    private fun findPhoto(node:AccessibilityNodeInfo?):AccessibilityNodeInfo?{if(node==null)return null;if(node.viewIdResourceName=="com.google.android.providers.media.module:id/icon_thumbnail"||node.viewIdResourceName=="com.android.providers.media.module:id/icon_thumbnail")return node;repeat(node.childCount){findPhoto(node.getChild(it))?.let{return it}};return null}
}
