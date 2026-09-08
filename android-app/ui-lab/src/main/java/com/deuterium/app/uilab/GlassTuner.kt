package com.deuterium.app.uilab

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import androidx.compose.ui.unit.dp
import kotlin.math.roundToInt
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.contentDescription

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GlassTuner(materials: GlassMaterials, onChange: (GlassMaterials) -> Unit, onClose: () -> Unit) {
    var selected by remember { mutableIntStateOf(0) }
    val parameters=if(selected==0)materials.bottomBar else materials.overlay
    fun update(value:GlassParameters){onChange(if(selected==0)materials.copy(bottomBar=value) else materials.copy(overlay=value))}
    IosSheet(onDismissRequest = onClose, containerColor = MaterialTheme.colorScheme.surface) {
        Column(Modifier.fillMaxWidth().fillMaxHeight(.9f).verticalScroll(rememberScrollState()).padding(horizontal = 24.dp).padding(bottom = 26.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("材质细节", style = MaterialTheme.typography.headlineSmall, modifier = Modifier.weight(1f))
                PlainButton(onClick = { update(if(selected==0)GlassMaterials().bottomBar else GlassMaterials().overlay) }) { Text("还原") }
            }
            Text("两组分别调整与保存，互不影响。", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            SegmentedControl(listOf("底栏玻璃","浮层玻璃"),selected,{selected=it},Modifier.padding(top=18.dp,bottom=16.dp))
            val preview = rememberGraphicsLayer()
            var origin by remember { mutableStateOf(Offset.Zero) }
            Box(Modifier.fillMaxWidth().height(118.dp).onGloballyPositioned { origin = it.positionInRoot() }) {
                Canvas(Modifier.matchParentSize().recordGlassBackdrop(preview)) {
                    drawCircle(Color(0xFF64BCAF), size.height*.48f, Offset(size.width*.23f,size.height*.55f))
                    drawCircle(Color(0xFFDEC087), size.height*.5f, Offset(size.width*.7f,size.height*.42f))
                    drawCircle(Color(0xFFA4B5D8), size.height*.25f, Offset(size.width*.47f,size.height*.75f))
                }
                LiquidGlass(Modifier.align(Alignment.Center).fillMaxWidth(.86f).height(85.dp), backdrop = preview, backdropOrigin = origin,parameters=parameters,enabled=true) {
                    Column(Modifier.align(Alignment.Center), horizontalAlignment = Alignment.CenterHorizontally) {
                        Text("柔光玻璃", style = MaterialTheme.typography.titleLarge)
                        Text("模糊与折射实时预览", style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
            Spacer(Modifier.height(20.dp))
            ParameterSlider("模糊半径", "${parameters.blur.roundToInt()} dp", parameters.blur, 0f..36f) { update(parameters.copy(blur = it)) }
            ParameterSlider("表面不透明度", "${(parameters.opacity*100).roundToInt()}%", parameters.opacity, .08f..1f) { update(parameters.copy(opacity = it)) }
            ParameterSlider("折射强度", parameters.refraction.roundToInt().toString(), parameters.refraction, 0f..24f) { update(parameters.copy(refraction = it)) }
            ParameterSlider("边缘高光", "${(parameters.highlight*100).roundToInt()}%", parameters.highlight, 0f..1f) { update(parameters.copy(highlight = it)) }
            MotionButton(onClose, Modifier.fillMaxWidth().padding(top = 8.dp).height(52.dp)) { Text("完成") }
        }
    }
}

@Composable
private fun ParameterSlider(title: String, value: String, amount: Float, range: ClosedFloatingPointRange<Float>, change: (Float) -> Unit) {
    Column(Modifier.padding(bottom = 10.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(title, style = MaterialTheme.typography.bodyMedium)
            Text(value, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)
        }
        IosSlider(amount, onValueChange = change, valueRange = range, modifier = Modifier.fillMaxWidth().semantics { contentDescription = title })
    }
}
