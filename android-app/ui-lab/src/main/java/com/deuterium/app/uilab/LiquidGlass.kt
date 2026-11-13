package com.deuterium.app.uilab

import android.os.Build
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawWithCache
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
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
import com.kyant.backdrop.backdrops.LayerBackdrop
import com.kyant.backdrop.backdrops.layerBackdrop
import com.kyant.backdrop.backdrops.rememberCanvasBackdrop
import com.kyant.backdrop.drawBackdrop
import com.kyant.backdrop.effects.blur
import com.kyant.backdrop.effects.lens
import com.kyant.backdrop.effects.colorControls
import com.kyant.backdrop.highlight.Highlight
import com.kyant.backdrop.highlight.HighlightStyle

val LocalGlassBackdrop = staticCompositionLocalOf<GraphicsLayer?> { null }

fun Modifier.recordGlassBackdrop(layer: GraphicsLayer): Modifier = drawWithContent {
    layer.record { this@drawWithContent.drawContent() }
    drawLayer(layer)
}

/** The source contains the page beneath the surface. Exported material excludes its children. */
@Composable
fun LiquidGlass(
    modifier: Modifier = Modifier,
    radius: Dp = 26.dp,
    backdrop: GraphicsLayer? = LocalGlassBackdrop.current,
    backdropOrigin: Offset = Offset.Zero,
    tint: Color = Color.Unspecified,
    onClick: (() -> Unit)? = null,
    enabled: Boolean = LocalOverlayGlassEnabled.current,
    screenCoordinates: Boolean = false,
    parameters: GlassParameters = LocalGlassParameters.current,
    topCornersOnly: Boolean = false,
    surfaceShape: Shape? = null,
    exportedBackdrop: LayerBackdrop? = null,
    content: @Composable BoxScope.() -> Unit
) {
    val tilt = LocalDeviceTilt.current
    val light = MaterialTheme.colorScheme.background.red >= .5f
    val surface = if (tint == Color.Unspecified) MaterialTheme.colorScheme.surface else tint
    val shape = surfaceShape ?: if (topCornersOnly) RoundedCornerShape(topStart = radius, topEnd = radius)
        else RoundedCornerShape(radius)
    val clickable = if (onClick == null) Modifier else Modifier.clickable(onClick = onClick)
    if (!enabled || Build.VERSION.SDK_INT < 31 || backdrop == null) {
        Box(modifier.clip(shape).background(surface).then(clickable), content = content)
        return
    }

    var position by remember { mutableStateOf(Offset.Zero) }
    val page = rememberCanvasBackdrop {
        translate(backdropOrigin.x - position.x, backdropOrigin.y - position.y) {
            drawLayer(backdrop)
        }
    }
    Box(
        modifier.onGloballyPositioned { coordinates ->
            position = if (screenCoordinates) coordinates.positionOnScreen() else coordinates.positionInRoot()
        }.drawBackdrop(
            backdrop = page,
            shape = { shape },
            effects = {
                colorControls(
                    brightness = if (light) parameters.brightness else -.14f,
                    contrast = if (light) 1f else .50f,
                    saturation = parameters.saturation
                )
                blur(parameters.blur.dp.toPx())
                val edgeWidth = minOf(parameters.refractionWidth.dp.toPx(), radius.toPx(), size.minDimension / 2f)
                val displacement = minOf(parameters.refraction.dp.toPx(), size.minDimension)
                lens(edgeWidth, displacement, depthEffect = parameters.depthEffect)
            },
            highlight = {
                val angle = tilt.value
                if (parameters.plainHighlight) Highlight.Plain.copy(alpha = parameters.highlight)
                else Highlight.Default.copy(
                    alpha = parameters.highlight,
                    style = HighlightStyle.Default(angle = 45f + 90f * angle.x + 45f * angle.y)
                )
            },
            shadow = null,
            exportedBackdrop = exportedBackdrop,
            onDrawSurface = { drawRect(surface.copy(alpha = parameters.opacity)) }
        ).then(clickable),
        content = content
    )
}

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
internal fun GlassScene(
    atmosphere: GraphicsLayer,
    backdrop: GraphicsLayer,
    modifier: Modifier = Modifier,
    foreground: GraphicsLayer = rememberGraphicsLayer(),
    pageBackdrop: LayerBackdrop? = null,
    content: @Composable BoxScope.() -> Unit
) {
    val base=MaterialTheme.colorScheme.background;val accent=MaterialTheme.colorScheme.primary
    val cached=rememberGraphicsLayer()
    val capture = if (pageBackdrop == null) Modifier else Modifier.layerBackdrop(pageBackdrop)
    Box(modifier.then(capture).drawWithCache {
        val brushes=atmosphereBrushes(size,base.red<.5f,accent)
        val dimensions=IntSize(size.width.toInt(),size.height.toInt())
        atmosphere.record(this,layoutDirection,dimensions){drawRect(base);brushes.forEach{drawRect(it)}}
        cached.compositingStrategy=androidx.compose.ui.graphics.layer.CompositingStrategy.Offscreen
        cached.record(this,layoutDirection,dimensions){drawLayer(atmosphere)}
        onDrawWithContent {
            foreground.record{this@onDrawWithContent.drawContent()}
            if (pageBackdrop == null) backdrop.record{drawLayer(atmosphere);drawLayer(foreground)}
            drawLayer(cached);drawLayer(foreground)
        }
    },content=content)
}
