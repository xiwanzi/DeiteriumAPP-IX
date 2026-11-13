package com.deuterium.app.uilab

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.kyant.backdrop.Backdrop
import com.kyant.backdrop.catalog.components.LiquidBottomTab
import com.kyant.backdrop.catalog.components.LiquidBottomTabs
import com.kyant.shapes.Capsule

internal val BottomNavigationRadius = 32.dp
internal val BottomNavigationInset = 4.dp

@Composable
fun MovingGlassNav(
    labels: List<String>,
    icons: List<ImageVector>,
    selected: Int,
    enabled: Boolean,
    badges: Set<Int> = emptySet(),
    backdrop: Backdrop? = null,
    parameters: GlassParameters = GlassMaterials().bottomBar,
    barHeight: Dp = 64.dp,
    onSelect: (Int) -> Unit
) {
    if (!enabled) return
    val light = MaterialTheme.colorScheme.background.red >= .5f
    val foreground = if (light) Color.Black else Color.White
    val latestSelected = rememberUpdatedState(selected)
    val latestSelect = rememberUpdatedState(onSelect)
    val readSelection = remember { { latestSelected.value } }
    val selectTab = remember { { index: Int -> latestSelect.value(index) } }

    val content: @Composable RowScope.() -> Unit = {
        labels.forEachIndexed { index, title ->
            val active = index == selected
            val selection = if (backdrop == null && active) {
                Modifier.testTag("navigation-selection")
                    .background(foreground.copy(alpha = .1f), Capsule())
            } else Modifier
            LiquidBottomTab(onClick = { selectTab(index) }, selected = active, modifier = selection) {
                Box {
                    Icon(icons[index], contentDescription = null,
                        modifier = Modifier.size(22.dp), tint = foreground)
                    if (index in badges) {
                        Box(
                            Modifier.align(Alignment.TopEnd).padding(start = 14.dp)
                                .size(6.dp).background(MaterialTheme.colorScheme.error, Capsule())
                        )
                    }
                }
                Text(title, color = foreground, maxLines = 1, fontSize = 12.sp,
                    fontWeight = FontWeight.Medium, lineHeight = 16.sp)
            }
        }
    }
    if (backdrop == null) {
        Row(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.surface, Capsule())
            .padding(BottomNavigationInset), verticalAlignment = Alignment.CenterVertically, content = content)
        return
    }

    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
    LiquidBottomTabs(
        selectedTabIndex = readSelection,
        onTabSelected = selectTab,
        backdrop = backdrop,
        tabsCount = labels.size,
        modifier = Modifier.padding(horizontal = 2.dp).graphicsLayer { translationY = 12.dp.toPx() },
        isLightTheme = light,
        interactive = LocalMotion.current,
        barHeight = barHeight - 8.dp,
        blurRadius = parameters.blur,
        refractionWidth = parameters.refractionWidth,
        refractionAmount = parameters.refraction,
        surfaceOpacity = parameters.opacity,
        highlightOpacity = parameters.highlight,
        saturation = parameters.saturation,
        content = content
    )
    }
}
