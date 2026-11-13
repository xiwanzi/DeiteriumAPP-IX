package com.deuterium.app.uilab

import android.graphics.RenderEffect
import android.os.Build
import androidx.compose.foundation.layout.Box
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.draw.drawWithCache
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.RectangleShape
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.graphics.layer.CompositingStrategy
import androidx.compose.ui.graphics.layer.GraphicsLayer
import androidx.compose.ui.graphics.layer.drawLayer
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.kyant.backdrop.backdrops.rememberCanvasBackdrop
import com.kyant.backdrop.drawPlainBackdrop
import com.kyant.backdrop.effects.blur
import com.kyant.backdrop.effects.colorControls
import com.kyant.backdrop.effects.effect
import kotlin.math.ceil

/** Blur the sampled page and fade it continuously into the unfiltered content. */
@Composable
fun GradientGlassHeader(backdrop: GraphicsLayer, modifier: Modifier = Modifier, protectedHeight: Dp? = null) {
    val parameters = LocalHeaderGlassParameters.current
    val light = MaterialTheme.colorScheme.background.red >= .5f
    val tint = if (light) Color.White else Color(0xFF191D25)
    val tintIntensity = if (protectedHeight != null) {
        if (light) .87f else .76f
    } else if (light) .8f else .60f
    val fadeStart = if (light) .50f else .24f
    val blurRadius = if (light) parameters.blur else maxOf(parameters.blur, 12f)
    val page = rememberCanvasBackdrop { drawLayer(backdrop) }
    if (Build.VERSION.SDK_INT >= 33) {
        Box(modifier.drawPlainBackdrop(
            backdrop = page,
            shape = { RectangleShape },
            effects = {
                if (!light) colorControls(brightness = -.12f, contrast = .62f, saturation = .95f)
                blur(blurRadius.dp.toPx())
                effect(RenderEffect.createRuntimeShaderEffect(
                    obtainRuntimeShader("AlphaMask", ProgressiveBlurShader).apply {
                        setFloatUniform("size", size.width, size.height)
                        setColorUniform("tint", tint.toArgb())
                        setFloatUniform("tintIntensity", tintIntensity)
                        val protectedFraction = protectedHeight?.let {
                            (it.toPx() / size.height.coerceAtLeast(1f)).coerceIn(0f, .95f)
                        }
                        setFloatUniform("fadeStart", protectedFraction ?: fadeStart)
                    },
                    "content"
                ))
            }
        ))
        return
    }
    val maskLayer = rememberGraphicsLayer()
    Box(modifier.clipToBounds().drawWithCache {
        val start = protectedHeight?.let {
            (it.toPx() / size.height.coerceAtLeast(1f)).coerceIn(0f, .95f)
        } ?: fadeStart
        fun falloff(t: Float): Float {
            val p = ((t - start) / (1f - start)).coerceIn(0f, 1f)
            return 1f - p * p * (3f - 2f * p)
        }
        val washStops = (0..32).map { index ->
            val t = index / 32f
            t to tint.copy(alpha = tintIntensity * falloff(t))
        }.toTypedArray()
        val maskStops = (0..32).map { index ->
            val t = index / 32f
            t to Color.White.copy(alpha = falloff(t))
        }.toTypedArray()
        val wash = Brush.verticalGradient(*washStops, startY = 0f, endY = size.height)
        val mask = Brush.verticalGradient(*maskStops, startY = 0f, endY = size.height)
        maskLayer.compositingStrategy = CompositingStrategy.Offscreen
        onDrawWithContent {
            maskLayer.record {
                this@onDrawWithContent.drawContent()
                drawRect(mask, blendMode = BlendMode.DstIn)
            }
            drawLayer(maskLayer)
            drawRect(wash)
        }
    }.drawPlainBackdrop(
        backdrop = page,
        shape = { RectangleShape },
        effects = {
            if (!light) colorControls(brightness = -.12f, contrast = .62f, saturation = .95f)
            val radius = blurRadius.dp.toPx()
            padding = ceil(radius * 3f)
            blur(radius)
        }
    ))
}

// AlphaMask from AndroidLiquidGlass 1.0.6 ProgressiveBlurContent (Apache-2.0).
private const val ProgressiveBlurShader = """
uniform shader content;
uniform float2 size;
layout(color) uniform half4 tint;
uniform float tintIntensity;
uniform float fadeStart;

half4 main(float2 coord) {
    float alpha = 1.0 - smoothstep(size.y * fadeStart, size.y, coord.y);
    return mix(content.eval(coord) * alpha, tint * alpha, tintIntensity);
}
"""
