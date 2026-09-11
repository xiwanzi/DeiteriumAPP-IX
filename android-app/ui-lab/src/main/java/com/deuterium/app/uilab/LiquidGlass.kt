package com.deuterium.app.uilab

import android.graphics.RenderEffect
import android.graphics.RuntimeShader
import android.graphics.Shader
import android.os.Build
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawWithCache
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.*
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.graphics.layer.GraphicsLayer
import androidx.compose.ui.graphics.layer.drawLayer
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import androidx.compose.ui.layout.positionOnScreen
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp

val LocalGlassBackdrop = staticCompositionLocalOf<GraphicsLayer?> { null }

fun Modifier.recordGlassBackdrop(layer: GraphicsLayer): Modifier = drawWithContent {
    layer.record { this@drawWithContent.drawContent() }
    drawLayer(layer)
}

/** Records only the background, never text or controls. No bitmap readback or cyclic layers. */
@Composable
fun LiquidGlass(
    modifier: Modifier = Modifier,
    radius: Dp = 26.dp,
    backdrop: GraphicsLayer? = LocalGlassBackdrop.current,
    backdropOrigin: Offset = Offset.Zero,
    tint: Color = Color.Unspecified,
    onClick: (() -> Unit)? = null,
    enabled:Boolean=LocalOverlayGlassEnabled.current,
    screenCoordinates:Boolean=false,
    parameters:GlassParameters=LocalGlassParameters.current,
    content: @Composable BoxScope.() -> Unit
) {
    val tilt = LocalDeviceTilt.current
    val dark = MaterialTheme.colorScheme.background.red < .5f
    val surface = MaterialTheme.colorScheme.surface
    val wash = if(tint == Color.Unspecified) surface else tint
    val frost = rememberGraphicsLayer()
    val refractionShader = remember { if(Build.VERSION.SDK_INT >= 33) RuntimeShader(GlassShader) else null }
    var position by remember { mutableStateOf(Offset.Zero) }
    Box(modifier.onGloballyPositioned { position = if(screenCoordinates)it.positionOnScreen() else it.positionInRoot() }
        .clip(RoundedCornerShape(radius))
        .then(if(onClick == null) Modifier else Modifier.clickable(onClick = onClick))
        .drawWithCache {
            val corner = radius.toPx()
            val sheen = Brush.linearGradient(listOf(Color.White.copy(alpha = (if(dark) .16f else .46f) * parameters.highlight), Color.Transparent, Color.White.copy(alpha = .05f * parameters.highlight)))
            frost.renderEffect = if(enabled && Build.VERSION.SDK_INT >= 31) {
                val blur = if(parameters.blur > .1f) RenderEffect.createBlurEffect(parameters.blur.dp.toPx(), parameters.blur.dp.toPx(), Shader.TileMode.CLAMP) else null
                if(Build.VERSION.SDK_INT >= 33 && refractionShader != null) {
                    val shader = refractionShader
                    shader.setFloatUniform("resolution", size.width, size.height)
                    shader.setFloatUniform("corner", corner)
                    shader.setFloatUniform("strength", parameters.refraction)
                    val refraction = RenderEffect.createRuntimeShaderEffect(shader, "content")
                    (if(blur == null) refraction else RenderEffect.createChainEffect(refraction, blur)).asComposeRenderEffect()
                } else blur?.asComposeRenderEffect()
            } else null
            onDrawBehind {
                if(enabled && backdrop != null && size.width > 0 && size.height > 0) {
                    frost.record {
                        translate(backdropOrigin.x-position.x, backdropOrigin.y-position.y) { drawLayer(backdrop) }
                    }
                    drawLayer(frost)
                }
                drawRect(wash.copy(alpha = if(!enabled) 1f else parameters.opacity))
                if(enabled) {
                    drawRect(if(dark) Color.Black.copy(alpha=.13f) else Color(0xFF292934).copy(alpha=.085f))
                    val angle = tilt.value
                    drawRect(Brush.radialGradient(listOf(Color.White.copy(alpha = .25f * parameters.highlight), Color.Transparent), center = Offset(size.width*(.25f+angle.x*.55f),size.height*(.2f+angle.y*.6f)), radius = size.width*.8f))
                    drawRect(sheen, alpha = .55f)
                    val movingRim = Brush.linearGradient(listOf(Color.White.copy(alpha=(if(dark).65f else 1f)*parameters.highlight),Color.White.copy(alpha=.12f*parameters.highlight),Color.White.copy(alpha=.65f*parameters.highlight)),
                        start=Offset(size.width*(.1f+angle.x*.5f),size.height*(.1f+angle.y*.5f)),end=Offset(size.width*(.9f-angle.x*.4f),size.height*(.9f-angle.y*.4f)))
                    drawRoundRect(movingRim, topLeft = Offset(.75.dp.toPx(), .75.dp.toPx()),
                        size = androidx.compose.ui.geometry.Size(size.width - 1.5.dp.toPx(), size.height - 1.5.dp.toPx()),
                        cornerRadius = CornerRadius(corner), style = Stroke(.85.dp.toPx()))
                }
            }
        }, content = content)
}

private const val GlassShader = """
uniform shader content;
uniform float2 resolution;
uniform float corner;
uniform float strength;
half4 main(float2 p) {
    float2 halfSize = resolution * 0.5;
    float r = min(corner, min(halfSize.x, halfSize.y));
    float2 q = abs(p - halfSize) - (halfSize - r);
    float d = length(max(q, float2(0.0))) + min(max(q.x, q.y), 0.0) - r;
    float edge = 1.0 - smoothstep(0.0, 20.0, -d);
    float2 normal = normalize(p - halfSize + float2(0.001));
    float2 refracted = clamp(p - normal * edge * edge * strength, float2(0.0), resolution);
    half4 color = content.eval(refracted);
    float light = max(0.0, dot(normal, normalize(float2(-0.6, -1.0))));
    return half4(color.rgb + half3(edge * light * 0.06), color.a);
}
"""

@Composable
fun GlassAtmosphere(modifier: Modifier = Modifier) {
    val dark = MaterialTheme.colorScheme.background.red < .5f
    val base = MaterialTheme.colorScheme.background
    val accent = MaterialTheme.colorScheme.primary
    Spacer(modifier.fillMaxSize().drawWithCache {
        val brushes=atmosphereBrushes(size,dark,accent)
        onDrawBehind { drawRect(base);brushes.forEach{drawRect(it)} }
    })
}

private fun atmosphereBrushes(size:Size,dark:Boolean,accent:Color)=listOf(
    Brush.radialGradient(listOf(accent.copy(alpha=if(dark).03f else .025f),Color.Transparent),
        center=Offset(size.width*.95f,size.height*.15f),radius=size.width*.95f),
    Brush.radialGradient(listOf(Color(0xFFE2C599).copy(alpha=if(dark).02f else .03f),Color.Transparent),
        center=Offset(size.width*.04f,size.height*.63f),radius=size.width*.9f),
    Brush.radialGradient(listOf(Color(0xFF98A8DC).copy(alpha=.025f),Color.Transparent),
        center=Offset(size.width,size.height*.95f),radius=size.width*.9f)
)

/** Reuse the static screen background, while glass samples the original drawing commands.
 * Sampling the cached texture instead introduces an extra filtering/rounding step in the glass. */
@Composable
internal fun GlassScene(atmosphere:GraphicsLayer,backdrop:GraphicsLayer,modifier:Modifier=Modifier,content:@Composable BoxScope.()->Unit) {
    val base=MaterialTheme.colorScheme.background;val accent=MaterialTheme.colorScheme.primary
    val cached=rememberGraphicsLayer();val foreground=rememberGraphicsLayer()
    Box(modifier.drawWithCache {
        val brushes=atmosphereBrushes(size,base.red<.5f,accent)
        val dimensions=IntSize(size.width.toInt(),size.height.toInt())
        atmosphere.record(this,layoutDirection,dimensions){drawRect(base);brushes.forEach{drawRect(it)}}
        cached.compositingStrategy=androidx.compose.ui.graphics.layer.CompositingStrategy.Offscreen
        cached.record(this,layoutDirection,dimensions){drawLayer(atmosphere)}
        onDrawWithContent {
            foreground.record{this@onDrawWithContent.drawContent()}
            backdrop.record{drawLayer(atmosphere);drawLayer(foreground)}
            drawLayer(cached);drawLayer(foreground)
        }
    },content=content)
}
