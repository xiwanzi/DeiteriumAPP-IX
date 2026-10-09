package com.deuterium.app.uilab

import android.graphics.BitmapFactory
import android.net.Uri
import android.util.AtomicFile
import android.util.Base64
import androidx.compose.runtime.*
import kotlinx.coroutines.*
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.sync.withPermit
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.io.File
import java.io.IOException
import java.security.MessageDigest
import java.time.Instant
import java.util.UUID
import java.util.concurrent.TimeUnit

data class ChatMedia(val id:String,val sha:String,val thumbnailSha:String,val bytes:Long,val thumbnailBytes:Long,val width:Int,val height:Int,val animated:Boolean,val type:String="image",val expiresAt:String?=null){
    fun json()=JSONObject().put("mediaId",id).put("sha256",sha).put("thumbnailSha256",thumbnailSha).put("sizeBytes",bytes).put("thumbnailBytes",thumbnailBytes).put("width",width).put("height",height).put("animated",animated).put("type",type).apply{expiresAt?.let{put("expiresAt",it)}}
    companion object {fun parse(v:JSONObject)=ChatMedia(v.getString("mediaId"),v.getString("sha256"),v.getString("thumbnailSha256"),v.getLong("sizeBytes"),v.getLong("thumbnailBytes"),v.getInt("width"),v.getInt("height"),v.optBoolean("animated"),v.optString("type","image"),v.optString("expiresAt").takeIf{it.isNotBlank()&&it!="null"})}
}

internal class PendingChatMedia(val id:String,val directory:File,val createdAt:Long,val importing:Boolean,val reply:ChatReply?,initialMedia:ChatMedia?=null){
    var media by mutableStateOf(initialMedia)
    var status by mutableStateOf("queued")
    var error by mutableStateOf<String?>(null)
    var width by mutableIntStateOf(initialMedia?.width ?: 240)
    var height by mutableIntStateOf(initialMedia?.height ?: 180)
    var animated by mutableStateOf(initialMedia?.animated ?: false)
    var prepared by mutableStateOf(false)
    var attempted=false
    val source get()=File(directory,"source")
    val image get()=File(directory,"image.webp")
    val thumbnail get()=File(directory,"preview.webp")
    val localFile get()=if(prepared)image else source
    val type get()=if(importing||media?.type=="sticker")"sticker" else "image"
    fun json()=JSONObject().put("id",id).put("createdAt",createdAt).put("importing",importing).put("prepared",prepared).put("attempted",attempted).put("width",width).put("height",height).put("animated",animated).apply{
        media?.let{put("media",it.json())};reply?.let{put("reply",JSONObject().put("id",it.messageId).put("name",it.name).put("text",it.text).put("remoteId",it.remoteId))}
    }
}

internal class BackendChatMedia(private val api:BackendApi,private val scope:CoroutineScope,objectClient:OkHttpClient?=null,private val accepted:(JSONObject)->Unit){
    private val identity=api.financialScope()
    val cache=ChatCache(api.context,identity)
    private val outbox=File(api.context.filesDir,"chat-outbox/${ChatCache.hash("${identity.origin}:${identity.owner}".toByteArray())}")
    val pending=mutableStateListOf<PendingChatMedia>()
    val stickers=mutableStateListOf<ChatMedia>()
    var stickerError by mutableStateOf<String?>(null);private set
    var stickersLoaded by mutableStateOf(false);private set
    var cacheEpoch by mutableLongStateOf(ChatCache.epoch);private set
    private val enqueueLock=Mutex();private val sending=Mutex();private val downloads=Semaphore(3)
    private val signing=Mutex();private val signed=linkedMapOf<String,JSONObject>()
    private var started=false
    private val fileLocks=mutableMapOf<String,Mutex>()
    private val objects=objectClient ?: OkHttpClient.Builder().connectTimeout(12,TimeUnit.SECONDS).readTimeout(40,TimeUnit.SECONDS).callTimeout(60,TimeUnit.SECONDS).followRedirects(false).retryOnConnectionFailure(false).build()
    private fun current(){identity.verifyCurrent(api.financialScope())}
    suspend fun start(){
        if(started)return;started=true
        val jobs=withContext(Dispatchers.IO){outbox.listFiles().orEmpty().filter{it.isDirectory&&it.name.matches(Regex("[a-f0-9-]{36}"))}.mapNotNull{folder->runCatching{
            val value=JSONObject(AtomicFile(File(folder,"pending.json")).readFully().toString(Charsets.UTF_8));val reply=value.optJSONObject("reply")?.let{ChatReply(it.getLong("id"),it.getString("name"),it.getString("text"),it.getString("remoteId"))}
            PendingChatMedia(value.getString("id"),folder,value.getLong("createdAt"),value.optBoolean("importing"),reply,value.optJSONObject("media")?.let(ChatMedia::parse)).apply{require(id==folder.name);prepared=value.optBoolean("prepared");attempted=value.optBoolean("attempted");width=value.optInt("width",240);height=value.optInt("height",180);animated=value.optBoolean("animated")}
        }.getOrNull()}.sortedBy{it.createdAt}}
        current();pending.addAll(jobs);scope.launch{ChatCache.cleared.collect{cacheEpoch=ChatCache.epoch}}
        scope.launch{while(isActive){expireUnsentFiles();delay(3600000)}}
        drain();refreshStickers()
    }
    private suspend fun expireUnsentFiles(){
        val cutoff=System.currentTimeMillis()-3*86400000L
        pending.filter{it.createdAt<=cutoff&&it.status!="sending"}.forEach{job->
            withContext(Dispatchers.IO){job.source.delete();job.image.delete();job.thumbnail.delete()}
            // Preserve a tiny request journal until an unknown send is reconciled.
            if(!job.attempted){job.status="failed";job.error="待发送图片已超过 3 天，请重新选择"}
        }
        val known=pending.map{it.directory.name}.toSet()
        withContext(Dispatchers.IO){outbox.listFiles().orEmpty().filter{it.isDirectory&&it.name.matches(Regex("[a-f0-9-]{36}"))&&it.name !in known&&it.lastModified()<=cutoff}.forEach{StorageFiles.clearChildren(it);it.delete()}}
    }
    suspend fun refreshStickers(){
        try{current();val r=api.request("GET","/chat/stickers").getJSONArray("stickers");current();stickers.clear();repeat(r.length()){stickers+=ChatMedia.parse(r.getJSONObject(it)).copy(type="sticker")};stickerError=null;stickersLoaded=true}
        catch(e:CancellationException){throw e}catch(e:Exception){stickerError=e.message ?: "贴纸暂时无法同步"}
    }
    private suspend fun save(job:PendingChatMedia){val json=job.json().toString();withContext(Dispatchers.IO){current();job.directory.mkdirs();val atomic=AtomicFile(File(job.directory,"pending.json"));val stream=atomic.startWrite();try{stream.write(json.toByteArray());atomic.finishWrite(stream)}catch(e:Exception){atomic.failWrite(stream);throw e}}}
    suspend fun enqueue(uri:String?,reply:ChatReply?,sticker:ChatMedia?=null,importing:Boolean=false){
        scope.async{enqueueInScope(uri,reply,sticker,importing)}.await()
    }
    private suspend fun enqueueInScope(uri:String?,reply:ChatReply?,sticker:ChatMedia?,importing:Boolean)=enqueueLock.withLock{
        current();require(pending.size<10){"待发送图片较多，请稍后再试"};require(!importing||stickers.size<10){"贴纸已满（10/10），删除已有贴纸后可继续添加"}
        val id=UUID.randomUUID().toString();val job=PendingChatMedia(id,File(outbox,id),System.currentTimeMillis(),importing,reply,sticker)
        try{if(uri!=null)withContext(Dispatchers.IO){
            job.directory.mkdirs();api.context.contentResolver.openInputStream(Uri.parse(uri))?.use{input->job.source.outputStream().use{output->val b=ByteArray(8192);var count=0L;while(true){val n=input.read(b);if(n<0)break;count+=n;require(count<=20L*1024*1024){"请选择 20MB 以内的图片"};output.write(b,0,n)}}} ?: error("无法读取图片")
            val bounds=BitmapFactory.Options().apply{inJustDecodeBounds=true};BitmapFactory.decodeFile(job.source.path,bounds)
            if(bounds.outWidth>0){
                val orientation=runCatching{android.media.ExifInterface(job.source.path).getAttributeInt(android.media.ExifInterface.TAG_ORIENTATION,1)}.getOrDefault(1)
                job.width=if(orientation in 5..8)bounds.outHeight else bounds.outWidth;job.height=if(orientation in 5..8)bounds.outWidth else bounds.outHeight
                job.animated=bounds.outMimeType=="image/gif"
            }
        }
        save(job);pending+=job;drain()}catch(e:Exception){withContext(NonCancellable+Dispatchers.IO){StorageFiles.clearChildren(job.directory);job.directory.delete()};throw e}
    }
    fun retry(job:PendingChatMedia){if(job.status=="sending")return;job.status="queued";job.error=null;drain()}
    suspend fun dismiss(job:PendingChatMedia){if(job.status=="sending")return;withContext(Dispatchers.IO){StorageFiles.clearChildren(job.directory);job.directory.delete()};pending.remove(job)}
    private fun drain(){scope.launch{sending.withLock{while(true){val job=pending.firstOrNull{it.status=="queued"} ?: break;job.status="sending"
        try{
            current();var result:JSONObject?=null
            if(job.attempted&&!job.importing){val known=api.request("GET","/chat/sends/${job.id}");if(known.optString("status")=="accepted")result=known}
            if(result==null){
                if(System.currentTimeMillis()-job.createdAt>3*86400000L){withContext(Dispatchers.IO){job.source.delete();job.image.delete();job.thumbnail.delete()};throw IOException("待发送图片已超过 3 天，请重新选择")}
                if(job.media==null){
                    if(!job.prepared){val prepared=ChatImageCompressor.prepare(job.source,job.importing);withContext(Dispatchers.IO){job.image.writeBytes(prepared.image);job.thumbnail.writeBytes(prepared.thumbnail)};job.width=prepared.width;job.height=prepared.height;job.animated=prepared.animated;job.prepared=true;save(job);withContext(Dispatchers.IO){job.source.delete()}}
                    upload(job)
                }
                if(job.importing){api.request("POST","/chat/stickers",JSONObject().put("mediaId",job.media!!.id));refreshStickers()}
                else{
                    job.attempted=true;save(job)
                    result=api.request("POST","/chat/messages",JSONObject().put("clientMessageId",job.id).put("content",if(job.type=="sticker")"[贴纸]" else "[图片]").put("media",JSONObject().put("mediaId",job.media!!.id).put("type",job.type)).apply{job.reply?.remoteId?.takeIf{it.isNotBlank()}?.let{put("replyToMessageId",it)}})
                    if(result.optString("status")!="accepted"){val error=result.optJSONObject("error");throw IOException(if(error?.optString("code")=="RESULT_UNKNOWN")"发送结果待确认，点击重试" else error?.optString("message") ?: "发送结果暂未确认，请重试")}
                }
            }
            current()
            // Only committed messages/collections enter the non-expiring cache.
            if(job.prepared)try{withContext(Dispatchers.IO){if(job.image.isFile&&job.thumbnail.isFile){val m=job.media!!;cache.putFile(m.sha,job.image.readBytes());cache.putFile(m.thumbnailSha,job.thumbnail.readBytes())}}}catch(e:CancellationException){throw e}catch(_:Exception){}
            result?.optJSONObject("message")?.let(accepted);withContext(Dispatchers.IO){StorageFiles.clearChildren(job.directory);job.directory.delete()};pending.remove(job)
        }catch(e:CancellationException){job.status="queued";throw e}catch(e:Exception){job.status="failed";job.error=when((e as? ApiFailure)?.code){"STORAGE_UNAVAILABLE","INVALID_MEDIA"->"图片未能发送，请稍后重试或重新选择";else->e.message ?: "图片发送失败，请重试"};if(job.importing)stickerError=job.error}
    }}}}
    private suspend fun put(auth:JSONObject,bytes:ByteArray,md5:String)=withContext(Dispatchers.IO){
        current();val url=java.net.URI(auth.getString("uploadUrl"));val headers=auth.getJSONObject("signedHeaders");val lower=headers.keys().asSequence().associate{it.lowercase() to headers.getString(it)}
        require(url.scheme=="https"&&url.rawUserInfo==null&&url.host.endsWith(".rains3.com")&&auth.getString("method")=="PUT"&&auth.getLong("sizeBytes")==bytes.size.toLong()&&lower["content-type"]=="image/webp"&&lower["content-md5"]==md5&&lower["if-none-match"]=="*"&&lower.keys.all{it in setOf("content-type","content-md5","if-none-match")}){"图片发送暂不可用"}
        val request=Request.Builder().url(url.toString()).put(bytes.toRequestBody("image/webp".toMediaType()));headers.keys().forEach{request.header(it,headers.getString(it))}
        objects.newCall(request.build()).execute().use{response->if(!response.isSuccessful&&response.code!=412)throw IOException("图片发送未完成，请重试")}
    }
    private suspend fun upload(job:PendingChatMedia){
        val (main,thumb)=withContext(Dispatchers.IO){job.image.readBytes() to job.thumbnail.readBytes()}
        val sha=withContext(Dispatchers.Default){ChatCache.hash(main)};val thumbSha=withContext(Dispatchers.Default){ChatCache.hash(thumb)}
        fun md5(bytes:ByteArray)=Base64.encodeToString(MessageDigest.getInstance("MD5").digest(bytes),Base64.NO_WRAP)
        val md5=md5(main);val thumbnailMd5=md5(thumb)
        var session=api.request("POST","/chat/media/uploads",JSONObject().put("clientRequestId",job.id).put("sha256",sha).put("thumbnailSha256",thumbSha).put("contentMd5",md5).put("thumbnailMd5",thumbnailMd5).put("sizeBytes",main.size).put("thumbnailBytes",thumb.size))
        if(session.optString("status")!="READY"){
            try{put(session.getJSONObject("authorization"),main,md5)}catch(e:CancellationException){throw e}catch(_:IOException){}
            try{put(session.getJSONObject("thumbnailAuthorization"),thumb,thumbnailMd5)}catch(e:CancellationException){throw e}catch(_:IOException){}
            for(attempt in 0..3){try{session=api.request("POST","/chat/media/uploads/${session.getString("uploadId")}/complete",JSONObject());break}catch(e:ApiFailure){if(e.code!="MEDIA_BUSY"||attempt==3)throw e;delay(500L*(attempt+1))}}
        }
        require(session.optString("status")=="READY"){"图片发送未完成，请重试"};current();job.media=ChatMedia.parse(session.getJSONObject("media")).copy(type=job.type);save(job)
    }
    suspend fun file(media:ChatMedia,messageId:String?=null,thumbnail:Boolean=false):File {
        current();val hash=if(thumbnail)media.thumbnailSha else media.sha
        withContext(Dispatchers.IO){cache.file(hash)}?.let{return it}
        val mutex=synchronized(fileLocks){fileLocks.getOrPut(hash){Mutex()}}
        return mutex.withLock{withContext(Dispatchers.IO){cache.file(hash)} ?: downloads.withPermit{
            val epoch=ChatCache.epoch
            val query=messageId?.takeIf{it.isNotBlank()}?.let{"?messageId=${java.net.URLEncoder.encode(it,"UTF-8")}"} ?: ""
            val access=signing.withLock{
                val key="${media.id}:$query";val old=signed[key]
                if(old!=null&&runCatching{Instant.parse(old.getString("urlExpiresAt")).isAfter(Instant.now().plusSeconds(20))}.getOrDefault(false))old
                else api.request("GET","/chat/media/files/${media.id}/access$query").also{if(signed.size>=256)signed.remove(signed.keys.first());signed[key]=it}
            }
            val link=java.net.URI(access.getString(if(thumbnail)"thumbnailUrl" else "url"));require(link.scheme=="https"&&link.rawUserInfo==null&&link.host.endsWith(".rains3.com")){"图片暂不可用"}
            val size=if(thumbnail)media.thumbnailBytes else media.bytes;require(size in 1..2L*1024*1024)
            withContext(Dispatchers.IO){current();objects.newCall(Request.Builder().url(link.toString()).get().build()).execute().use{response->
                if(!response.isSuccessful)throw IOException("图片暂不可用，请重试")
                val bytes=response.body?.byteStream()?.use{input->val out=java.io.ByteArrayOutputStream();val buffer=ByteArray(8192);while(true){val n=input.read(buffer);if(n<0)break;require(out.size()+n<=size){"图片不完整"};out.write(buffer,0,n)};out.toByteArray()} ?: error("图片不完整");require(bytes.size.toLong()==size){"图片不完整"};current();cache.putFile(hash,bytes,epoch)
            }}
        }}
    }
    suspend fun deleteSticker(media:ChatMedia){current();api.request("DELETE","/chat/stickers/${media.id}");refreshStickers()}
    suspend fun collect(media:ChatMedia,messageId:String){
        if(media.width<=512&&media.height<=512&&(media.animated||media.bytes<=512*1024)){
            try{api.request("POST","/chat/stickers",JSONObject().put("mediaId",media.id));refreshStickers();return}catch(e:ApiFailure){if(e.code !in setOf("MEDIA_EXPIRED","MEDIA_NOT_FOUND"))throw e}
        }
        val file=file(media,messageId);enqueue(Uri.fromFile(file).toString(),null,importing=true)
    }
}
