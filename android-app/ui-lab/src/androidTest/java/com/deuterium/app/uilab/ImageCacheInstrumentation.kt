package com.deuterium.app.uilab

import android.app.Instrumentation
import android.graphics.Bitmap
import android.os.Bundle
import coil3.decode.DataSource
import coil3.request.CachePolicy
import coil3.request.ErrorResult
import coil3.request.SuccessResult
import coil3.request.allowHardware
import coil3.toBitmap
import kotlinx.coroutines.runBlocking
import org.json.JSONObject
import java.io.ByteArrayOutputStream
import java.time.Instant

object ImageCacheInstrumentation {
    fun run(test:Instrumentation){
        val result=Bundle()
        try{runBlocking{
            val context=test.targetContext
            val images=AppImages.get(context)
            val scope=BackendApi.get(context).financialScope()
            val id="instrumented_"+java.util.UUID.randomUUID().toString()
            val source="asset:$id"
            val bitmap=Bitmap.createBitmap(512,512,Bitmap.Config.ARGB_8888).apply{eraseColor(0xff357ed3.toInt())}
            val bytes=ByteArrayOutputStream().also{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)}.toByteArray();bitmap.recycle()
            fun view(signature:String)=JSONObject().put("assetId",id).put("status","READY").put("url","https://images.invalid/photo?signature=$signature")
                .put("sha256",ImageCacheStore.hash(bytes)).put("urlExpiresAt",Instant.now().plusSeconds(600).toString()).put("retainUntil",JSONObject.NULL)
            RemoteImageUrls.remember(view("one"),scope)
            images.seed(source,bytes)
            fun request()=images.request(source).newBuilder().size(96,96).allowHardware(false).build()
            val first=images.loader.execute(request())
            check(first is SuccessResult&&first.dataSource==DataSource.DISK){"First read did not use seeded disk bytes: $first"}
            check(first.image.toBitmap().width<=96&&first.image.toBitmap().height<=96){"Avatar decoded larger than its target"}
            val second=images.loader.execute(request())
            check(second is SuccessResult&&second.dataSource==DataSource.MEMORY_CACHE){"Second read missed shared memory cache"}
            RemoteImageUrls.remember(view("two"),scope)
            val rotated=images.loader.execute(request())
            check(rotated is SuccessResult&&rotated.dataSource==DataSource.MEMORY_CACHE){"Signature rotation invalidated image contents"}
            check(images.memoryBytes<=AppImages.MEMORY_LIMIT)
            RemoteImageUrls.remember(view("two").put("retainUntil",Instant.now().minusSeconds(1).toString()),scope)
            val expired=images.loader.execute(request())
            check(expired is ErrorResult&&(expired.throwable as? ApiFailure)?.code=="IMAGE_EXPIRED"){"Expired object was resurrected from memory"}
            RemoteImageUrls.remember(view("private").put("purpose","DISPUTE_EVIDENCE"),scope)
            val sensitive=images.request(source)
            check(sensitive.memoryCachePolicy==CachePolicy.DISABLED&&sensitive.diskCachePolicy==CachePolicy.DISABLED)
            images.clear()
            check(images.diskBytes()==0L&&images.memoryBytes==0L){"Clear did not release cached image data"}
            result.putString("disk_memory_signature_reuse","PASS")
            result.putString("target_size_and_retention","PASS")
            result.putString("private_evidence_and_clear","PASS")
        };test.finish(-1,result)}catch(error:Throwable){result.putString("error",error.stackTraceToString());test.finish(1,result)}
    }
}
