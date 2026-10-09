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
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.layout.SubcomposeLayout
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

private class ChatBackdrop(val layer:GraphicsLayer){var origin by mutableStateOf(Offset.Zero)}
private val LocalChatBackdrop=staticCompositionLocalOf<ChatBackdrop?>{null}

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
    val base=LocalGlassParameters.current
    val backdrop=LocalChatBackdrop.current
    // The two roles follow the user's material controls, with distinct reading density.
    val opacity=(base.opacity*(if(panel)if(dark)1.10f else 1.18f else if(dark).86f else .90f)).coerceIn(0f,.94f)
    val material=base.copy(
        blur=(base.blur*(if(panel)1.25f else .38f)).coerceIn(0f,96f),
        opacity=if(Build.VERSION.SDK_INT>=31)opacity else maxOf(opacity,.92f),
        refraction=base.refraction*(if(panel).45f else .65f),
        highlight=base.highlight*(if(panel).65f else .85f)
    )
    LiquidGlass(modifier,radius=if(panel)28.dp else 26.dp,
        backdrop=backdrop?.layer ?: LocalGlassBackdrop.current,backdropOrigin=backdrop?.origin ?: Offset.Zero,
        tint=lerp(colors.surface,colors.primary,if(dark).08f else .015f),parameters=material,topCornersOnly=panel,content=content)
}
