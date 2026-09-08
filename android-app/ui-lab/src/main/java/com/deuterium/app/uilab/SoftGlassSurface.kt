package com.deuterium.app.uilab

import androidx.compose.foundation.layout.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.layer.GraphicsLayer
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

val LocalOverlayGlassEnabled=staticCompositionLocalOf{true}
val LocalOverlayBackdrop=staticCompositionLocalOf<GraphicsLayer?>{null}

@Composable
fun SoftGlassSurface(modifier:Modifier=Modifier,radius:Dp=28.dp,tint:Color=Color.Unspecified,content:@Composable BoxScope.()->Unit) {
    LiquidGlass(modifier,radius=radius,backdrop=LocalOverlayBackdrop.current,tint=tint,enabled=LocalOverlayGlassEnabled.current,screenCoordinates=true,content=content)
}
