package com.deuterium.app.uilab

import androidx.compose.runtime.*
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import org.json.JSONArray
import org.json.JSONObject
import java.net.URLEncoder
import java.time.Instant

internal data class CouponReceipt(val level:Int,val endsAt:Instant,val pending:Boolean,val releaseBatchId:String?=null)

/** Server receipts synchronize devices; the small journal survives an offline ack. */
class CouponAttention(private val api:BackendApi) {
    private val owner=api.financialScope()
    private val lock=Mutex()
    private val receipts=mutableStateMapOf<String,CouponReceipt>()
    private val announced=mutableStateMapOf<String,Boolean>()
    private val available=mutableStateListOf<StoreCoupon>()
    private var serverClock=Instant.now()
    private var clockNanos=System.nanoTime()
    var now by mutableStateOf(serverClock);private set
    val unread:List<StoreCoupon> get()=available.filter{it.visible(now)&&(receipts[it.id]?.level ?: 0)<2}
    val arrivals:List<StoreCoupon> get() {
        val shownBatches=receipts.values.mapNotNull{it.releaseBatchId}.toSet()
        return unread.filter{announced[it.id]!=true&&(receipts[it.id]?.level ?: 0)<1&&(it.releaseBatchId==null||it.releaseBatchId !in shownBatches)}
    }
    val hasUnread:Boolean get()=unread.isNotEmpty()

    init {
        val saved=api.couponReceipts(owner)
        saved.keys().forEach{id->runCatching {
            val row=saved.getJSONObject(id)
            CouponReceipt(row.getInt("level").coerceIn(1,2),Instant.parse(row.getString("endsAt")),row.getBoolean("pending"),row.optString("releaseBatchId").takeUnless{it.isBlank()||it=="null"})
        }.getOrNull()?.let{receipts[id]=it}}
    }

    fun tick(){now=serverClock.plusNanos((System.nanoTime()-clockNanos).coerceAtLeast(0))}

    suspend fun refresh():Boolean=lock.withLock {
        try {
            owner.verifyCurrent(api.financialScope())
            flushLocked()
            val next=mutableListOf<StoreCoupon>();val notices=mutableMapOf<String,Boolean>()
            val cursors=mutableSetOf<String>();var cursor:String?=null
            do {
                owner.verifyCurrent(api.financialScope())
                val value=api.request("GET","/store/coupons/attention?limit=100"+(cursor?.let{"&cursor=${URLEncoder.encode(it,"UTF-8")}"} ?: ""))
                owner.verifyCurrent(api.financialScope())
                val clock=Instant.parse(value.getString("_serverTime"))
                val items=value.getJSONArray("items")
                for(index in 0 until items.length()) {
                    val item=items.getJSONObject(index);val coupon=storeCoupon(item)
                    next.add(coupon);notices[coupon.id]=item.getBoolean("announced")
                }
                val page=value.optJSONObject("_page")
                cursor=page?.optString("nextCursor")?.takeUnless{it.isBlank()||it=="null"}
                check(page?.optBoolean("hasMore")!=true||cursor!=null)
                if(cursor!=null)check(cursors.add(cursor))
                serverClock=clock;clockNanos=System.nanoTime()
            } while(cursor!=null)
            tick()
            available.clear();available.addAll(next.distinctBy{it.id})
            announced.clear();announced.putAll(notices)
            // Keep pending receipts even after expiry until the server accepts them.
            if(receipts.entries.removeAll{!it.value.pending&&!now.isBefore(it.value.endsAt)})persist()
            true
        } catch(cancel:CancellationException){throw cancel}
        catch(_:Exception){false}
    }

    fun mark(coupons:List<StoreCoupon>,viewed:Boolean=false) {
        if(owner.owner.isBlank()||owner!=api.financialScope())return
        val level=if(viewed)2 else 1
        var changed=false
        for(coupon in coupons) {
            if((receipts[coupon.id]?.level ?: 0)>=level)continue
            receipts[coupon.id]=CouponReceipt(level,coupon.endsAt,true,coupon.releaseBatchId);changed=true
        }
        if(changed)persist()
    }

    suspend fun flush()=lock.withLock{flushLocked()}

    private suspend fun flushLocked() {
        for(level in 1..2) {
            val ids=receipts.filterValues{it.pending&&it.level==level}.keys.toList()
            for(batch in ids.chunked(100)) {
                try {
                    owner.verifyCurrent(api.financialScope())
                    val response=api.request("POST","/store/coupons/attention",JSONObject().put("couponIds",JSONArray(batch)).put("viewed",level==2))
                    owner.verifyCurrent(api.financialScope())
                    check(response.getBoolean("acknowledged"))
                    for(id in batch)receipts[id]?.takeIf{it.level==level}?.let{receipts[id]=it.copy(pending=false)}
                    persist()
                } catch(cancel:CancellationException){throw cancel}
                catch(_:Exception){return}
            }
        }
    }

    private fun persist() {
        val value=JSONObject()
        receipts.forEach{(id,receipt)->value.put(id,JSONObject().put("level",receipt.level).put("endsAt",receipt.endsAt.toString()).put("pending",receipt.pending).put("releaseBatchId",receipt.releaseBatchId))}
        api.saveCouponReceipts(owner,value)
    }
}

fun couponArrivalNeedsDetails(coupons:List<StoreCoupon>):Boolean {
    val coupon=coupons.singleOrNull() ?: return true
    return coupon.type!="ORDER"||coupon.benefit!="FIXED"||coupon.restricted||coupon.name.length>24||coupon.scope.length>30
}
