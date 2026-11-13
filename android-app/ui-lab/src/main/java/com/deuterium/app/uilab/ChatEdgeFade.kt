package com.deuterium.app.uilab

import androidx.compose.foundation.layout.Box
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithCache
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.graphics.layer.CompositingStrategy
import androidx.compose.ui.graphics.layer.GraphicsLayer
import androidx.compose.ui.graphics.layer.drawLayer
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import kotlin.math.roundToInt

/** Restore the original chat background over messages. Never paint a generic black/white veil. */
internal fun Modifier.chatBottomFade(dockHeight: Dp, fallback: Color, background: GraphicsLayer?,
    origin: () -> Offset, fadeLayer: GraphicsLayer): Modifier = drawWithCache {
    val top = (size.height - dockHeight.toPx()).coerceIn(0f, size.height)
    val bounds = IntSize(size.width.roundToInt(), (size.height - top).roundToInt())
    val mask = chatEdgeMask(0f, 48.dp.toPx())
    fadeLayer.compositingStrategy = CompositingStrategy.Offscreen
    onDrawWithContent {
        drawContent()
        if (bounds.width > 0 && bounds.height > 0) translate(0f, top) {
            drawChatBackground(fadeLayer, background, fallback, origin() + Offset(0f, top), bounds, mask)
        }
    }
}

/** Our flat toolbar needs TG's combined wallpaper + title-glass protection, without moving controls. */
@Composable
internal fun ChatTopFade(modifier: Modifier = Modifier, protectTitle: Boolean = true) {
    val background = LocalGlassBackdrop.current
    val fallback = MaterialTheme.colorScheme.background
    val fadeLayer = rememberGraphicsLayer()
    var origin by remember { mutableStateOf(Offset.Zero) }
    Box(modifier.onGloballyPositioned { origin = it.positionInRoot() }.drawWithCache {
        val bounds = IntSize(size.width.roundToInt(), size.height.roundToInt())
        val peak = if (protectTitle) 1f - (1f - 207f / 255f) * (1f - .85f) else 207f / 255f
        val span = if (protectTitle) 24.dp else 48.dp
        val mask = chatEdgeMask(size.height, size.height - minOf(span.toPx(), size.height), peak)
        fadeLayer.compositingStrategy = CompositingStrategy.Offscreen
        onDrawBehind {
            if (bounds.width > 0 && bounds.height > 0)
                drawChatBackground(fadeLayer, background, fallback, origin, bounds, mask)
        }
    })
}

private fun DrawScope.drawChatBackground(fade: GraphicsLayer, background: GraphicsLayer?,
    fallback: Color, origin: Offset, bounds: IntSize, mask: Brush) {
    fade.record(this, layoutDirection, bounds) {
        drawRect(fallback)
        background?.let { translate(-origin.x, -origin.y) { drawLayer(it) } }
        drawRect(mask, blendMode = BlendMode.DstIn)
    }
    drawLayer(fade)
}

// TG's integer alpha steps for its wallpaper fade, expressed as an alpha-only mask.
private fun chatEdgeMask(start: Float, end: Float, peak: Float = 207f / 255f): Brush = Brush.verticalGradient(
    0f to Color.White.copy(alpha = 0f),
    1f / 3f to Color.White.copy(alpha = peak * 85f / 207f),
    2f / 3f to Color.White.copy(alpha = peak * 157f / 207f),
    1f to Color.White.copy(alpha = peak),
    startY = start, endY = end
)
