package com.deuterium.app.uilab

import androidx.compose.animation.*
import androidx.compose.foundation.layout.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.*
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.repeatOnLifecycle
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun CouponArrivalHost(attention:CouponAttention,ready:Boolean,topInset:Dp,onOpen:()->Unit,modifier:Modifier=Modifier) {
    val lifecycle=LocalLifecycleOwner.current
    val overlays=LocalIosOverlayRegistry.current
    var fresh by remember(attention){mutableStateOf(false)}
    var banner by remember(attention){mutableStateOf<List<StoreCoupon>?>(null)}
    val motion=LocalMotion.current
    LaunchedEffect(attention,lifecycle) {
        lifecycle.repeatOnLifecycle(Lifecycle.State.RESUMED) {
            fresh=false
            val ticker=launch{while(true){attention.tick();delay(1000)}}
            try {
                while(true) {
                    if(attention.refresh())fresh=true
                    delay(30000)
                }
            } finally {ticker.cancel();fresh=false;banner=null}
        }
    }
    val incoming=attention.arrivals
    val idle=ready&&(overlays?.count ?: 0)==0
    LaunchedEffect(fresh,idle,incoming.map{it.id},banner!=null) {
        if(!fresh||!idle||banner!=null||incoming.isEmpty())return@LaunchedEffect
        delay(350)
        attention.arrivals.takeIf{it.isNotEmpty()}?.let{banner=it}
    }
    val activeBanner=banner?.filter{coupon->attention.unread.any{it.id==coupon.id}}
    val shown=idle&&!activeBanner.isNullOrEmpty()
    LaunchedEffect(banner?.map{it.id},shown) {
        if(!shown)return@LaunchedEffect
        withFrameNanos{}
        attention.mark(activeBanner.orEmpty());attention.flush()
    }
    LaunchedEffect(activeBanner?.isEmpty()){if(activeBanner?.isEmpty()==true)banner=null}
    LaunchedEffect(banner,shown){if(shown){delay(6000);banner=null}}
    Box(modifier) {
        AnimatedVisibility(shown,Modifier.align(Alignment.TopCenter).padding(top=topInset,start=16.dp,end=16.dp),
            enter=if(motion)fadeIn()+slideInVertically{-it/2} else EnterTransition.None,
            exit=if(motion)fadeOut()+slideOutVertically{-it/2} else ExitTransition.None) {
            val batch=activeBanner.orEmpty()
            InAppNoticeCard(
                if(batch.size==1)"收到一份新优惠" else "收到 ${batch.size} 份新优惠",
                batch.singleOrNull()?.let{"${it.name} · ${it.benefitText}"} ?: "已放入我的优惠，点此查看",
                onOpen={banner=null;onOpen()},onDismiss={banner=null})
        }
    }
}
