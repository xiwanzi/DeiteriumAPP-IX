package com.deuterium.app.uilab

import org.json.JSONObject
import androidx.compose.runtime.*
import kotlinx.coroutines.delay
import java.time.Instant

/** Single-unit browse preview, never a checkout amount or permission to spend. */
data class ShopOfferPreview(val couponId:String,val couponName:String,val condition:String,val finalPrice:Long,val saving:Long,val stack:Boolean,val expiresAt:Instant?=null)

fun shopMoney(cents:Long)="¥${credit(cents)}"

internal fun shopOfferPreview(value:JSONObject?,salePrice:Long):ShopOfferPreview? {
    if(value==null)return null
    return runCatching {
        val payable=apiCents(value.getString("finalPrice"));val saving=apiCents(value.getString("saving"))
        require(payable>=0&&payable<salePrice&&saving==salePrice-payable)
        val name=value.getString("couponName");val condition=value.getString("condition");val id=value.getString("couponId")
        require(name.isNotBlank()&&condition.isNotBlank()&&id.isNotBlank())
        ShopOfferPreview(id,name,condition,payable,saving,value.optBoolean("stack"),value.optString("expiresAt").takeIf{it.isNotBlank()}?.let{Instant.parse(it)})
    }.getOrNull()
}

@Composable
fun rememberShopOffer(product:ShopProduct):ShopOfferPreview? {
    var offer by remember(product.id,product.offerPreview){mutableStateOf(product.offerPreview?.takeIf{it.expiresAt?.isAfter(Instant.now())!=false})}
    LaunchedEffect(product.id,product.offerPreview) {
        product.offerPreview?.expiresAt?.let{end->delay((end.toEpochMilli()-System.currentTimeMillis()).coerceAtLeast(0));offer=null}
    }
    return offer
}
