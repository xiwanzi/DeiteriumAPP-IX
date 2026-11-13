package com.deuterium.app.uilab

import android.os.Build
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ProvideTextStyle
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.kyant.backdrop.Backdrop
import com.kyant.backdrop.backdrops.rememberCanvasBackdrop
import com.kyant.backdrop.catalog.components.LiquidButton
import com.kyant.shapes.Capsule

internal val LocalControlBackdrop = staticCompositionLocalOf<Backdrop?> { null }

internal enum class DialogAction { Normal, Confirm, Cancel, Destructive }

internal val LocalDialogAction = staticCompositionLocalOf { DialogAction.Normal }

@Composable
internal fun rememberControlBackdrop(): Backdrop {
    val colors = MaterialTheme.colorScheme
    val surface = if (colors.background.red >= .5f) colors.surface else kyantCatalogTint(false)
    val base = rememberCanvasBackdrop { drawRect(surface) }
    return LocalControlBackdrop.current ?: base
}

@Composable
internal fun CatalogAction(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    shape: Shape = Capsule(),
    tint: Color = Color.Unspecified,
    surface: Color = Color.Unspecified,
    foreground: Color = MaterialTheme.colorScheme.onSurface,
    minHeight: Dp = 48.dp,
    padding: PaddingValues = PaddingValues(horizontal = 16.dp, vertical = 4.dp),
    role: Role = Role.Button,
    selected: Boolean? = null,
    glass: Boolean = false,
    content: @Composable RowScope.() -> Unit
) {
    val colors = MaterialTheme.colorScheme
    val ink = if (enabled) foreground else colors.onSurfaceVariant.copy(alpha = .78f)
    val label: @Composable RowScope.() -> Unit = {
        CompositionLocalProvider(LocalContentColor provides ink) {
            ProvideTextStyle(MaterialTheme.typography.labelLarge) { content() }
        }
    }
    val backdrop = LocalControlBackdrop.current
    if (glass && enabled && backdrop != null && Build.VERSION.SDK_INT >= 31 && LocalOverlayGlassEnabled.current) {
        LiquidButton(
            onClick = onClick, backdrop = backdrop, modifier = modifier,
            isInteractive = LocalMotion.current, tint = tint, surfaceColor = surface,
            enabled = enabled, shape = shape, minHeight = minHeight, contentPadding = padding,
            role = role, selected = selected, content = label
        )
        return
    }
    val fill = if (tint != Color.Unspecified) tint else surface
    val painted = if (fill == Color.Unspecified) modifier else modifier.background(fill, shape)
    val selection = if (selected == null) Modifier else Modifier.semantics { this.selected = selected }
    Row(
        painted.heightIn(min = minHeight).clip(shape).then(selection)
            .clickable(enabled = enabled, role = role, onClick = onClick).padding(padding),
        horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically,
        content = label
    )
}

@Composable
internal fun GlassIconButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    tint: Color = Color.Unspecified,
    surface: Color = Color.Unspecified,
    glass: Boolean = false,
    content: @Composable () -> Unit
) {
    CatalogAction(onClick, modifier.widthIn(min = 44.dp), enabled, tint = tint, surface = surface,
        minHeight = 44.dp, padding = PaddingValues(0.dp), glass = glass) { content() }
}
