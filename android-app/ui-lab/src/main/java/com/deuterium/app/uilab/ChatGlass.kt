package com.deuterium.app.uilab

import android.os.Build
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.layer.GraphicsLayer
import androidx.compose.ui.graphics.layer.drawLayer
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.layout.SubcomposeLayout
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.DpOffset
import androidx.compose.ui.graphics.Shape
import com.kyant.backdrop.Backdrop
import com.kyant.backdrop.backdrops.LayerBackdrop
import com.kyant.backdrop.backdrops.rememberCanvasBackdrop
import com.kyant.backdrop.backdrops.rememberLayerBackdrop
import com.kyant.backdrop.drawBackdrop
import com.kyant.backdrop.effects.blur
import com.kyant.backdrop.effects.colorControls
import com.kyant.backdrop.effects.lens
import com.kyant.backdrop.highlight.Highlight
import com.kyant.backdrop.shadow.Shadow

private class ChatBackdrop(val layer:GraphicsLayer){var origin by mutableStateOf(Offset.Zero)}
private val LocalChatBackdrop=staticCompositionLocalOf<ChatBackdrop?>{null}
internal val LocalExpressionSurface = staticCompositionLocalOf<LayerBackdrop?> { null }

internal data class ChatGlassStyle(val parameters: GlassParameters, val tint: Color)

internal fun chatGlassStyle(materials:GlassMaterials,panel:Boolean,dark:Boolean):ChatGlassStyle {
    val parameters=if(panel){if(dark)materials.chatPanelDark else materials.chatPanelLight}
        else if(dark)materials.chatInputDark else materials.chatInputLight
    // Calibrated separately: TG's input glass and the frosted keyboard surface differ visibly.
    return ChatGlassStyle(parameters, if (!dark) Color.White
        else if (panel) Color(0xFF2C3749) else Color(0xFF263140))
}

/** Measure the dock first, then draw messages behind it with the same bottom content inset.
 * Only the message layer is recorded; sampling a layer that also contains glass is cyclic. */
@Composable
internal fun ChatGlassLayout(body:@Composable (Dp)->Unit,composer:@Composable ()->Unit){
    val layer=rememberGraphicsLayer()
    val foreground=rememberGraphicsLayer()
    val edgeLayer=rememberGraphicsLayer()
    val scene=LocalGlassBackdrop.current
    val background=MaterialTheme.colorScheme.background
    val backdrop=remember(layer){ChatBackdrop(layer)}
    val enabled=LocalOverlayGlassEnabled.current
    SubcomposeLayout(Modifier.fillMaxSize()) { constraints ->
        val dock=subcompose("dock"){
            CompositionLocalProvider(LocalChatBackdrop provides backdrop){Box(Modifier.fillMaxWidth()){composer()}}
        }.single().measure(constraints.copy(minHeight=0))
        val messages=subcompose("messages"){
            Box(Modifier.fillMaxSize().onGloballyPositioned{backdrop.origin=it.positionInRoot()}
                .chatBottomFade(dock.height.toDp(), background, scene, { backdrop.origin }, edgeLayer)
                .then(if(enabled)Modifier.drawWithContent {
                    foreground.record{this@drawWithContent.drawContent()}
                    layer.record{
                        // Blur the composited backdrop, including its opaque page background;
                        // otherwise sharp source pixels can leak through the blurred alpha.
                        drawRect(background)
                        scene?.let{translate(-backdrop.origin.x,-backdrop.origin.y){drawLayer(it)}}
                        drawLayer(foreground)
                    }
                    drawLayer(foreground)
                } else Modifier)) { body(dock.height.toDp()) }
        }.single().measure(constraints)
        layout(constraints.maxWidth,constraints.maxHeight){
            messages.place(0,0)
            dock.place(0,constraints.maxHeight-dock.height)
        }
    }
}

@Composable
internal fun ChatGlassSurface(modifier:Modifier=Modifier,panel:Boolean=false,content:@Composable BoxScope.()->Unit){
    val colors=MaterialTheme.colorScheme
    val dark=colors.background.red<.5f
    val style=chatGlassStyle(LocalGlassMaterials.current,panel,dark)
    val backdrop=LocalChatBackdrop.current
    val source = backdrop?.layer ?: LocalGlassBackdrop.current
    var position by remember { mutableStateOf(Offset.Zero) }
    val page = rememberCanvasBackdrop {
        source?.let { translate((backdrop?.origin?.x ?: 0f) - position.x,
            (backdrop?.origin?.y ?: 0f) - position.y) { drawLayer(it) } }
    }
    val panelSurface = rememberLayerBackdrop()
    val shape = if (panel) RoundedCornerShape(topStart = 29.dp, topEnd = 29.dp)
        else RoundedCornerShape(22.dp)
    CompositionLocalProvider(LocalExpressionSurface provides if (panel) panelSurface else null) {
        ChatMaterialSurface(
            modifier.onGloballyPositioned { position = it.positionInRoot() },
            if (source == null) null else page, style, shape,
            shadowRadius = if (panel) 0.dp else 1.dp,
            exportedBackdrop = if (panel) panelSurface else null, content = content
        )
    }
}

/** Small controls sample the panel's grid, not the chat again. The capture excludes these controls. */
@Composable
internal fun ExpressionGlassSurface(backdrop: Backdrop?, modifier: Modifier = Modifier,
    content: @Composable BoxScope.() -> Unit) {
    val dark = MaterialTheme.colorScheme.background.red < .5f
    ChatMaterialSurface(modifier, backdrop,
        ChatGlassStyle(GlassParameters(blur = 6f, opacity = .85f, refraction = 7f,
            refractionWidth = 8f, highlight = if (dark) .16f else 1f,
            saturation = 3f, depthEffect = false), if (dark) Color(0xFF20262F) else Color.White),
        RoundedCornerShape(18.dp), shadowRadius = 3.7.dp, content = content)
}

@Composable
private fun ChatMaterialSurface(modifier: Modifier, backdrop: Backdrop?, style: ChatGlassStyle,
    shape: Shape, shadowRadius: Dp, exportedBackdrop: LayerBackdrop? = null,
    content: @Composable BoxScope.() -> Unit) {
    val dark = MaterialTheme.colorScheme.background.red < .5f
    if (!LocalOverlayGlassEnabled.current || Build.VERSION.SDK_INT < 31 || backdrop == null) {
        Box(modifier.clip(shape).background(style.tint), content = content)
        return
    }
    val p = style.parameters
    Box(modifier.drawBackdrop(
        backdrop = backdrop, shape = { shape },
        effects = {
            // The chat palette supplies the tint; avoid the generic dark contrast compression.
            colorControls(saturation = p.saturation)
            blur(p.blur.dp.toPx())
            lens(minOf(p.refractionWidth.dp.toPx(), size.minDimension / 5f),
                p.refraction.dp.toPx(), depthEffect = false)
        },
        highlight = null,
        shadow = if (dark || shadowRadius == 0.dp) null else {
            { Shadow(radius = shadowRadius, offset = DpOffset(0.dp, shadowRadius / 5f),
                color = Color.Black.copy(alpha = if (shadowRadius > 2.dp) .25f else .125f)) }
        },
        exportedBackdrop = exportedBackdrop,
        onDrawSurface = { drawRect(style.tint.copy(alpha = p.opacity)) }
    ).border(.5.dp, Brush.verticalGradient(listOf(Color.White.copy(alpha = p.highlight),
        Color.White.copy(alpha = if (dark) p.highlight * .5f else p.highlight))), shape), content = content)
}
