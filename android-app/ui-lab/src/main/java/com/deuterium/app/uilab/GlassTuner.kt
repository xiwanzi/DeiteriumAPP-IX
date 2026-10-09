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
    val dark=MaterialTheme.colorScheme.background.red<.5f
    fun group(value:GlassMaterials)=when(selected){0->value.bottomBar;1->value.overlay;2->if(dark)value.chatInputDark else value.chatInputLight;else->if(dark)value.chatPanelDark else value.chatPanelLight}
    val parameters=group(materials)
    fun update(value:GlassParameters){onChange(when(selected){
        0->materials.copy(bottomBar=value);1->materials.copy(overlay=value)
        2->if(dark)materials.copy(chatInputDark=value) else materials.copy(chatInputLight=value)
        else->if(dark)materials.copy(chatPanelDark=value) else materials.copy(chatPanelLight=value)
    })}
    IosSheet(onDismissRequest = onClose, containerColor = MaterialTheme.colorScheme.surface) {
      CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface) {
        Column(Modifier.fillMaxWidth().fillMaxHeight(.9f).padding(horizontal = 24.dp).padding(bottom = 12.dp)) {
          Column(Modifier.weight(1f).verticalScroll(rememberScrollState())) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("材质细节", style = MaterialTheme.typography.headlineSmall, modifier = Modifier.weight(1f))
                PlainButton(onClick = { update(group(GlassMaterials())) }) { Text("还原") }
            }
            Text("各区域独立保存，互不影响。", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            SegmentedControl(listOf("底栏","浮层","输入栏","表情面板"),selected,{selected=it},Modifier.padding(top=18.dp,bottom=12.dp))
            if(selected>=2)Text("当前调整${if(dark)"深色" else "浅色"}参数，另一种模式单独保存。",Modifier.padding(bottom=12.dp),style=MaterialTheme.typography.bodySmall,color=MaterialTheme.colorScheme.onSurfaceVariant)
            val preview = rememberGraphicsLayer()
            val background=MaterialTheme.colorScheme.background
            val style=if(selected>=2)chatGlassStyle(materials,selected==3,dark) else null
            var origin by remember { mutableStateOf(Offset.Zero) }
            Box(Modifier.fillMaxWidth().height(118.dp).onGloballyPositioned { origin = it.positionInRoot() }) {
                Canvas(Modifier.matchParentSize().recordGlassBackdrop(preview)) {
                    drawRect(background)
                    drawCircle(Color(0xFF64BCAF), size.height*.48f, Offset(size.width*.23f,size.height*.55f))
                    drawCircle(Color(0xFFDEC087), size.height*.5f, Offset(size.width*.7f,size.height*.42f))
                    drawCircle(Color(0xFFA4B5D8), size.height*.25f, Offset(size.width*.47f,size.height*.75f))
                }
                LiquidGlass(Modifier.align(if(selected==3)Alignment.BottomCenter else Alignment.Center).fillMaxWidth(if(selected==3)1f else .86f).height(85.dp), backdrop = preview, backdropOrigin = origin,parameters=parameters,enabled=true,tint=style?.tint ?: Color.Unspecified,shadeOpacity=style?.shade,topCornersOnly=selected==3) {
                    Column(Modifier.align(Alignment.Center), horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(when(selected){2->"输入消息";3->"表情面板";else->"柔光玻璃"}, style = MaterialTheme.typography.titleLarge)
                        Text("模糊与折射实时预览", style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
            Spacer(Modifier.height(20.dp))
            ParameterSlider("模糊半径", "${parameters.blur.roundToInt()} dp", parameters.blur, 0f..if(selected>=2)72f else 36f) { update(parameters.copy(blur = it)) }
            ParameterSlider("表面不透明度", "${(parameters.opacity*100).roundToInt()}%", parameters.opacity, .08f..1f) { update(parameters.copy(opacity = it)) }
            ParameterSlider("折射强度", parameters.refraction.roundToInt().toString(), parameters.refraction, 0f..24f) { update(parameters.copy(refraction = it)) }
            ParameterSlider("边缘高光", "${(parameters.highlight*100).roundToInt()}%", parameters.highlight, 0f..1f) { update(parameters.copy(highlight = it)) }
          }
            MotionButton(onClose, Modifier.fillMaxWidth().padding(top = 8.dp).height(52.dp)) { Text("完成") }
        }
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
