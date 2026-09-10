package com.deuterium.app.uilab

import android.graphics.RenderEffect
import android.graphics.Shader
import android.os.Build
import androidx.compose.foundation.layout.Box
import androidx.compose.runtime.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.draw.drawWithCache
import androidx.compose.ui.graphics.*
import androidx.compose.ui.graphics.layer.GraphicsLayer
import androidx.compose.ui.graphics.layer.drawLayer
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.IntSize
import kotlin.math.ceil

/** Sample the header plus a blur margin, with a continuous fade across the complete header. */
@Composable
fun GradientGlassHeader(backdrop:GraphicsLayer,modifier:Modifier=Modifier) {
    val parameters=LocalHeaderGlassParameters.current
    val tint=MaterialTheme.colorScheme.surface
    val frost=rememberGraphicsLayer();val maskLayer=rememberGraphicsLayer()
    Box(modifier.clipToBounds().drawWithCache {
        fun falloff(t:Float)=1f-t*t*(3f-2f*t)
        val stops=(0..32).map{index->val t=index/32f;t to tint.copy(alpha=.93f*falloff(t))}.toTypedArray()
        val wash=Brush.verticalGradient(*stops,startY=0f,endY=size.height)
        val mask=Brush.verticalGradient(*(0..32).map{index->val t=index/32f;t to Color.White.copy(alpha=falloff(t))}.toTypedArray(),startY=0f,endY=size.height)
        frost.renderEffect=if(Build.VERSION.SDK_INT>=31&&parameters.blur>.1f)RenderEffect.createBlurEffect(parameters.blur.dp.toPx(),parameters.blur.dp.toPx(),Shader.TileMode.CLAMP).asComposeRenderEffect() else null
        maskLayer.compositingStrategy=androidx.compose.ui.graphics.layer.CompositingStrategy.Offscreen
        maskLayer.clip=true
        onDrawBehind {
            if(backdrop.size.width>0&&backdrop.size.height>0){
                // Preserve neighboring pixels for the blur without filtering the whole screen.
                val margin=if(Build.VERSION.SDK_INT>=31)ceil(parameters.blur.dp.toPx()*3f).toInt() else 0
                val sample=IntSize(backdrop.size.width,minOf(backdrop.size.height,ceil(size.height).toInt()+margin))
                frost.record(size=sample){drawLayer(backdrop)}
                maskLayer.record{drawLayer(frost);drawRect(mask,blendMode=BlendMode.DstIn)}
                drawLayer(maskLayer)
            }
            drawRect(wash)
        }
    })
}
