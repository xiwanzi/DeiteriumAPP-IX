package com.deuterium.app.uilab

import java.math.BigInteger
import java.math.BigDecimal
import java.time.Instant

/** Demo-only coupon rules; production checkout remains entirely server-owned. */
internal data class DemoCoupon(
    val id:String,val name:String,val minimum:Long=0,val amountOff:Long=0,val rate:Int=10000,val cap:Long=0,
    val stack:Boolean=true,val productIds:Set<String> = emptySet(),val storeIds:Set<String> = emptySet(),
    val starts:Instant=Instant.parse("2020-01-01T00:00:00Z"),val ends:Instant=Instant.parse("2099-01-01T00:00:00Z"),
) {
    val condition:String get()=if(amountOff>0){if(minimum>0)"满${shortMoney(minimum)}减${shortMoney(amountOff)}"else"无门槛减${shortMoney(amountOff)}"}
        else "${BigDecimal(rate).movePointLeft(3).stripTrailingZeros().toPlainString()}折"+(if(cap>0)" · 最高减${shortMoney(cap)}"else"")
}

private fun shortMoney(cents:Long)=BigDecimal.valueOf(cents,2).stripTrailingZeros().toPlainString()

// Same per-unit half-up rounding as promotionRateV209 in the Go service.
// A payable rate <= 10000 guarantees the result is in 0..amount (within Long).
internal fun demoRate(amount:Long,rate:Int):Long {
    require(amount>=0&&rate in 1..10000)
    return BigInteger.valueOf(amount).multiply(BigInteger.valueOf(rate.toLong())).add(BigInteger.valueOf(5000)).divide(BigInteger.valueOf(10000)).toLong()
}

internal fun bestDemoCoupon(original:Long,sale:Long,productId:String,storeId:String,coupons:List<DemoCoupon>,now:Instant):ShopOfferPreview? {
    require(original>=sale&&sale>=0)
    var best:ShopOfferPreview?=null
    for(coupon in coupons.sortedWith(compareBy<DemoCoupon>{it.ends}.thenBy{it.id})) {
        if(now<coupon.starts||now>=coupon.ends||coupon.minimum<0||coupon.amountOff<0||coupon.cap<0||coupon.rate !in 1..10000)continue
        if(coupon.productIds.isNotEmpty()&&productId !in coupon.productIds)continue
        if(coupon.storeIds.isNotEmpty()&&storeId !in coupon.storeIds)continue
        val basis=if(coupon.stack)sale else original
        if(basis<coupon.minimum)continue
        val reduction=if(coupon.amountOff>0)coupon.amountOff.coerceAtMost(basis)else (basis-demoRate(basis,coupon.rate)).let{if(coupon.cap>0)it.coerceAtMost(coupon.cap)else it}
        val payable=basis-reduction
        if(payable>=sale||payable>=(best?.finalPrice ?: Long.MAX_VALUE))continue
        best=ShopOfferPreview(coupon.id,coupon.name,coupon.condition,payable,sale-payable,coupon.stack)
    }
    return best
}
