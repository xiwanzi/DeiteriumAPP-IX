package com.deuterium.app.uilab

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.net.Uri
import android.util.Base64
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.io.ByteArrayOutputStream
import java.io.IOException
import java.security.MessageDigest
import java.time.Instant
import java.util.UUID
import java.util.concurrent.TimeUnit

/** Signed object requests use a separate client with no application credentials. */
object BackendAssets {
    private const val LIMIT=20*1024*1024
    private val objectHttp=OkHttpClient.Builder().connectTimeout(15,TimeUnit.SECONDS).readTimeout(45,TimeUnit.SECONDS)
        .callTimeout(90,TimeUnit.SECONDS).retryOnConnectionFailure(false).followRedirects(false).build()
    private fun bytes(context:Context,uri:Uri):ByteArray=context.contentResolver.openInputStream(uri)?.use{input->
        val out=ByteArrayOutputStream();val buffer=ByteArray(8192)
        while(true){val count=input.read(buffer);if(count<0)break;require(out.size()+count<=LIMIT){"单张图片不能超过 20 MB"};out.write(buffer,0,count)}
        out.toByteArray()
    } ?: error("无法读取图片")
    private fun digest(value:ByteArray,algorithm:String)=MessageDigest.getInstance(algorithm).digest(value)
    suspend fun upload(context:Context,uri:String,purpose:String,businessType:String,businessRef:String?=null):String {
        if(uri.startsWith("asset:"))return uri
        val api=BackendApi.get(context)
        val data=withContext(Dispatchers.IO){bytes(context,Uri.parse(uri))}
        val bounds=BitmapFactory.Options().apply{inJustDecodeBounds=true};BitmapFactory.decodeByteArray(data,0,data.size,bounds)
        val mime=bounds.outMimeType;require(bounds.outWidth>0&&bounds.outHeight>0&&mime in setOf("image/png","image/jpeg","image/webp")){"请选择 PNG、JPEG 或 WebP 图片"}
        val md5=Base64.encodeToString(digest(data,"MD5"),Base64.NO_WRAP)
        val cacheKey=digest("$purpose:$businessType:${businessRef.orEmpty()}:$md5".toByteArray(),"SHA-256").joinToString(""){"%02x".format(it)}
        val pending=api.uploadState(cacheKey) ?: JSONObject().put("createKey",UUID.randomUUID().toString()).put("completeKey",UUID.randomUUID().toString()).also{api.saveUploadState(cacheKey,it)}
        var session=if(pending.has("uploadId"))api.request("GET","/assets/uploads/${pending.getString("uploadId")}") else {
            val input=JSONObject().put("clientRequestId",pending.getString("createKey")).put("purpose",purpose).put("businessType",businessType)
                .put("fileName","image.${if(mime=="image/jpeg")"jpg" else mime.substringAfter('/')}").put("contentType",mime).put("sizeBytes",data.size).put("contentMd5",md5).put("altText",if(purpose=="AVATAR")"玩家头像" else "业务图片")
            if(!businessRef.isNullOrBlank())input.put("businessRef",businessRef)
            api.request("POST","/assets/uploads",input).also{pending.put("uploadId",it.getString("uploadId"));api.saveUploadState(cacheKey,pending)}
        }
        val uploadId=session.getString("uploadId")
        suspend fun ready(value:JSONObject):String?=if(value.optString("status")=="READY"){
            val source="asset:${value.getString("assetId")}"
            AppImages.get(context).seed(source,data)
            source
        } else null
        ready(session)?.let{return it}
        if(session.optString("status") in listOf("REJECTED","EXPIRED")){api.clearUploadState(cacheKey);error("图片上传已失效，请重试或重新选择图片")}
        val completeBody=JSONObject().put("clientRequestId",pending.getString("completeKey"))
        if(pending.optBoolean("attempted")) {
            runCatching{api.request("POST","/assets/uploads/$uploadId/complete",completeBody)}.getOrNull()?.let{session=it;ready(it)?.let{asset->return asset}}
        }
        if(session.optString("status")=="AUTHORIZED") {
            var authorization=session.getJSONObject("authorization")
            if(Instant.parse(authorization.getString("expiresAt")).isBefore(Instant.now().plusSeconds(10))) {
                session=api.request("POST","/assets/uploads/$uploadId/renew",JSONObject().put("clientRequestId",UUID.randomUUID().toString()))
                authorization=session.getJSONObject("authorization")
            }
            require(authorization.getString("provider")=="RAINYUN_S3"&&authorization.getString("method")=="PUT"){"服务器返回了不支持的图片上传方式"}
            require(authorization.getLong("sizeBytes")==data.size.toLong()){"上传授权与实际文件大小不一致"}
            val url=authorization.getString("uploadUrl");val parsed=java.net.URI(url)
            require(parsed.scheme=="https"&&parsed.rawUserInfo==null&&parsed.host.endsWith(".rains3.com")){"图片上传地址无效"}
            val headers=authorization.getJSONObject("signedHeaders")
            val headerValues=headers.keys().asSequence().associate{it.lowercase() to headers.getString(it)}
            require(headerValues["if-none-match"]=="*"&&headerValues["content-md5"]==md5&&headerValues["content-type"]==mime){"图片上传授权校验失败"}
            val builder=Request.Builder().url(url).put(data.toRequestBody(mime.toMediaType()))
            headers.keys().forEach{header->require(header.lowercase() in setOf("content-type","content-md5","if-none-match")){"上传授权包含不支持的请求头"};builder.header(header,headers.getString(header))}
            pending.put("attempted",true);api.saveUploadState(cacheKey,pending)
            try { withContext(Dispatchers.IO){objectHttp.newCall(builder.build()).execute().use{response->if(!response.isSuccessful&&response.code!=412)throw ApiFailure("UPLOAD_FAILED","图片上传未完成（${response.code}）")}} }
            catch(error:IOException){ /* Verify the original upload before deciding whether it failed. */ }
            session=api.request("POST","/assets/uploads/$uploadId/complete",completeBody)
        }
        repeat(20){
            ready(session)?.let{return it}
            if(session.optString("status") in listOf("REJECTED","EXPIRED"))api.clearUploadState(cacheKey)
            require(session.optString("status")=="VERIFYING"){"图片未通过服务器校验，请重新选择"}
            delay(session.optLong("retryAfterSeconds",2).coerceIn(1,5)*1000)
            session=api.request("GET","/assets/uploads/$uploadId")
        }
        error("图片校验仍在进行，请稍后重试原图片")
    }
    fun imageKey(context:Context,source:String?):String?=RemoteImageUrls.key(BackendApi.get(context).financialScope(),source)
    suspend fun bitmap(context:Context,source:String,maxSide:Int=1000):Bitmap=AppImages.get(context).bitmap(source,maxSide)
}
