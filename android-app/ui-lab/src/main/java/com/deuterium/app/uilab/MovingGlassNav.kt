package com.deuterium.app.uilab

import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.Dp
import kotlin.math.*

@Composable
fun MovingGlassNav(labels:List<String>,icons:List<ImageVector>,selected:Int,enabled:Boolean,badges:Set<Int> = emptySet(),onSelect:(Int)->Unit) {
    val motion=LocalMotion.current
    var dragging by remember { mutableStateOf(false) }
    var dragPosition by remember { mutableFloatStateOf(selected.toFloat()) }
    var dragSpeed by remember { mutableFloatStateOf(0f) }
    val target=if(dragging)dragPosition else selected.toFloat()
    val position=animateFloatAsState(target,if(!motion||dragging)snap() else spring(.72f,380f),label="navigation-position")
    val latestSelect by rememberUpdatedState(onSelect)
    BoxWithConstraints(Modifier.fillMaxSize().padding(horizontal=6.dp)) {
        val cell=maxWidth/labels.size
        val cellPx=with(LocalDensity.current){cell.toPx()}
        NavigationIndicator(position,target,dragging,dragSpeed,motion,cell,cellPx)
        Row(Modifier.fillMaxSize().pointerInput(enabled,cellPx) {
            if(enabled) detectHorizontalDragGestures(onDragStart={dragging=true;dragPosition=(it.x/cellPx-.5f).coerceIn(0f,labels.lastIndex.toFloat())},
                onDragEnd={latestSelect(dragPosition.roundToInt().coerceIn(labels.indices));dragging=false;dragSpeed=0f},
                onDragCancel={dragging=false;dragSpeed=0f}) {change,amount->
                change.consume();dragPosition=(dragPosition+amount/cellPx).coerceIn(0f,labels.lastIndex.toFloat());dragSpeed=(abs(amount)/cellPx).coerceAtMost(.3f)
            }
        },verticalAlignment=Alignment.CenterVertically) {
            labels.forEachIndexed { index,title ->
                Column(Modifier.weight(1f).clip(RoundedCornerShape(24.dp)).selectable(index==selected,interactionSource=remember{MutableInteractionSource()},indication=null,enabled=enabled,role=Role.Tab,onClick={latestSelect(index)}).padding(vertical=7.dp),horizontalAlignment=Alignment.CenterHorizontally) {
                    Box(Modifier.size(60.dp,34.dp),contentAlignment=Alignment.Center){Icon(icons[index],null,Modifier.size(22.dp),tint=if(index==selected)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface.copy(alpha=.78f));if(index in badges)Box(Modifier.align(Alignment.TopEnd).padding(end=11.dp,top=3.dp).size(6.dp).background(MaterialTheme.colorScheme.error,CircleShape))}
                    Text(title,Modifier.padding(top=3.dp),style=MaterialTheme.typography.labelMedium,color=if(index==selected)MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface.copy(alpha=.78f))
                }
            }
        }
    }
}

/** Keep the original two springs, but only the indicator observes animated position. */
@Composable
private fun NavigationIndicator(position:State<Float>,target:Float,dragging:Boolean,dragSpeed:Float,motion:Boolean,cell:Dp,cellPx:Float) {
    val stretch by animateFloatAsState(if(motion) (abs(target-position.value)*.30f+if(dragging).15f+dragSpeed else 0f).coerceIn(0f,.68f) else 0f,
        spring(.70f,450f),label="navigation-stretch")
    Box(Modifier.offset(x=(cell-60.dp)/2,y=7.dp).width(60.dp).height(34.dp).graphicsLayer {
        translationX=cellPx*position.value;scaleX=1f+stretch;scaleY=1f-stretch*.17f
    }.background(MaterialTheme.colorScheme.primaryContainer.copy(alpha=.84f),CircleShape))
}
