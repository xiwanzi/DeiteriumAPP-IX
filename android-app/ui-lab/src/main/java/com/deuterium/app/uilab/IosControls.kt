package com.deuterium.app.uilab

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

@Composable
fun PlainButton(onClick:()->Unit,modifier:Modifier=Modifier,enabled:Boolean=true,content:@Composable RowScope.()->Unit) {
    Row(modifier.heightIn(min=44.dp).clip(RoundedCornerShape(12.dp)).clickable(enabled=enabled,role=Role.Button,onClick=onClick).padding(horizontal=12.dp,vertical=8.dp),verticalAlignment=Alignment.CenterVertically,horizontalArrangement=Arrangement.Center) {
        CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.primary.copy(alpha=if(enabled)1f else .35f)){ProvideTextStyle(MaterialTheme.typography.labelLarge){content()}}
    }
}
@Composable
fun SecondaryButton(onClick:()->Unit,modifier:Modifier=Modifier,enabled:Boolean=true,shape:Shape=CircleShape,content:@Composable RowScope.()->Unit) {
    MotionButton(onClick,modifier,enabled,shape,ButtonDefaults.buttonColors(containerColor=MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.7f),contentColor=MaterialTheme.colorScheme.primary),content)
}
@Composable
fun ChoiceChip(selected:Boolean,onClick:()->Unit,label:@Composable ()->Unit,modifier:Modifier=Modifier) {
    Box(modifier.heightIn(min=38.dp).clip(CircleShape).background(if(selected)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.55f))
        .selectable(selected,role=Role.RadioButton,onClick=onClick).padding(horizontal=16.dp,vertical=8.dp),contentAlignment=Alignment.Center) {
        CompositionLocalProvider(LocalContentColor provides if(selected)MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurfaceVariant){ProvideTextStyle(MaterialTheme.typography.bodyMedium){label()}}
    }
}
@Composable
fun AssistChoice(onClick:()->Unit,label:@Composable ()->Unit,modifier:Modifier=Modifier)=ChoiceChip(false,onClick,label,modifier)

@Composable
fun IosSwitch(checked:Boolean,onCheckedChange:(Boolean)->Unit,modifier:Modifier=Modifier,enabled:Boolean=true) {
    val position by animateFloatAsState(if(checked)1f else 0f,if(LocalMotion.current)spring(.8f,600f) else snap(),label="toggle")
    Box(modifier.size(55.dp,44.dp).toggleable(checked,enabled=enabled,role=Role.Switch,onValueChange=onCheckedChange).alpha(if(enabled)1f else .35f),contentAlignment=Alignment.Center) {
        Box(Modifier.size(51.dp,31.dp).background(lerp(MaterialTheme.colorScheme.surfaceVariant,Color(0xFF34C759),position),CircleShape))
        Box(Modifier.align(Alignment.CenterStart).offset(x=(4+20*position).dp).size(27.dp).shadow(2.dp,CircleShape).background(Color.White,CircleShape))
    }
}

@Composable
fun SegmentedControl(labels:List<String>,selected:Int,onSelect:(Int)->Unit,modifier:Modifier=Modifier) {
    val position by animateFloatAsState(selected.toFloat(),if(LocalMotion.current)spring(.85f,500f) else snap(),label="segment")
    BoxWithConstraints(modifier.fillMaxWidth().height(38.dp).clip(RoundedCornerShape(11.dp)).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha=.8f)).padding(3.dp)) {
        val cell=maxWidth/labels.size
        Box(Modifier.width(cell).fillMaxHeight().graphicsLayer{translationX=cell.toPx()*position}.shadow(1.dp,RoundedCornerShape(8.dp)).background(MaterialTheme.colorScheme.surface,RoundedCornerShape(8.dp)))
        Row(Modifier.fillMaxSize()){labels.forEachIndexed { index,title -> Box(Modifier.weight(1f).fillMaxHeight().selectable(selected==index,role=Role.Tab,onClick={onSelect(index)}),contentAlignment=Alignment.Center){Text(title,style=MaterialTheme.typography.bodyMedium,fontWeight=if(index==selected)FontWeight.SemiBold else FontWeight.Normal)} }}
    }
}

@Composable
fun IosSheet(onDismissRequest:()->Unit,containerColor:Color=MaterialTheme.colorScheme.surface,replacementContent:(@Composable ()->Unit)?=null,content:@Composable ColumnScope.()->Unit) {
    IosOverlayHost(onDismissRequest) {
        if(replacementContent!=null){replacementContent();return@IosOverlayHost}
        BoxWithConstraints(Modifier.fillMaxSize().statusBarsPadding().imePadding()) {
            Box(Modifier.matchParentSize().clickable(interactionSource=remember{MutableInteractionSource()},indication=null,onClick=onDismissRequest))
            SoftGlassSurface(Modifier.align(Alignment.BottomCenter).fillMaxWidth().heightIn(max=maxHeight*.94f),radius=30.dp,tint=containerColor) {
                Column(Modifier.navigationBarsPadding()){
                    Box(Modifier.fillMaxWidth().height(27.dp),contentAlignment=Alignment.Center){Box(Modifier.size(34.dp,4.dp).background(MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha=.25f),CircleShape))}
                    content()
                }
            }
        }
    }
}

@Composable
fun IosDialog(onDismissRequest:()->Unit,title:@Composable ()->Unit,text:@Composable ()->Unit,confirmButton:@Composable ()->Unit,dismissButton:(@Composable ()->Unit)?=null) {
    IosOverlayHost(onDismissRequest) {
        SoftGlassSurface(Modifier.widthIn(max=340.dp).fillMaxWidth(.88f),radius=25.dp) {
            Column(horizontalAlignment=Alignment.CenterHorizontally) {
                Column(Modifier.fillMaxWidth().padding(23.dp),horizontalAlignment=Alignment.CenterHorizontally,verticalArrangement=Arrangement.spacedBy(10.dp)) {
                    ProvideTextStyle(MaterialTheme.typography.titleMedium.copy(textAlign=TextAlign.Center)){title()}
                    ProvideTextStyle(MaterialTheme.typography.bodyMedium.copy(textAlign=TextAlign.Center)){Box(Modifier.heightIn(max=380.dp)){text()}}
                }
                HorizontalDivider(color=MaterialTheme.colorScheme.outlineVariant.copy(alpha=.65f))
                Row(Modifier.fillMaxWidth().heightIn(min=51.dp),verticalAlignment=Alignment.CenterVertically){
                    if(dismissButton!=null){Box(Modifier.weight(1f),contentAlignment=Alignment.Center){dismissButton()};Box(Modifier.width(.5.dp).height(51.dp).background(MaterialTheme.colorScheme.outlineVariant))}
                    Box(Modifier.weight(1f),contentAlignment=Alignment.Center){confirmButton()}
                }
            }
        }
    }
}

@Composable
fun IosSlider(value:Float,onValueChange:(Float)->Unit,valueRange:ClosedFloatingPointRange<Float>,modifier:Modifier=Modifier) {
    val latest by rememberUpdatedState(onValueChange)
    val p=((value-valueRange.start)/(valueRange.endInclusive-valueRange.start)).coerceIn(0f,1f)
    BoxWithConstraints(modifier.height(44.dp).semantics { progressBarRangeInfo=ProgressBarRangeInfo(value,valueRange);setProgress{latest(it.coerceIn(valueRange));true} }
        .pointerInput(valueRange){detectTapGestures{latest(valueRange.start+(it.x/size.width).coerceIn(0f,1f)*(valueRange.endInclusive-valueRange.start))}}
        .pointerInput(valueRange){detectHorizontalDragGestures{change,_->change.consume();latest(valueRange.start+(change.position.x/size.width).coerceIn(0f,1f)*(valueRange.endInclusive-valueRange.start))}}) {
        Box(Modifier.align(Alignment.Center).fillMaxWidth().height(4.dp).background(MaterialTheme.colorScheme.surfaceVariant,CircleShape))
        Box(Modifier.align(Alignment.CenterStart).fillMaxWidth(p).height(4.dp).background(MaterialTheme.colorScheme.primary,CircleShape))
        Box(Modifier.align(Alignment.CenterStart).offset(x=(maxWidth-26.dp)*p).size(26.dp).shadow(2.dp,CircleShape).background(Color.White,CircleShape))
    }
}
