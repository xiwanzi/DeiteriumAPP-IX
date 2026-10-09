package com.deuterium.app.uilab

import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf

@Immutable
data class GlassParameters(
    val blur: Float = 36f,
    val opacity: Float = .65f,
    val refraction: Float = 24f,
    val highlight: Float = .30f
)

@Immutable
data class HeaderGlassParameters(val blur:Float=18f,val fade:Float=32f)

@Immutable
data class GlassMaterials(
    val bottomBar:GlassParameters=GlassParameters(4f,.12f,24f,.30f),
    val overlay:GlassParameters=GlassParameters(36f,.65f,24f,.30f),
    val header:HeaderGlassParameters=HeaderGlassParameters(),
    val chatInputLight:GlassParameters=GlassParameters(18f,.78f,6f,.12f),
    val chatInputDark:GlassParameters=GlassParameters(18f,.68f,6f,.12f),
    val chatPanelLight:GlassParameters=GlassParameters(52f,.92f,2f,.06f),
    val chatPanelDark:GlassParameters=GlassParameters(52f,.89f,2f,.06f)
)

val LocalGlassParameters = staticCompositionLocalOf { GlassParameters() }
val LocalHeaderGlassParameters = staticCompositionLocalOf { HeaderGlassParameters() }
val LocalGlassMaterials=staticCompositionLocalOf{GlassMaterials()}
