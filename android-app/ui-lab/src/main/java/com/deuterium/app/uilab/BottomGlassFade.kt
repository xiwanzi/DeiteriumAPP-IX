package com.deuterium.app.uilab

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

/** The reference's bottom fade is outside the glass, over the scrolling page. It takes no space. */
@Composable
internal fun BottomGlassFade(modifier: Modifier = Modifier) {
    val base = if (MaterialTheme.colorScheme.background.red >= .5f) Color.White else Color.Black
    Box(modifier.fillMaxWidth().height(113.dp).background(Brush.verticalGradient(
        0f to base.copy(alpha = 0f),
        .04f to base.copy(alpha = .012f),
        .13f to base.copy(alpha = .035f),
        .21f to base.copy(alpha = .087f),
        .28f to base.copy(alpha = .142f),
        .86f to base.copy(alpha = .726f),
        1f to base.copy(alpha = .84f)
    )))
}
