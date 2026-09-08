package com.deuterium.app.uilab

import android.os.SystemClock
import androidx.compose.animation.core.*
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.paneTitle
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.delay

@Composable
fun AppLaunchOverlay(ready:Boolean,motion:Boolean,session:LabSession,nativeReady:Boolean) {
    var visible by remember { mutableStateOf(!session.launchFinished||!ready) }
    var started by remember { mutableLongStateOf(SystemClock.uptimeMillis()) }
    val opacity=remember { Animatable(if(visible)1f else 0f) }
    val arrival=remember { Animatable(if(session.launchFinished)1f else 0f) }
    LaunchedEffect(nativeReady) {
        if(nativeReady){started=SystemClock.uptimeMillis();if(motion)arrival.animateTo(1f,tween(620,easing=FastOutSlowInEasing)) else arrival.snapTo(1f)}
    }
    LaunchedEffect(ready,nativeReady) {
        if(!nativeReady)return@LaunchedEffect
        if(!ready){visible=true;opacity.snapTo(1f)}
        else if(visible) {
            withFrameNanos { };withFrameNanos { }
            if(!session.launchFinished)delay(((if(motion)620L else 100L)-(SystemClock.uptimeMillis()-started)).coerceAtLeast(0))
            opacity.animateTo(0f,tween(if(motion)320 else 100,easing=FastOutSlowInEasing))
            visible=false;session.launchFinished=true
        }
    }
    if(!visible)return
    val ink=MaterialTheme.colorScheme.primary
    val progress=rememberInfiniteTransition(label="launch-progress")
    val orbit by progress.animateFloat(0f,1f,infiniteRepeatable(tween(if(motion)1250 else 100000,easing=LinearEasing)),label="launch-line")
    Box(Modifier.fillMaxSize().graphicsLayer { alpha=opacity.value }.background(MaterialTheme.colorScheme.background)
        .pointerInput(Unit){awaitPointerEventScope{while(true)awaitPointerEvent().changes.forEach{it.consume()}}}
        .semantics{paneTitle="正在打开 Deuterium";liveRegion=LiveRegionMode.Polite},contentAlignment=Alignment.Center) {
        LoginVersionBadge(Modifier.size(104.dp).offset(y=(-94f*arrival.value).dp).graphicsLayer{
            val initialScale=192f/104f
            val scale=initialScale+(1f-initialScale)*arrival.value;scaleX=scale;scaleY=scale
        })
        Column(horizontalAlignment=Alignment.CenterHorizontally,modifier=Modifier.offset(y=25.dp).graphicsLayer{
            alpha=arrival.value;translationY=(1f-arrival.value)*8.dp.toPx()
        }) {
            Text("Deuterium",fontSize=29.sp,fontWeight=FontWeight.SemiBold,letterSpacing=(-.6).sp)
            Text("与你的世界，保持连接",Modifier.padding(top=9.dp),style=MaterialTheme.typography.bodyMedium,color=MaterialTheme.colorScheme.onSurfaceVariant)
            Canvas(Modifier.padding(top=31.dp).size(62.dp,3.dp)) {
                drawLine(ink.copy(alpha=.12f),Offset(0f,center.y),Offset(size.width,center.y),size.height,StrokeCap.Round)
                val x=if(motion)(.15f+.7f*(.5f-.5f*kotlin.math.cos(orbit*2f*Math.PI).toFloat()))*size.width else size.width*.5f
                drawLine(ink.copy(alpha=.55f),Offset(x-size.width*.12f,center.y),Offset(x+size.width*.12f,center.y),size.height,StrokeCap.Round)
            }
        }
        Text("Deuterium IX",Modifier.align(Alignment.BottomCenter).navigationBarsPadding().padding(bottom=36.dp).graphicsLayer{alpha=arrival.value},style=MaterialTheme.typography.labelSmall,color=MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.65f),letterSpacing=2.sp)
    }
}
