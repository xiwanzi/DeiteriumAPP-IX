package com.deuterium.app.uilab

import android.os.Build
import androidx.compose.animation.core.*
import androidx.compose.foundation.*
import androidx.compose.foundation.gestures.*
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.selection.*
import androidx.compose.foundation.shape.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.*
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.*
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.semantics.*
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.*
import androidx.compose.ui.window.*
import com.kyant.backdrop.catalog.components.LiquidSlider
import com.kyant.backdrop.catalog.components.LiquidToggle
import com.kyant.shapes.Capsule

@Composable
fun PlainButton(
    onClick: () -> Unit, modifier: Modifier = Modifier, enabled: Boolean = true,
    content: @Composable RowScope.() -> Unit
) {
    val action = LocalDialogAction.current
    val colors = MaterialTheme.colorScheme
    val light = colors.background.red >= .5f
    val tint = when (action) {
        DialogAction.Confirm -> if (light) Color(0xFF0088FF) else Color(0xFF0091FF)
        DialogAction.Destructive -> colors.error
        else -> Color.Unspecified
    }
    val surface = when (action) {
        DialogAction.Cancel -> colors.surfaceVariant.copy(alpha = .82f)
        DialogAction.Normal -> Color.Unspecified
        else -> if (enabled) Color.Unspecified else colors.surfaceVariant
    }
    val ink = when (action) {
        DialogAction.Normal -> colors.primary
        DialogAction.Cancel -> colors.onSurface
        else -> Color.White
    }
    CatalogAction(
        onClick, if (action == DialogAction.Normal) modifier else modifier.fillMaxWidth(), enabled,
        tint = if (enabled) tint else Color.Unspecified,
        surface = surface, foreground = ink,
        minHeight = if (action == DialogAction.Normal) 44.dp else 48.dp,
        padding = if (action == DialogAction.Normal) PaddingValues(horizontal = 12.dp, vertical = 8.dp)
            else PaddingValues(horizontal = 10.dp, vertical = 6.dp)
    ) {
        if (action == DialogAction.Normal) content()
        else ProvideTextStyle(MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium)) { content() }
    }
}

@Composable
fun SecondaryButton(
    onClick: () -> Unit, modifier: Modifier = Modifier, enabled: Boolean = true,
    shape: Shape = Capsule(), content: @Composable RowScope.() -> Unit
) {
    MotionButton(onClick, modifier, enabled, shape, ButtonDefaults.buttonColors(
        containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = .55f),
        contentColor = MaterialTheme.colorScheme.onSurface
    ), content)
}

@Composable
fun ChoiceChip(selected: Boolean, onClick: () -> Unit, label: @Composable () -> Unit, modifier: Modifier = Modifier) {
    val colors = MaterialTheme.colorScheme
    CatalogAction(
        onClick, modifier,
        surface = if (selected) colors.primaryContainer else colors.surfaceVariant.copy(alpha = .68f),
        foreground = if (selected) colors.onPrimaryContainer else colors.onSurfaceVariant,
        minHeight = 38.dp, padding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
        role = Role.RadioButton, selected = selected
    ) { ProvideTextStyle(MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Normal)) { label() } }
}

@Composable
fun AssistChoice(onClick: () -> Unit, label: @Composable () -> Unit, modifier: Modifier = Modifier) =
    ChoiceChip(false, onClick, label, modifier)

@Composable
fun IosSwitch(checked: Boolean, onCheckedChange: (Boolean) -> Unit, modifier: Modifier = Modifier, enabled: Boolean = true) {
    val light = MaterialTheme.colorScheme.background.red >= .5f
    val checkedState = rememberUpdatedState(checked)
    val onChange = rememberUpdatedState(onCheckedChange)
    val read = remember { { checkedState.value } }
    val update = remember { { value: Boolean -> onChange.value(value) } }
    if (Build.VERSION.SDK_INT >= 31 && LocalOverlayGlassEnabled.current && LocalMotion.current) {
        LiquidToggle(read, update, rememberControlBackdrop(),
            modifier.alpha(if (enabled) 1f else .38f), isLightTheme = light, enabled = enabled)
        return
    }
    val track = if (light) Color(0xFF787878).copy(alpha = .2f) else Color(0xFF787880).copy(alpha = .36f)
    val accent = if (light) Color(0xFF34C759) else Color(0xFF30D158)
    Box(
        modifier.size(64.dp, 44.dp).alpha(if (enabled) 1f else .38f)
            .toggleable(checked, enabled = enabled, role = Role.Switch, onValueChange = onCheckedChange),
        contentAlignment = Alignment.CenterStart
    ) {
        Box(Modifier.size(64.dp, 28.dp).background(if (checked) accent else track, Capsule()))
        Box(Modifier.offset(x = if (checked) 22.dp else 2.dp).size(40.dp, 24.dp)
            .shadow(2.dp, Capsule()).background(Color.White, Capsule()))
    }
}

@Composable
fun SegmentedControl(labels: List<String>, selected: Int, onSelect: (Int) -> Unit, modifier: Modifier = Modifier) {
    val position by animateFloatAsState(selected.toFloat(),
        if (LocalMotion.current) spring(.85f, 500f) else snap(), label = "segment")
    val colors = MaterialTheme.colorScheme
    val light = colors.background.red >= .5f
    val track = if (light) Color(0xFF787878).copy(alpha = .12f) else Color(0xFF787880).copy(alpha = .24f)
    BoxWithConstraints(modifier.fillMaxWidth().height(38.dp).clip(RoundedCornerShape(11.dp)).background(track).padding(3.dp)) {
        val cell = maxWidth / labels.size
        Box(Modifier.width(cell).fillMaxHeight().graphicsLayer { translationX = cell.toPx() * position }
            .background(colors.surface, RoundedCornerShape(8.dp)))
        Row(Modifier.fillMaxSize()) {
            labels.forEachIndexed { index, title ->
                Box(
                    Modifier.weight(1f).fillMaxHeight().clip(Capsule())
                        .selectable(selected == index, role = Role.Tab, onClick = { onSelect(index) }),
                    contentAlignment = Alignment.Center
                ) {
                    Text(title, style = MaterialTheme.typography.bodyMedium,
                        fontWeight = if (index == selected) FontWeight.SemiBold else FontWeight.Normal)
                }
            }
        }
    }
}

@Composable
fun IosSheet(
    onDismissRequest: () -> Unit,
    containerColor: Color = MaterialTheme.colorScheme.surface,
    replacementContent: (@Composable () -> Unit)? = null,
    surfaceParameters: GlassParameters = LocalGlassParameters.current,
    surfaceRadius: Dp = 44.dp,
    dimAmount: Float = 0f,
    content: @Composable ColumnScope.() -> Unit
) {
    IosOverlayHost(onDismissRequest, dimAmount = dimAmount) {
        if(replacementContent!=null){replacementContent();return@IosOverlayHost}
        BoxWithConstraints(Modifier.fillMaxSize().statusBarsPadding().imePadding()) {
            Box(Modifier.matchParentSize().clickable(interactionSource=remember{MutableInteractionSource()},indication=null,onClick=onDismissRequest))
            SoftGlassSurface(
                Modifier.align(Alignment.BottomCenter).fillMaxWidth().heightIn(max = maxHeight * .94f),
                tint = if (containerColor == MaterialTheme.colorScheme.surface) Color.Unspecified else containerColor,
                parameters = surfaceParameters,
                radius = surfaceRadius
            ) {
                Column(Modifier.navigationBarsPadding()){
                    Box(Modifier.fillMaxWidth().height(27.dp),contentAlignment=Alignment.Center){Box(Modifier.size(34.dp,4.dp).background(MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.25f),CircleShape))}
                    content()
                }
            }
        }
    }
}

@Composable
fun IosDialog(
    onDismissRequest: () -> Unit,
    title: @Composable () -> Unit,
    text: @Composable () -> Unit,
    confirmButton: @Composable () -> Unit,
    dismissButton: (@Composable () -> Unit)? = null,
    destructive: Boolean = false
) {
    val light = MaterialTheme.colorScheme.background.red >= .5f
    val colors = MaterialTheme.colorScheme
    IosOverlayHost(onDismissRequest, dimAmount = if (light) .23f else .40f) {
        KyantDialogGlass(Modifier.widthIn(max = 360.dp).fillMaxWidth(.88f)) {
            Column {
                Box(Modifier.fillMaxWidth().padding(start = 28.dp, top = 24.dp, end = 28.dp, bottom = 12.dp)) {
                    CompositionLocalProvider(LocalContentColor provides colors.onSurface) {
                        ProvideTextStyle(MaterialTheme.typography.titleLarge.copy(
                            fontSize = 24.sp, lineHeight = 31.sp, fontWeight = FontWeight.Medium,
                            textAlign = TextAlign.Start
                        )) { title() }
                    }
                }
                Box(Modifier.fillMaxWidth().heightIn(max = 360.dp)
                    .verticalScroll(rememberScrollState()).padding(horizontal = 24.dp, vertical = 12.dp)) {
                    CompositionLocalProvider(LocalContentColor provides colors.onSurface.copy(alpha = .76f)) {
                        ProvideTextStyle(MaterialTheme.typography.bodyMedium.copy(textAlign = TextAlign.Start)) { text() }
                    }
                }
                Row(Modifier.fillMaxWidth().padding(start = 24.dp, end = 24.dp, top = 12.dp, bottom = 24.dp),
                    horizontalArrangement = Arrangement.spacedBy(16.dp), verticalAlignment = Alignment.CenterVertically) {
                    if (dismissButton != null) {
                        Box(Modifier.weight(1f), contentAlignment = Alignment.Center) {
                            CompositionLocalProvider(LocalDialogAction provides DialogAction.Cancel) { dismissButton() }
                        }
                    }
                    val action = if (destructive) DialogAction.Destructive else DialogAction.Confirm
                    Box(Modifier.weight(1f), contentAlignment = Alignment.Center) {
                        CompositionLocalProvider(LocalDialogAction provides action) { confirmButton() }
                    }
                }
            }
        }
    }
}

@Composable
fun IosSlider(value:Float,onValueChange:(Float)->Unit,valueRange:ClosedFloatingPointRange<Float>,modifier:Modifier=Modifier) {
    val current = rememberUpdatedState(value)
    val onChange = rememberUpdatedState(onValueChange)
    val read = remember { { current.value } }
    val update = remember { { next: Float -> onChange.value(next) } }
    val semantics = modifier.height(44.dp).semantics {
        progressBarRangeInfo = ProgressBarRangeInfo(value, valueRange)
        setProgress { update(it.coerceIn(valueRange)); true }
    }
    if (Build.VERSION.SDK_INT >= 31 && LocalOverlayGlassEnabled.current && LocalMotion.current) {
        LiquidSlider(read, update, valueRange, .01f, rememberControlBackdrop(), semantics,
            isLightTheme = MaterialTheme.colorScheme.background.red >= .5f)
        return
    }
    val latest = onChange.value
    val p=((value-valueRange.start)/(valueRange.endInclusive-valueRange.start)).coerceIn(0f,1f)
    BoxWithConstraints(semantics
        .pointerInput(valueRange){detectTapGestures{latest(valueRange.start+(it.x/size.width).coerceIn(0f,1f)*(valueRange.endInclusive-valueRange.start))}}
        .pointerInput(valueRange){detectHorizontalDragGestures{change,_->change.consume();latest(valueRange.start+(change.position.x/size.width).coerceIn(0f,1f)*(valueRange.endInclusive-valueRange.start))}}) {
        Box(Modifier.align(Alignment.Center).fillMaxWidth().height(6.dp).background(MaterialTheme.colorScheme.surfaceVariant,Capsule()))
        Box(Modifier.align(Alignment.CenterStart).fillMaxWidth(p).height(6.dp).background(MaterialTheme.colorScheme.primary,Capsule()))
        Box(Modifier.align(Alignment.CenterStart).offset(x=(maxWidth-40.dp)*p).size(40.dp,24.dp).shadow(2.dp,Capsule()).background(Color.White,Capsule()))
    }
}
