package com.deuterium.app.uilab

import android.os.Build
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.layer.GraphicsLayer
import androidx.compose.ui.graphics.layer.drawLayer
import androidx.compose.ui.graphics.drawscope.translate
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.layout.SubcomposeLayout
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

private class ChatBackdrop(val layer:GraphicsLayer){var origin by mutableStateOf(Offset.Zero)}
private val LocalChatBackdrop=staticCompositionLocalOf<ChatBackdrop?>{null}

internal data class ChatGlassStyle(val parameters:GlassParameters,val tint:Color,val shade:Float)

internal fun chatGlassStyle(materials:GlassMaterials,panel:Boolean,dark:Boolean):ChatGlassStyle {
    val parameters=if(panel){if(dark)materials.chatPanelDark else materials.chatPanelLight}
        else if(dark)materials.chatInputDark else materials.chatInputLight
    return ChatGlassStyle(parameters,
        if(panel){if(dark)Color(0xFF1D2C3D) else Color(0xFFF8FAFC)}
        else if(dark)Color(0xFF1C2A3B) else Color(0xFFF7F9FC),
        if(dark).045f else if(panel).015f else .02f)
}

/** Measure the dock first, then draw messages behind it with the same bottom content inset.
 * Only the message layer is recorded; sampling a layer that also contains glass is cyclic. */
@Composable
internal fun ChatGlassLayout(body:@Composable (Dp)->Unit,composer:@Composable ()->Unit){
    val layer=rememberGraphicsLayer()
    val foreground=rememberGraphicsLayer()
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
                } else Modifier)){body(dock.height.toDp())}
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
    val material=if(Build.VERSION.SDK_INT>=31)style.parameters else style.parameters.copy(opacity=maxOf(style.parameters.opacity,.94f))
    LiquidGlass(modifier,radius=if(panel)28.dp else 26.dp,
        backdrop=backdrop?.layer ?: LocalGlassBackdrop.current,backdropOrigin=backdrop?.origin ?: Offset.Zero,
        tint=style.tint,parameters=material,shadeOpacity=style.shade,topCornersOnly=panel,content=content)
}
