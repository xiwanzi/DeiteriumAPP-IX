package com.deuterium.app.uilab

import org.json.JSONArray
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import java.time.Instant

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
}
