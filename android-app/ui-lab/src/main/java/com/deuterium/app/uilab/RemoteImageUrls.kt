package com.deuterium.app.uilab

import androidx.compose.runtime.mutableStateMapOf
import org.json.JSONArray
import org.json.JSONObject
import java.net.URI
import java.time.Instant

/** Business views authorize these signed image URLs; another player's asset is not an owner-only asset lookup. */
internal object RemoteImageUrls {
    private data class Entry(val access:ImageAccess,val expiresAt:Instant,val refreshPath:String?)
    private val urls=mutableStateMapOf<String,Entry>()
    private fun id(scope:FinancialScope,source:String)="${scope.origin}|${scope.owner}|${source.removePrefix("asset:")}"
    @Synchronized
    fun remember(value:Any?,scope:FinancialScope,depth:Int=0,refreshPath:String?=null) {
        if(depth>12)return
        when(value) {
            is JSONObject->{
                val asset=value.optString("assetId");val url=value.optString("url")
                val status=value.optString("status")
                val expired=status in setOf("EXPIRED","REMOVED","DELETED")
                if(asset.isNotBlank()&&((status=="READY"&&safe(url))||expired)) {
                    val previous=urls[id(scope,asset)]
                    val expiry=runCatching{Instant.parse(value.getString("urlExpiresAt"))}.getOrDefault(Instant.now().plusSeconds(60))
                    val retainUntil=if(value.has("retainUntil"))runCatching{Instant.parse(value.getString("retainUntil"))}.getOrNull() else previous?.access?.retainUntil
                    if(urls.size>=1024&&!urls.containsKey(id(scope,asset)))urls.remove(urls.keys.first())
                    urls[id(scope,asset)]=Entry(ImageAccess(url,value.optString("sha256"),value.optString("contentType").takeIf{it.isNotBlank()},retainUntil,expired,value.optString("purpose")=="DISPUTE_EVIDENCE"),expiry,refreshPath ?: previous?.refreshPath)
                }
                value.keys().forEach{remember(value.opt(it),scope,depth+1,refreshPath)}
            }
            is JSONArray->for(index in 0 until value.length())remember(value.opt(index),scope,depth+1,refreshPath)
        }
    }
    @Synchronized
    fun key(scope:FinancialScope,source:String?):String?=if(source?.startsWith("asset:")==true)urls[id(scope,source)]?.access?.url ?: source else source
    @Synchronized
    fun access(scope:FinancialScope,source:String):ImageAccess?=urls[id(scope,source)]?.access
    @Synchronized
    fun refreshPath(scope:FinancialScope,source:String):String?=urls[id(scope,source)]?.refreshPath
    @Synchronized
    fun clear(){urls.clear()}
    @Synchronized
    fun resolve(scope:FinancialScope,source:String,now:Instant=Instant.now()):String? {
        val entry=urls[id(scope,source)] ?: return null
        if(entry.access.expired||entry.access.retainUntil?.isAfter(now)==false)throw ApiFailure("IMAGE_EXPIRED","图片已过期")
        if(!entry.expiresAt.isAfter(now))throw ApiFailure("IMAGE_URL_EXPIRED","图片链接已过期，请刷新当前页面")
        return entry.access.url
    }
    fun safe(url:String)=runCatching{val uri=URI(url);uri.scheme=="https"&&uri.host!=null&&uri.rawUserInfo==null}.getOrDefault(false)
}
