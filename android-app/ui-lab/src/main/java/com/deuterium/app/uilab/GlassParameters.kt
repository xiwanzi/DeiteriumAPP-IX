package com.deuterium.app.uilab

import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

@Immutable
data class GlassParameters(
    val blur: Float = 4f,
    val opacity: Float = .50f,
    val refraction: Float = 48f,
    val highlight: Float = 1f,
    val refractionWidth: Float = 24f,
    val saturation: Float = 1.50f,
    val depthEffect: Boolean = true,
    val brightness: Float = 0f,
    val plainHighlight: Boolean = false
)

@Immutable
data class HeaderGlassParameters(val blur: Float = 4f, val fade: Float = 32f)

// AndroidLiquidGlass 1.0.6 catalog and the upstream Glass Bottom Sheet tutorial.
internal fun kyantCatalogTint(light: Boolean) =
    if (light) Color(0xFFFAFAFA) else Color(0xFF353944)

internal fun kyantSheetTint(light: Boolean) = if (light) Color.White else kyantCatalogTint(false)

internal fun kyantDialogParameters(light: Boolean) = GlassParameters(
    blur = if (light) 16f else 8f,
    opacity = if (light) .60f else .40f,
    brightness = if (light) .20f else 0f,
    plainHighlight = true
)

private val KyantBottomBar = GlassParameters(
    blur = 1.5f, opacity = .76f, refraction = 16f, highlight = 1f,
    refractionWidth = 16f, saturation = 1.25f, depthEffect = false
)

@Immutable
data class GlassMaterials(
    val bottomBar: GlassParameters = KyantBottomBar,
    val overlay: GlassParameters = GlassParameters(),
    val header: HeaderGlassParameters = HeaderGlassParameters(),
    val chatInputLight: GlassParameters = GlassParameters(
        blur = 6f, opacity = .85f, refraction = 8f, highlight = 1f,
        refractionWidth = 9f, saturation = 3f, depthEffect = false
    ),
    val chatInputDark: GlassParameters = GlassParameters(
        blur = 6f, opacity = .85f, refraction = 8f, highlight = .16f,
        refractionWidth = 9f, saturation = 3f, depthEffect = false
    ),
    val chatPanelLight: GlassParameters = GlassParameters(
        blur = 40f, opacity = .85f, refraction = 8f, highlight = 1f,
        refractionWidth = 24f, saturation = 3f, depthEffect = false
    ),
    val chatPanelDark: GlassParameters = GlassParameters(
        blur = 40f, opacity = .85f, refraction = 8f, highlight = .16f,
        refractionWidth = 24f, saturation = 3f, depthEffect = false
    )
)

internal val TransactionGlass = GlassParameters(
    blur = 26f, opacity = .92f, refraction = 8f, highlight = .32f,
    refractionWidth = 8f, saturation = .90f, depthEffect = false, plainHighlight = true
)

val LocalGlassParameters = staticCompositionLocalOf { GlassParameters() }
val LocalHeaderGlassParameters = staticCompositionLocalOf { HeaderGlassParameters() }
val LocalGlassMaterials = staticCompositionLocalOf { GlassMaterials() }
