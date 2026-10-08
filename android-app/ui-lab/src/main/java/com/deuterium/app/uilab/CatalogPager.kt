package com.deuterium.app.uilab

import androidx.compose.runtime.*
import kotlinx.coroutines.CancellationException
import org.json.JSONObject
import java.net.URLEncoder

/** Each browse/search view owns its cursor; cached detail records are kept separately. */
internal class CatalogPager<T>(
    private val request:suspend (String)->JSONObject,
    private val decode:(JSONObject)->T,
    private val key:(T)->String,
    private val publish:(List<T>)->Unit = {},
) {
    val items=mutableStateListOf<T>()
    var loading by mutableStateOf(false);private set
    var error by mutableStateOf<String?>(null);private set
    var hasMore by mutableStateOf(false);private set
    var query by mutableStateOf("");private set
    private var path:String?=null
    private var cursor:String?=null
    private var generation=0L
    private var loadedAt=0L
    private var failedAppend=false
    private val seen=mutableSetOf<String>()

    suspend fun refresh(query:String="",categoryId:String="",force:Boolean=false,badge:String="",categoryGroup:String="") {
        val next="/store/products?limit=40"+
            (if(query.isBlank())"" else "&q=${encode(query.trim())}")+
            (if(categoryId.isBlank())"" else "&categoryId=${encode(categoryId)}")+
            (if(badge.isBlank())"" else "&badge=${encode(badge)}")+
            (if(categoryGroup.isBlank())"" else "&categoryGroup=${encode(categoryGroup)}")
        if(next==path&&!force&&(loading||(error==null&&loadedAt!=0L&&System.nanoTime()-loadedAt<30_000_000_000L)))return
        val changed=next!=path
        this.query=query.trim()
        path=next;generation++
        if(changed){items.clear();cursor=null;hasMore=false;seen.clear();loadedAt=0}
        read(append=false)
    }

    suspend fun more() {
        if(loading||!hasMore||cursor==null)return
        read(append=true)
    }

    suspend fun retry() {
        if(loading||path==null)return
        read(append=failedAppend)
    }

    private suspend fun read(append:Boolean) {
        val current=generation
        val base=path ?: return
        val requestedCursor=if(append)cursor else null
        loading=true;error=null;failedAppend=append
        try {
            val response=request(base+(requestedCursor?.let{"&cursor=${encode(it)}"} ?: ""))
            val values=response.getJSONArray("items")
            val next=(0 until values.length()).map{decode(values.getJSONObject(it))}
            val page=response.optJSONObject("_page")
            val nextCursor=page?.optString("nextCursor")?.takeUnless{it.isBlank()||it=="null"}
            check(page?.optBoolean("hasMore")!=true||nextCursor!=null){"服务器分页结果不完整，请稍后重试"}
            check(nextCursor==null||(!append||nextCursor !in seen)&&nextCursor!=requestedCursor){"服务器分页游标重复"}
            if(current!=generation)return
            publish(next)
            val merged=if(append)(items+next).associateBy(key).values.toList() else next.distinctBy(key)
            items.clear();items.addAll(merged)
            if(!append)seen.clear()
            nextCursor?.let{seen.add(it)}
            cursor=nextCursor;hasMore=nextCursor!=null;loadedAt=System.nanoTime()
        } catch(cancelled:CancellationException) {
            throw cancelled
        } catch(failure:Exception) {
            if(current==generation)error=failure.message ?: "商城暂时无法加载，请重试"
        } finally {
            if(current==generation)loading=false
        }
    }

    private fun encode(value:String)=URLEncoder.encode(value,"UTF-8")
}
