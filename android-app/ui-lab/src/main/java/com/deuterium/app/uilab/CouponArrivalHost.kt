package com.deuterium.app.uilab

import androidx.compose.animation.*
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.ConfirmationNumber
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.*
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.*
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.repeatOnLifecycle
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun CouponArrivalHost(attention:CouponAttention,ready:Boolean,topInset:Dp,onOpen:()->Unit,modifier:Modifier=Modifier,allowPopup:Boolean=true) {
    val lifecycle=LocalLifecycleOwner.current
    val overlays=LocalIosOverlayRegistry.current
    val scope=rememberCoroutineScope()
    var fresh by remember(attention){mutableStateOf(false)}
    var welcome by remember(attention){mutableStateOf(true)}
    var popup by remember(attention){mutableStateOf<List<StoreCoupon>?>(null)}
    var banner by remember(attention){mutableStateOf<List<StoreCoupon>?>(null)}
    val motion=LocalMotion.current
    LaunchedEffect(attention,lifecycle) {
        lifecycle.repeatOnLifecycle(Lifecycle.State.RESUMED) {
            welcome=true;fresh=false
            val ticker=launch{while(true){attention.tick();delay(1000)}}
            try {
                while(true) {
                    if(attention.refresh())fresh=true
                    delay(30000)
                }
            } finally {ticker.cancel();fresh=false;popup=null;banner=null}
        }
    }
    val incoming=attention.arrivals
    val idle=ready&&(overlays?.count ?: 0)==0
    LaunchedEffect(fresh,idle,allowPopup,incoming.map{it.id},popup!=null,banner!=null) {
        if(!fresh||!idle||popup!=null||banner!=null)return@LaunchedEffect
        if(incoming.isEmpty()){welcome=false;return@LaunchedEffect}
        // Let the destination finish appearing before introducing another surface.
        delay(350)
        val batch=attention.arrivals
        if(batch.isEmpty())return@LaunchedEffect
        if(welcome&&allowPopup)popup=batch else banner=batch
        welcome=false
    }
    val activePopup=popup?.filter{coupon->attention.unread.any{it.id==coupon.id}}
    val activeBanner=banner?.filter{coupon->attention.unread.any{it.id==coupon.id}}
    LaunchedEffect(popup?.map{it.id},banner?.map{it.id}) {
        val shown=popup ?: banner ?: return@LaunchedEffect
        withFrameNanos{}
        attention.mark(shown);attention.flush()
    }
    LaunchedEffect(activePopup?.size,activeBanner?.size) {
        if(activePopup?.isEmpty()==true)popup=null
        if(activeBanner?.isEmpty()==true)banner=null
    }
    LaunchedEffect(banner){if(banner!=null){delay(6000);banner=null}}
    Box(modifier) {
        AnimatedVisibility(!activeBanner.isNullOrEmpty()&&idle,Modifier.align(Alignment.TopCenter).padding(top=topInset,start=16.dp,end=16.dp),
            enter=if(motion)fadeIn()+slideInVertically{-it/2} else EnterTransition.None,
            exit=if(motion)fadeOut()+slideOutVertically{-it/2} else ExitTransition.None) {
            val batch=activeBanner.orEmpty()
            LiquidGlass(Modifier.fillMaxWidth().semantics{liveRegion=LiveRegionMode.Polite},backdrop=LocalOverlayBackdrop.current,enabled=LocalOverlayGlassEnabled.current,onClick={banner=null;onOpen()}) {
                Row(Modifier.padding(start=16.dp,top=12.dp,end=8.dp,bottom=12.dp),verticalAlignment=Alignment.CenterVertically) {
                    Icon(Icons.Outlined.ConfirmationNumber,null,Modifier.size(26.dp),tint=MaterialTheme.colorScheme.primary)
                    Column(Modifier.weight(1f).padding(horizontal=12.dp)) {
                        Text(if(batch.size==1)"收到一份新优惠" else "收到 ${batch.size} 份新优惠",style=MaterialTheme.typography.titleMedium)
                        Text(batch.singleOrNull()?.let{"${it.name} · ${it.benefitText}"} ?: "已放入我的优惠，点此查看",Modifier.padding(top=3.dp),style=MaterialTheme.typography.bodySmall,maxLines=2,overflow=TextOverflow.Ellipsis)
                    }
                    IconButton({banner=null},Modifier.size(44.dp)){Icon(Icons.Outlined.Close,"关闭优惠提醒",Modifier.size(18.dp))}
                }
            }
        }
    }
    if(!activePopup.isNullOrEmpty()) {
        val batch=activePopup
        CouponArrivalDialog(batch,onDismiss={
            if(!couponArrivalNeedsDetails(batch))attention.mark(batch,viewed=true)
            popup=null;scope.launch{attention.flush()}
        },onOpen={popup=null;onOpen()})
    }
}

@Composable
fun CouponArrivalDialog(coupons:List<StoreCoupon>,onDismiss:()->Unit,onOpen:()->Unit) {
    val details=couponArrivalNeedsDetails(coupons)
    IosDialog(onDismiss,{
        Column(horizontalAlignment=Alignment.CenterHorizontally) {
            Box(Modifier.padding(bottom=16.dp).size(48.dp).background(MaterialTheme.colorScheme.primary.copy(alpha=.10f),RoundedCornerShape(15.dp)),contentAlignment=Alignment.Center){
                Icon(Icons.Outlined.ConfirmationNumber,null,Modifier.size(27.dp),tint=MaterialTheme.colorScheme.primary)
            }
            Text(if(coupons.size==1)"收到新的优惠" else "收到 ${coupons.size} 份新优惠",fontSize=21.sp,fontWeight=FontWeight.SemiBold)
        }
    },{
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()),horizontalAlignment=Alignment.CenterHorizontally) {
            Text("已放入「我的优惠」",style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
            if(coupons.size==1) {
                val coupon=coupons.single()
                Text(coupon.benefitText,Modifier.padding(top=19.dp,bottom=8.dp),fontSize=38.sp,lineHeight=44.sp,letterSpacing=(-1).sp,fontWeight=FontWeight.SemiBold)
                Text(coupon.name,style=MaterialTheme.typography.titleMedium)
                Text(coupon.conditions,Modifier.padding(top=12.dp),style=MaterialTheme.typography.bodyMedium)
                Text(coupon.scope,Modifier.padding(top=8.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant,maxLines=2,overflow=TextOverflow.Ellipsis)
                Text((if(coupon.stack)"可与商品折扣同享" else "与商品折扣自动择优")+(if(coupon.type=="ITEM")" · 仅限一件" else ""),Modifier.padding(top=7.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                Text("有效至 ${DateTimeFormatter.ofPattern("M月d日 HH:mm").withZone(ZoneId.of("Asia/Shanghai")).format(coupon.endsAt)} · 北京时间",Modifier.padding(top=14.dp),style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
            } else {
                Column(Modifier.padding(top=16.dp),verticalArrangement=Arrangement.spacedBy(10.dp)) {
                    coupons.take(3).forEach{coupon->
                        Row(Modifier.fillMaxWidth().background(MaterialTheme.colorScheme.primary.copy(alpha=.055f),RoundedCornerShape(13.dp)).padding(13.dp),verticalAlignment=Alignment.CenterVertically) {
                            Column(Modifier.weight(1f).padding(end=10.dp)) {
                                Text(coupon.name,style=MaterialTheme.typography.bodyMedium,textAlign=TextAlign.Start,maxLines=1,overflow=TextOverflow.Ellipsis)
                                Text(if(coupon.type=="ITEM")"单品优惠" else "整单优惠",Modifier.padding(top=3.dp),style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            Text(coupon.benefitText,style=MaterialTheme.typography.titleMedium,color=MaterialTheme.colorScheme.primary)
                        }
                    }
                    if(coupons.size>3)Text("还有 ${coupons.size-3} 份优惠，一起去看看。",Modifier.fillMaxWidth(),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
                }
                Text("适用范围与有效期可在优惠页查看。",Modifier.fillMaxWidth().padding(top=16.dp),style=MaterialTheme.typography.bodySmall,textAlign=TextAlign.Center,color=MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Text("结算时自动为你选用最优惠的一张。",Modifier.fillMaxWidth().padding(top=12.dp),style=MaterialTheme.typography.labelSmall,textAlign=TextAlign.Center,color=MaterialTheme.colorScheme.onSurfaceVariant)
        }
    },{
        PlainButton(if(details)onOpen else onDismiss,Modifier.fillMaxWidth().heightIn(min=52.dp)) {
            Text(if(details)"去看看" else "好的",fontWeight=FontWeight.SemiBold)
        }
    },if(details)({PlainButton(onDismiss,Modifier.fillMaxWidth().heightIn(min=52.dp)){Text("好的")}}) else null)
}
