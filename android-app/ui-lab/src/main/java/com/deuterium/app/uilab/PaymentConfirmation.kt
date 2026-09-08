package com.deuterium.app.uilab

import androidx.compose.animation.core.*
import androidx.compose.foundation.Canvas
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.graphics.drawscope.scale
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlin.math.roundToInt

/** Transparent reference motion frames and fitted native confirmation paths; no background shape. */
@Composable
fun PaymentConfirmation(modifier:Modifier,play:Boolean=true,waiting:Boolean=false,authenticating:Boolean=false,onComplete:()->Unit) {
    val context=LocalContext.current
    var movie by remember { mutableStateOf(FacePayAssets.movie) }
    LaunchedEffect(Unit) { PaymentSound.prepare(context);movie=FacePayAssets.load(context) }
    val frame=remember { Animatable(if(authenticating)0f else 21f) }
    val motion=LocalMotion.current
    val haptic=LocalHapticFeedback.current
    val complete by rememberUpdatedState(onComplete)
    var sounded by rememberSaveable { mutableStateOf(false) }
    LaunchedEffect(play,waiting,authenticating,motion) {
        if(!play){frame.snapTo(27f);return@LaunchedEffect}
        if(authenticating){
            if(motion)frame.animateTo(5f,tween(180,easing=LinearEasing)) else frame.snapTo(0f)
            return@LaunchedEffect
        }
        if(waiting) {
            if(!motion){frame.snapTo(21f);return@LaunchedEffect}
            frame.animateTo(21f,tween(((21f-frame.value).coerceAtLeast(0f)*33.333f).roundToInt(),easing=LinearEasing))
            while(isActive){frame.snapTo(9f);frame.animateTo(21f,tween(400,easing=LinearEasing))}
            return@LaunchedEffect
        }
        if(motion)frame.animateTo(27f,tween(((27f-frame.value).coerceAtLeast(0f)*33.333f).roundToInt(),easing=LinearEasing))
        else frame.snapTo(27f)
        delay(if(motion)140 else 70)
        if(!sounded){sounded=true;PaymentSound.confirm();haptic.performHapticFeedback(HapticFeedbackType.Confirm)}
        delay(if(motion)130 else 70)
        complete()
    }
    Canvas(modifier) {
        val asset=movie ?: return@Canvas
        val index=frame.value.roundToInt().coerceIn(0,asset.frames.lastIndex)
        val layers=asset.frames[index]
        val side=minOf(size.width,size.height,188.dp.toPx())
        if(index<22) {
            val tile=asset.viewport.roundToInt()
            drawImage(asset.atlas,srcOffset=androidx.compose.ui.unit.IntOffset(index%6*tile,index/6*tile),srcSize=androidx.compose.ui.unit.IntSize(tile,tile),
                dstOffset=androidx.compose.ui.unit.IntOffset(((size.width-side)*.5f).roundToInt(),((size.height-side)*.5f).roundToInt()),
                dstSize=androidx.compose.ui.unit.IntSize(side.roundToInt(),side.roundToInt()),filterQuality=androidx.compose.ui.graphics.FilterQuality.High)
            return@Canvas
        }
        translate((size.width-side)*.5f,(size.height-side)*.5f) {
            scale(side/asset.viewport,pivot=androidx.compose.ui.geometry.Offset.Zero) {
                layers.forEach { drawPath(it.path,it.color) }
            }
        }
    }
}
