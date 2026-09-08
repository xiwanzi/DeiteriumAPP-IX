package com.deuterium.app.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp

enum class DeuteriumThemeMode(val label: String) {
    System("跟随系统"),
    Light("浅色"),
    Dark("深色")
}

enum class DeuteriumColorPreset(val label: String, val swatch: Color) {
    Emerald("翡翠", Color(0xFF1F6F5A)),
    Ocean("海蓝", Color(0xFF286A9C)),
    Amethyst("紫晶", Color(0xFF7654A6)),
    Sakura("樱粉", Color(0xFFB94E75)),
    Amber("琥珀", Color(0xFF9A651A))
}

val LocalDeuteriumColorPreset = compositionLocalOf { DeuteriumColorPreset.Emerald }

private val DeuteriumShapes = Shapes(
    extraSmall = RoundedCornerShape(12.dp),
    small = RoundedCornerShape(12.dp),
    medium = RoundedCornerShape(16.dp),
    large = RoundedCornerShape(24.dp),
    extraLarge = RoundedCornerShape(28.dp)
)

@Composable
fun DeuteriumTheme(
    themeMode: DeuteriumThemeMode = DeuteriumThemeMode.System,
    colorPreset: DeuteriumColorPreset = DeuteriumColorPreset.Emerald,
    dynamicColor: Boolean = false,
    content: @Composable () -> Unit
) {
    val darkTheme = when (themeMode) {
        DeuteriumThemeMode.System -> isSystemInDarkTheme()
        DeuteriumThemeMode.Light -> false
        DeuteriumThemeMode.Dark -> true
    }
    val context = LocalContext.current
    val colors: ColorScheme = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            if (darkTheme) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        }
        darkTheme -> darkColors(colorPreset)
        else -> lightColors(colorPreset)
    }

    CompositionLocalProvider(
        LocalDeuteriumColorPreset provides colorPreset
    ) {
        MaterialTheme(
            colorScheme = colors,
            shapes = DeuteriumShapes,
            typography = MaterialTheme.typography,
            content = content
        )
    }
}

private fun lightColors(preset: DeuteriumColorPreset): ColorScheme {
    return when (preset) {
        DeuteriumColorPreset.Emerald -> lightColorScheme(
            primary = Color(0xFF1F6F5A),
            onPrimary = Color.White,
            primaryContainer = Color(0xFFD6F4E8),
            onPrimaryContainer = Color(0xFF062017),
            secondary = Color(0xFF53645D),
            onSecondary = Color.White,
            secondaryContainer = Color(0xFFD7E8DF),
            onSecondaryContainer = Color(0xFF101F19),
            tertiary = Color(0xFF42657E),
            onTertiary = Color.White,
            tertiaryContainer = Color(0xFFCBE6FF),
            onTertiaryContainer = Color(0xFF001E31),
            background = Color(0xFFFAFDFC),
            onBackground = Color(0xFF191C1B),
            surface = Color(0xFFFAFDFC),
            onSurface = Color(0xFF191C1B),
            surfaceVariant = Color(0xFFDCE5E0),
            onSurfaceVariant = Color(0xFF404944),
            error = Color(0xFFBA1A1A),
            onError = Color.White
        )
        DeuteriumColorPreset.Ocean -> lightColorScheme(
            primary = Color(0xFF286A9C),
            onPrimary = Color.White,
            primaryContainer = Color(0xFFD1E9FF),
            onPrimaryContainer = Color(0xFF001D33),
            secondary = Color(0xFF4F6172),
            onSecondary = Color.White,
            secondaryContainer = Color(0xFFD3E5F7),
            onSecondaryContainer = Color(0xFF0B1D2C),
            tertiary = Color(0xFF6B5B8E),
            onTertiary = Color.White,
            tertiaryContainer = Color(0xFFE9DDFF),
            onTertiaryContainer = Color(0xFF241442),
            background = Color(0xFFFAFCFF),
            onBackground = Color(0xFF181C20),
            surface = Color(0xFFFAFCFF),
            onSurface = Color(0xFF181C20),
            surfaceVariant = Color(0xFFDDE3EA),
            onSurfaceVariant = Color(0xFF41484D),
            error = Color(0xFFBA1A1A),
            onError = Color.White
        )
        DeuteriumColorPreset.Amethyst -> lightColorScheme(
            primary = Color(0xFF7654A6),
            onPrimary = Color.White,
            primaryContainer = Color(0xFFECDCFF),
            onPrimaryContainer = Color(0xFF2A0D4D),
            secondary = Color(0xFF655A6F),
            onSecondary = Color.White,
            secondaryContainer = Color(0xFFECDDF7),
            onSecondaryContainer = Color(0xFF20172A),
            tertiary = Color(0xFF815159),
            onTertiary = Color.White,
            tertiaryContainer = Color(0xFFFFDADD),
            onTertiaryContainer = Color(0xFF331016),
            background = Color(0xFFFFFBFF),
            onBackground = Color(0xFF1D1A20),
            surface = Color(0xFFFFFBFF),
            onSurface = Color(0xFF1D1A20),
            surfaceVariant = Color(0xFFE8E0EB),
            onSurfaceVariant = Color(0xFF4A454E),
            error = Color(0xFFBA1A1A),
            onError = Color.White
        )
        DeuteriumColorPreset.Sakura -> lightColorScheme(
            primary = Color(0xFFB94E75),
            onPrimary = Color.White,
            primaryContainer = Color(0xFFFFD9E4),
            onPrimaryContainer = Color(0xFF3F001D),
            secondary = Color(0xFF74565F),
            onSecondary = Color.White,
            secondaryContainer = Color(0xFFFFD9E4),
            onSecondaryContainer = Color(0xFF2A151C),
            tertiary = Color(0xFF7D5734),
            onTertiary = Color.White,
            tertiaryContainer = Color(0xFFFFDCC0),
            onTertiaryContainer = Color(0xFF2E1600),
            background = Color(0xFFFFFBFF),
            onBackground = Color(0xFF201A1C),
            surface = Color(0xFFFFFBFF),
            onSurface = Color(0xFF201A1C),
            surfaceVariant = Color(0xFFF2DDE3),
            onSurfaceVariant = Color(0xFF514348),
            error = Color(0xFFBA1A1A),
            onError = Color.White
        )
        DeuteriumColorPreset.Amber -> lightColorScheme(
            primary = Color(0xFF9A651A),
            onPrimary = Color.White,
            primaryContainer = Color(0xFFFFDDB5),
            onPrimaryContainer = Color(0xFF301800),
            secondary = Color(0xFF715B41),
            onSecondary = Color.White,
            secondaryContainer = Color(0xFFFDDDBF),
            onSecondaryContainer = Color(0xFF281805),
            tertiary = Color(0xFF58643A),
            onTertiary = Color.White,
            tertiaryContainer = Color(0xFFDCEABC),
            onTertiaryContainer = Color(0xFF161F00),
            background = Color(0xFFFFFBFF),
            onBackground = Color(0xFF201B16),
            surface = Color(0xFFFFFBFF),
            onSurface = Color(0xFF201B16),
            surfaceVariant = Color(0xFFF0E0CF),
            onSurfaceVariant = Color(0xFF4F4539),
            error = Color(0xFFBA1A1A),
            onError = Color.White
        )
    }
}

private fun darkColors(preset: DeuteriumColorPreset): ColorScheme {
    return when (preset) {
        DeuteriumColorPreset.Emerald -> darkColorScheme(
            primary = Color(0xFFB8DCCA),
            onPrimary = Color(0xFF003829),
            primaryContainer = Color(0xFF00513D),
            onPrimaryContainer = Color(0xFFD6F4E8),
            secondary = Color(0xFFBBCBC2),
            onSecondary = Color(0xFF26332D),
            secondaryContainer = Color(0xFF3C4A43),
            onSecondaryContainer = Color(0xFFD7E8DF),
            tertiary = Color(0xFFAACCE8),
            onTertiary = Color(0xFF0D344D),
            tertiaryContainer = Color(0xFF294C65),
            onTertiaryContainer = Color(0xFFCBE6FF),
            background = Color(0xFF101413),
            onBackground = Color(0xFFE0E3E0),
            surface = Color(0xFF101413),
            onSurface = Color(0xFFE0E3E0),
            surfaceVariant = Color(0xFF404944),
            onSurfaceVariant = Color(0xFFC0C9C3),
            error = Color(0xFFFFB4AB),
            onError = Color(0xFF690005)
        )
        DeuteriumColorPreset.Ocean -> darkColorScheme(
            primary = Color(0xFF9CCBFF),
            onPrimary = Color(0xFF003354),
            primaryContainer = Color(0xFF064B77),
            onPrimaryContainer = Color(0xFFD1E9FF),
            secondary = Color(0xFFB7C9DA),
            onSecondary = Color(0xFF21323F),
            secondaryContainer = Color(0xFF384956),
            onSecondaryContainer = Color(0xFFD3E5F7),
            tertiary = Color(0xFFD6C0F8),
            onTertiary = Color(0xFF3B2B5D),
            tertiaryContainer = Color(0xFF524374),
            onTertiaryContainer = Color(0xFFE9DDFF),
            background = Color(0xFF101419),
            onBackground = Color(0xFFE0E3E8),
            surface = Color(0xFF101419),
            onSurface = Color(0xFFE0E3E8),
            surfaceVariant = Color(0xFF41484D),
            onSurfaceVariant = Color(0xFFC1C7CE),
            error = Color(0xFFFFB4AB),
            onError = Color(0xFF690005)
        )
        DeuteriumColorPreset.Amethyst -> darkColorScheme(
            primary = Color(0xFFD7BAFF),
            onPrimary = Color(0xFF45236E),
            primaryContainer = Color(0xFF5D3B8B),
            onPrimaryContainer = Color(0xFFECDCFF),
            secondary = Color(0xFFCFC1D9),
            onSecondary = Color(0xFF352B3F),
            secondaryContainer = Color(0xFF4C4256),
            onSecondaryContainer = Color(0xFFECDDF7),
            tertiary = Color(0xFFF3B8C0),
            onTertiary = Color(0xFF4C252C),
            tertiaryContainer = Color(0xFF663B43),
            onTertiaryContainer = Color(0xFFFFDADD),
            background = Color(0xFF15121A),
            onBackground = Color(0xFFE8E0EB),
            surface = Color(0xFF15121A),
            onSurface = Color(0xFFE8E0EB),
            surfaceVariant = Color(0xFF4A454E),
            onSurfaceVariant = Color(0xFFCBC3CE),
            error = Color(0xFFFFB4AB),
            onError = Color(0xFF690005)
        )
        DeuteriumColorPreset.Sakura -> darkColorScheme(
            primary = Color(0xFFFFB0C8),
            onPrimary = Color(0xFF650033),
            primaryContainer = Color(0xFF913F5C),
            onPrimaryContainer = Color(0xFFFFD9E4),
            secondary = Color(0xFFE2BDC7),
            onSecondary = Color(0xFF412931),
            secondaryContainer = Color(0xFF5A3F47),
            onSecondaryContainer = Color(0xFFFFD9E4),
            tertiary = Color(0xFFF0BD92),
            onTertiary = Color(0xFF49290C),
            tertiaryContainer = Color(0xFF63401F),
            onTertiaryContainer = Color(0xFFFFDCC0),
            background = Color(0xFF181215),
            onBackground = Color(0xFFEDE0E3),
            surface = Color(0xFF181215),
            onSurface = Color(0xFFEDE0E3),
            surfaceVariant = Color(0xFF514348),
            onSurfaceVariant = Color(0xFFD5C2C7),
            error = Color(0xFFFFB4AB),
            onError = Color(0xFF690005)
        )
        DeuteriumColorPreset.Amber -> darkColorScheme(
            primary = Color(0xFFFFB957),
            onPrimary = Color(0xFF522D00),
            primaryContainer = Color(0xFF754A00),
            onPrimaryContainer = Color(0xFFFFDDB5),
            secondary = Color(0xFFE0C2A4),
            onSecondary = Color(0xFF402C16),
            secondaryContainer = Color(0xFF58422A),
            onSecondaryContainer = Color(0xFFFDDDBF),
            tertiary = Color(0xFFC0CE93),
            onTertiary = Color(0xFF2B350F),
            tertiaryContainer = Color(0xFF414C24),
            onTertiaryContainer = Color(0xFFDCEABC),
            background = Color(0xFF17130E),
            onBackground = Color(0xFFEDE1D5),
            surface = Color(0xFF17130E),
            onSurface = Color(0xFFEDE1D5),
            surfaceVariant = Color(0xFF4F4539),
            onSurfaceVariant = Color(0xFFD2C4B4),
            error = Color(0xFFFFB4AB),
            onError = Color(0xFF690005)
        )
    }
}
