package com.deuterium.app.uilab

import org.json.JSONArray
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import java.time.Instant
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit

class RemoteImageUrlsTest {
    private val scope=FinancialScope("image-viewer","https://server.example")
    private fun view(id:String,url:String,expiry:String="2099-01-01T00:00:00Z")=JSONObject().put("assetId",id).put("status","READY").put("url",url).put("urlExpiresAt",expiry)
    @Test fun embeddedPhotosAndAvatarKeepAuthorizedUrlsForAssetIds() {
        RemoteImageUrls.remember(JSONObject().put("photos",JSONArray().put(view("photo","https://s3.example/photo?signature=one")))
            .put("seller",JSONObject().put("avatar",view("avatar","https://s3.example/avatar?signature=two"))),scope)
        assertEquals("https://s3.example/photo?signature=one",RemoteImageUrls.resolve(scope,"asset:photo"))
        assertEquals("https://s3.example/avatar?signature=two",RemoteImageUrls.resolve(scope,"asset:avatar"))
        assertNull(RemoteImageUrls.resolve(FinancialScope("other",scope.origin),"asset:photo"))
        assertNull(RemoteImageUrls.resolve(FinancialScope(scope.owner,"https://other.example"),"asset:photo"))
    }
    @Test fun expiredViewRequiresBusinessRefreshAndNewUrlInvalidatesRenderingKey() {
        RemoteImageUrls.remember(view("expired","https://s3.example/old","2020-01-01T00:00:00Z"),scope)
        assertEquals("IMAGE_URL_EXPIRED",assertThrows(ApiFailure::class.java){RemoteImageUrls.resolve(scope,"asset:expired",Instant.parse("2026-01-01T00:00:00Z"))}.code)
        RemoteImageUrls.remember(view("expired","https://s3.example/new"),scope)
        assertEquals("https://s3.example/new",RemoteImageUrls.key(scope,"asset:expired"))
        assertEquals("https://s3.example/new",RemoteImageUrls.resolve(scope,"asset:expired"))
    }
    @Test fun unsafeAndUnreadyImageLinksAreNotAccepted() {
        for(url in listOf("http://s3.example/a","https://user:password@s3.example/a","file:///a"))assertFalse(RemoteImageUrls.safe(url))
        RemoteImageUrls.remember(view("unready","https://s3.example/a").put("status","VERIFYING"),scope)
        assertNull(RemoteImageUrls.resolve(scope,"asset:unready"))
    }

    @Test fun oneResponseKeepsSequentialRetentionAndRefreshPathInheritance() {
        RemoteImageUrls.clear()
        val expiry="2099-02-01T00:00:00Z"
        RemoteImageUrls.remember(JSONArray()
            .put(view("same","https://s3.example/one").put("retainUntil",expiry))
            .put(view("same","https://s3.example/two")),scope,refreshPath="/orders/original")
        assertEquals(Instant.parse(expiry),RemoteImageUrls.access(scope,"asset:same")!!.retainUntil)
        assertEquals("https://s3.example/two",RemoteImageUrls.resolve(scope,"asset:same"))
        RemoteImageUrls.remember(view("same","https://s3.example/three"),scope)
        assertEquals("/orders/original",RemoteImageUrls.refreshPath(scope,"asset:same"))
        RemoteImageUrls.remember(view("same","https://s3.example/four").put("retainUntil",JSONObject.NULL),scope)
        assertNull(RemoteImageUrls.access(scope,"asset:same")!!.retainUntil)
    }

    @Test fun imageReadsDoNotWaitForResponseTraversalAndClearCannotResurrectOldEntries() {
        RemoteImageUrls.clear();RemoteImageUrls.remember(view("visible","https://s3.example/old"),scope)
        val entered=CountDownLatch(1);val resume=CountDownLatch(1)
        val slow=object:JSONObject(){override fun keys():MutableIterator<String>{entered.countDown();check(resume.await(5,TimeUnit.SECONDS));return super.keys()}}
            .put("photos",JSONArray().put(view("visible","https://s3.example/stale")))
        val workers=Executors.newFixedThreadPool(2)
        try {
            val parsing=workers.submit{RemoteImageUrls.remember(slow,scope)}
            assertTrue(entered.await(5,TimeUnit.SECONDS))
            val read=workers.submit<String?>{RemoteImageUrls.resolve(scope,"asset:visible")}
            assertEquals("https://s3.example/old",read.get(1,TimeUnit.SECONDS))
            workers.submit{RemoteImageUrls.clear()}.get(1,TimeUnit.SECONDS)
            resume.countDown();parsing.get(5,TimeUnit.SECONDS)
            assertNull(RemoteImageUrls.resolve(scope,"asset:visible"))
        } finally {resume.countDown();workers.shutdownNow()}
    }
}
