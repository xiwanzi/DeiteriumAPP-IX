package com.deuterium.app.uilab

import android.os.Build
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.layer.GraphicsLayer
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.kyant.backdrop.backdrops.rememberLayerBackdrop
import com.kyant.shapes.RoundedRectangle

val LocalOverlayGlassEnabled = staticCompositionLocalOf { true }
val LocalOverlayBackdrop = staticCompositionLocalOf<GraphicsLayer?> { null }

@Composable
fun SoftGlassSurface(
    modifier: Modifier = Modifier,
    radius: Dp = 44.dp,
    tint: Color = Color.Unspecified,
    parameters: GlassParameters = LocalGlassParameters.current,
    content: @Composable BoxScope.() -> Unit
) {
    val light = MaterialTheme.colorScheme.background.red >= .5f
    val controls = if (Build.VERSION.SDK_INT >= 31 && LocalOverlayGlassEnabled.current) {
        rememberLayerBackdrop()
    } else null
    LiquidGlass(
        modifier = modifier,
        radius = radius,
        backdrop = LocalOverlayBackdrop.current,
        tint = if (tint == Color.Unspecified) kyantSheetTint(light) else tint,
        parameters = parameters,
        screenCoordinates = true,
        exportedBackdrop = controls
    ) {
        CompositionLocalProvider(LocalControlBackdrop provides controls) { content() }
    }
}

@Composable
internal fun KyantDialogGlass(
    modifier: Modifier = Modifier,
    parameters: GlassParameters? = null,
    content: @Composable BoxScope.() -> Unit
) {
    val light = MaterialTheme.colorScheme.background.red >= .5f
    val controls = if (Build.VERSION.SDK_INT >= 31 && LocalOverlayGlassEnabled.current) {
        rememberLayerBackdrop()
    } else null
    LiquidGlass(
        modifier = modifier,
        radius = 48.dp,
        backdrop = LocalOverlayBackdrop.current,
        tint = kyantCatalogTint(light),
        screenCoordinates = true,
        parameters = parameters ?: kyantDialogParameters(light),
        surfaceShape = RoundedRectangle(48.dp),
        exportedBackdrop = controls
    ) {
        CompositionLocalProvider(LocalControlBackdrop provides controls) { content() }
    }
}
