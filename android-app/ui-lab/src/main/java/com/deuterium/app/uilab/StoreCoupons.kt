package com.deuterium.app.uilab

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ConfirmationNumber
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.*
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.json.JSONObject
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

data class StoreCoupon(val id:String,val name:String,val type:String,val benefit:String,val amountOff:Long,val rate:Int,
    val minimumSpend:Long,val maxDiscount:Long,val stack:Boolean,val scope:String,val startsAt:Instant,val endsAt:Instant) {
    fun visible(now:Instant)=!now.isBefore(startsAt)&&now.isBefore(endsAt)
    val benefitText:String get()=if(benefit=="FIXED")"减 ${credit(amountOff)}" else "${java.math.BigDecimal(rate).divide(java.math.BigDecimal(1000)).stripTrailingZeros().toPlainString()} 折"
    val conditions:String get()=(if(minimumSpend>0)"满 ${credit(minimumSpend)} 信用点可用" else "无门槛")+(if(benefit=="PERCENT"&&maxDiscount>0)" · 最多减 ${credit(maxDiscount)}" else "")
}

fun storeCoupon(value:JSONObject):StoreCoupon {
    val shops=value.optJSONArray("storeIds")?.length() ?: 0
    val products=value.optJSONArray("productIds")?.length() ?: 0
    val scope=value.optString("scopeDescription").takeIf{it.isNotBlank()&&it!="null"}
        ?: "${if(shops==0)"全部店铺" else "$shops 家指定店铺"} · ${if(products==0)"全部商品" else "$products 件指定商品"}"
    return StoreCoupon(value.getString("couponId"),value.getString("name"),value.getString("type"),value.getString("benefit"),
        apiCents(value.getString("amountOff")),value.getInt("discountRate"),apiCents(value.getString("minimumSpend")),
        apiCents(value.getString("maxDiscount")),value.getBoolean("stackWithProductDiscount"),scope,
        Instant.parse(value.getString("startsAt")),Instant.parse(value.getString("endsAt")))
}

fun productLimitDescriptions(value:JSONObject?):List<String> {
    if(value==null)return emptyList()
    val result=mutableListOf<String>()
    val days=listOf("一","二","三","四","五","六","日")
    for((key,label) in listOf("lifetime" to "累计","daily" to "每日","weekly" to "每周","monthly" to "每月")) {
        val count=value.optInt(key);if(count<=0)continue
        val reset=when(key){"daily"->"每日 ${value.optString("dailyTime","00:00")} 刷新";"weekly"->"周${days[(value.optInt("weeklyDay",1)-1).coerceIn(0,6)]} ${value.optString("weeklyTime","00:00")} 刷新";"monthly"->"每月 ${value.optInt("monthlyDay",1)} 日 ${value.optString("monthlyTime","00:00")} 刷新";else->""}
        result+="每位玩家${label}限购 $count 件${if(reset.isBlank())"" else " · $reset"}"
    }
    return result
}

@Composable
fun StoreCouponsPage(state:LabState,topInset:Dp) {
    val network=state.commerce.network
    val scope=rememberCoroutineScope()
    var now by remember{mutableStateOf(network?.couponNow() ?: Instant.now())}
    LaunchedEffect(network){network?.refreshCoupons();while(true){now=network?.couponNow() ?: Instant.now();delay(1000)}}
    LaunchedEffect(network){while(true){delay(30000);network?.refreshCoupons()}}
    val coupons=network?.coupons?.filter{it.visible(now)} ?: emptyList()
    LazyColumn(contentPadding=PaddingValues(start=20.dp,end=20.dp,top=topInset,bottom=40.dp),verticalArrangement=Arrangement.spacedBy(18.dp)) {
        item{Column(Modifier.padding(horizontal=4.dp,vertical=8.dp)){Text("为你准备的优惠",style=MaterialTheme.typography.headlineLarge);Text("每笔结算自动选用一张最优惠的券。",Modifier.padding(top=10.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)}}
        network?.couponError?.let{message->item{Text(message,color=MaterialTheme.colorScheme.error)}}
        if(coupons.isEmpty())item{
            Column(Modifier.fillMaxWidth().padding(vertical=64.dp),horizontalAlignment=Alignment.CenterHorizontally){
                Icon(Icons.Outlined.ConfirmationNumber,null,Modifier.size(44.dp),tint=MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.6f))
                Text(if(network?.couponsLoading==true)"正在查看优惠" else "期待下一份优惠",Modifier.padding(top=22.dp),style=MaterialTheme.typography.titleLarge)
                Text("优惠发放后，会自动出现在这里。",Modifier.padding(top=10.dp),color=MaterialTheme.colorScheme.onSurfaceVariant,style=MaterialTheme.typography.bodyMedium)
            }
        }
        items(coupons,key={it.id}){coupon->StoreCouponCard(coupon)}
        item{Row(Modifier.fillMaxWidth(),horizontalArrangement=Arrangement.Center){PlainButton({scope.launch{network?.refreshCoupons()}},enabled=network?.couponsLoading!=true){Text("刷新优惠")};if(network?.hasMoreCoupons==true)PlainButton({scope.launch{network.refreshCoupons(true)}},enabled=!network.couponsLoading){Text("查看更多")}}}
    }
}

@Composable
private fun StoreCouponCard(coupon:StoreCoupon) {
    Surface(shape=RoundedCornerShape(26.dp),color=MaterialTheme.colorScheme.surface,modifier=Modifier.fillMaxWidth()) {
        Column(Modifier.padding(24.dp)) {
            Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.SpaceBetween){Icon(Icons.Outlined.ConfirmationNumber,null,Modifier.size(25.dp),tint=MaterialTheme.colorScheme.primary);Text(if(coupon.type=="ITEM")"单品优惠" else "整单优惠",style=MaterialTheme.typography.labelMedium,color=MaterialTheme.colorScheme.primary)}
            Text(coupon.name,Modifier.padding(top=24.dp),style=MaterialTheme.typography.titleLarge)
            Text(coupon.benefitText,Modifier.padding(top=14.dp),fontSize=38.sp,letterSpacing=(-1.2).sp,fontWeight=FontWeight.SemiBold)
            Text(coupon.conditions,Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodyMedium)
            Text(coupon.scope,Modifier.padding(top=14.dp,bottom=22.dp),color=MaterialTheme.colorScheme.onSurfaceVariant,style=MaterialTheme.typography.bodySmall)
            HorizontalDivider(color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.4f))
            Text((if(coupon.stack)"可与商品折扣同享" else "与商品折扣自动择优")+(if(coupon.type=="ITEM")" · 仅限一件" else ""),Modifier.padding(top=16.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
            Text("有效至 ${DateTimeFormatter.ofPattern("M月d日 HH:mm").withZone(ZoneId.of("Asia/Shanghai")).format(coupon.endsAt)} · 北京时间",Modifier.padding(top=7.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
            Row(Modifier.padding(top=12.dp),verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.spacedBy(6.dp)){Icon(Icons.Outlined.Check,null,Modifier.size(14.dp),tint=MaterialTheme.colorScheme.primary);Text("无需领取，结算自动使用",style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.primary)}
        }
    }
}

@Composable
fun ShopPriceLabel(product:ShopProduct,quantity:Int=1,modifier:Modifier=Modifier) {
    Column(modifier){Text("${credit(product.price*quantity)} 信用点",style=MaterialTheme.typography.bodyMedium);if(product.originalPrice>product.price)Text(credit(product.originalPrice*quantity),style=MaterialTheme.typography.bodySmall,textDecoration=TextDecoration.LineThrough,color=MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.55f))}
}
