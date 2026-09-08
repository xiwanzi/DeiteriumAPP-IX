package com.deuterium.app.uilab

import android.content.Context
import android.graphics.Bitmap
import android.net.Uri
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.setValue
import coil3.ImageLoader
import coil3.decode.ImageSource
import coil3.disk.DiskCache
import coil3.fetch.Fetcher
import coil3.fetch.SourceFetchResult
import coil3.intercept.Interceptor
import coil3.key.Keyer
import coil3.memory.MemoryCache
import coil3.request.CachePolicy
import coil3.request.ErrorResult
import coil3.request.ImageRequest
import coil3.request.SuccessResult
import coil3.request.allowHardware
import coil3.request.crossfade
import coil3.toBitmap
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.OkHttpClient
import okio.Path.Companion.toOkioPath
import java.io.File
import java.io.IOException
import java.time.Instant
import java.util.concurrent.TimeUnit

internal data class RemoteImageData(val scope:FinancialScope,val source:String,val authorization:String?,val generation:Long)

/** One cache and decoder pipeline for every remote avatar, card and gallery in the app. */
class AppImages private constructor(private val context:Context) {
    companion object {
        const val DISK_LIMIT = 200L * 1024 * 1024
        const val MEMORY_LIMIT = 32L * 1024 * 1024
        @Volatile private var instance:AppImages?=null
        fun get(context:Context):AppImages=instance ?: synchronized(this){instance ?: AppImages(context.applicationContext).also{instance=it}}
        fun sessionEnded(){instance?.invalidate();RemoteImageUrls.clear()}
    }

    internal val store=ImageCacheStore(
        DiskCache.Builder().directory(File(context.cacheDir,"remote-images-v1").toOkioPath())
            .maxSizeBytes(minOf(DISK_LIMIT,(context.cacheDir.usableSpace/20).coerceAtLeast(1024*1024))).build(),
        OkHttpClient.Builder().connectTimeout(12,TimeUnit.SECONDS).readTimeout(25,TimeUnit.SECONDS)
            .callTimeout(60,TimeUnit.SECONDS).followRedirects(false).retryOnConnectionFailure(false).build(),
    )
    var epoch by mutableLongStateOf(0);private set
    val loader:ImageLoader=ImageLoader.Builder(context)
        .memoryCache{MemoryCache.Builder().maxSizeBytes(minOf(MEMORY_LIMIT,Runtime.getRuntime().maxMemory()/10)).build()}
        .diskCache(store.disk)
        .crossfade(false)
        .components{
            add(Interceptor{chain->
                val data=chain.request.data as? RemoteImageData
                if(data!=null){
                    checkRequest(data)
                    val access=RemoteImageUrls.access(data.scope,data.source)
                    if(access?.expired==true||access?.retainUntil?.isAfter(Instant.now())==false)
                        throw ApiFailure("IMAGE_EXPIRED","图片已过期")
                }
                chain.proceed()
            })
            add(Keyer<RemoteImageData>{data,_->"${ImageCacheStore.key(data.scope,data.source)}:${data.generation}"})
            add(Fetcher.Factory<RemoteImageData>{data,_,_->Fetcher{
                checkRequest(data)
                val cached=store.open(ImageCacheStore.key(data.scope,data.source),RemoteImageUrls.access(data.scope,data.source),
                    {validRequest(data)},{renew->resolve(data,renew)})
                SourceFetchResult(ImageSource(cached.file,store.disk.fileSystem,ImageCacheStore.key(data.scope,data.source),cached),cached.mimeType,cached.dataSource)
            }})
        }.build()

    private fun validRequest(data:RemoteImageData)=data.generation==store.generation.get()&&BackendApi.get(context).financialScope()==data.scope
    private fun checkRequest(data:RemoteImageData){if(!validRequest(data))throw CancellationException("Image account changed")}

    fun request(source:String):ImageRequest {
        val api=BackendApi.get(context)
        val remote=source.startsWith("asset:")||source.startsWith("https://")
        val data=if(remote)RemoteImageData(api.financialScope(),source,RemoteImageUrls.key(api.financialScope(),source),store.generation.get()) else Uri.parse(source)
        val privateEvidence=remote&&RemoteImageUrls.access(api.financialScope(),source)?.privateEvidence==true
        return ImageRequest.Builder(context).data(data).crossfade(false)
            .diskCachePolicy(if(privateEvidence)CachePolicy.DISABLED else CachePolicy.ENABLED)
            .memoryCachePolicy(if(privateEvidence)CachePolicy.DISABLED else CachePolicy.ENABLED).build()
    }

    private suspend fun resolve(data:RemoteImageData,renew:Boolean):ImageAccess {
        checkRequest(data)
        if(data.source.startsWith("https://")){
            require(RemoteImageUrls.safe(data.source)){"图片地址无效"}
            return ImageAccess(data.source)
        }
        val api=BackendApi.get(context)
        if(!renew){
            try {
                RemoteImageUrls.resolve(data.scope,data.source)?.let{return RemoteImageUrls.access(data.scope,data.source) ?: ImageAccess(it)}
            }catch(failure:ApiFailure){if(failure.code!="IMAGE_URL_EXPIRED")throw failure}
        }
        val path=RemoteImageUrls.refreshPath(data.scope,data.source) ?: "/assets/${data.source.removePrefix("asset:")}"
        api.request("GET",path)
        checkRequest(data)
        val url=RemoteImageUrls.resolve(data.scope,data.source) ?: throw ApiFailure("IMAGE_UNAVAILABLE","图片暂时无法加载")
        require(RemoteImageUrls.safe(url)){"图片地址无效"}
        return RemoteImageUrls.access(data.scope,data.source) ?: ImageAccess(url)
    }

    suspend fun bitmap(source:String,maxSide:Int):Bitmap {
        val result=loader.execute(request(source).newBuilder().size(maxSide,maxSide).allowHardware(false).build())
        if(result is ErrorResult)throw result.throwable
        return (result as SuccessResult).image.toBitmap()
    }

    suspend fun seed(source:String,bytes:ByteArray){
        val scope=BackendApi.get(context).financialScope()
        try{store.seed(ImageCacheStore.key(scope,source),bytes,RemoteImageUrls.access(scope,source))}
        catch(cancelled:CancellationException){throw cancelled}
        catch(_:IOException){ /* A successful upload is still successful if local storage is full. */ }
    }

    private fun invalidate(){store.invalidateRequests();loader.memoryCache?.clear();epoch=store.generation.get()}
    suspend fun clear(){
        invalidate()
        store.clear()
        epoch=store.generation.get()
        // A response completing just before invalidation can only populate its old generation.
        loader.memoryCache?.clear()
    }
    suspend fun diskBytes():Long=withContext(Dispatchers.IO){store.disk.size}
    val memoryBytes:Long get()=loader.memoryCache?.size ?: 0
    val diskBudget:Long get()=store.disk.maxSize
}
