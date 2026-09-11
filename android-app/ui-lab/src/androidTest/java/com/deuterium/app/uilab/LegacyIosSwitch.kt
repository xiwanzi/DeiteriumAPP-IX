// Frozen 2.0.10 reference; only the function name differs from release.
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
fun LegacyIosSwitch(checked:Boolean,onCheckedChange:(Boolean)->Unit,modifier:Modifier=Modifier,enabled:Boolean=true) {
    val position by animateFloatAsState(if(checked)1f else 0f,if(LocalMotion.current)spring(.8f,600f) else snap(),label="toggle")
    Box(modifier.size(55.dp,44.dp).toggleable(checked,enabled=enabled,role=Role.Switch,onValueChange=onCheckedChange).alpha(if(enabled)1f else .35f),contentAlignment=Alignment.Center) {
        Box(Modifier.size(51.dp,31.dp).background(lerp(MaterialTheme.colorScheme.surfaceVariant,Color(0xFF34C759),position),CircleShape))
        Box(Modifier.align(Alignment.CenterStart).offset(x=(4+20*position).dp).size(27.dp).shadow(2.dp,CircleShape).background(Color.White,CircleShape))
    }
}
