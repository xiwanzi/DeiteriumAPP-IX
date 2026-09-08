package com.deuterium.app.uilab

import androidx.compose.animation.core.*
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.*
import androidx.compose.ui.focus.*
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.*
import androidx.compose.ui.graphics.layer.GraphicsLayer
import androidx.compose.ui.input.nestedscroll.*
import androidx.compose.ui.platform.*
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.*
import kotlinx.coroutines.*

@Stable
class PageChromeState(offset:Float=0f,query:String="") {
    var offset by mutableFloatStateOf(offset)
    var query by mutableStateOf(query)
    val searchProgress:Float get()=(offset/68f).coerceIn(0f,1f)
    val titleProgress:Float get()=((offset-68f)/60f).coerceIn(0f,1f)
    fun connection(density:Float)=object:NestedScrollConnection {
        override fun onPreScroll(available:Offset,source:NestedScrollSource):Offset {
            if(available.y>=0)return Offset.Zero
            val consume=minOf(-available.y/density,128f-offset)
            offset+=consume;return Offset(0f,-consume*density)
        }
        override fun onPostScroll(consumed:Offset,available:Offset,source:NestedScrollSource):Offset {
            if(available.y<=0)return Offset.Zero
            val consume=minOf(available.y/density,offset)
            offset-=consume;return Offset(0f,consume*density)
        }
    }
    suspend fun expand(){AnimationState(offset).animateTo(0f,tween(260)){offset=value}}
    companion object { val saver=listSaver<PageChromeState,Any>(save={listOf(it.offset,it.query)},restore={PageChromeState(it[0] as Float,it[1] as String)}) }
}
@Composable fun rememberPageChrome()=rememberSaveable(saver=PageChromeState.saver){PageChromeState()}

@Composable
fun CollapsingChrome(title:String,placeholder:String,state:PageChromeState,statusTop:Dp,backdrop:GraphicsLayer,
    onSearch:()->Unit,actions:@Composable RowScope.()->Unit) {
    val p=state.titleProgress
    val searchHeight=(68f*(1f-state.searchProgress)).dp
    val height=statusTop+48.dp+60.dp*(1f-p)+searchHeight
    val tail=LocalHeaderGlassParameters.current.fade.coerceIn(16f,96f).dp+48.dp
    val colors=MaterialTheme.colorScheme
    BoxWithConstraints(Modifier.fillMaxWidth().height(height+tail)) {
        // The trailing fade extends beneath the toolbar and reaches zero without a hard cutoff.
        GradientGlassHeader(backdrop,Modifier.fillMaxWidth().height(statusTop+48.dp+tail).graphicsLayer{alpha=p})
        val titleSize=(34f-17f*p).sp
        Row(Modifier.fillMaxWidth().padding(top=statusTop).height(48.dp).padding(horizontal=12.dp),verticalAlignment=Alignment.CenterVertically) {
            Spacer(Modifier.weight(1f))
            actions()
        }
        Box(Modifier.fillMaxWidth().padding(top=statusTop+54.dp*(1f-p)+11.dp*p).height((46f-20f*p).dp),contentAlignment=Alignment.CenterStart) {
            Text(title,Modifier.graphicsLayer {
                translationX=24.dp.toPx()*(1f-p)+(this@BoxWithConstraints.constraints.maxWidth-size.width)/2*p
            },fontSize=titleSize,lineHeight=(42f-16f*p).sp,fontWeight=FontWeight.Bold,letterSpacing=(-.6f+.3f*p).sp,maxLines=1)
        }
        if(searchHeight>0.dp)Box(Modifier.align(Alignment.BottomCenter).offset(y=-tail).padding(horizontal=20.dp).fillMaxWidth().height(searchHeight).clipToBounds()) {
            Row(Modifier.fillMaxWidth().height(52.dp).graphicsLayer{alpha=1f-state.searchProgress}.clip(androidx.compose.foundation.shape.RoundedCornerShape(17.dp)).background(colors.surfaceVariant.copy(alpha=.58f)).clickable(onClick=onSearch).padding(horizontal=15.dp),verticalAlignment=Alignment.CenterVertically){Icon(Icons.Outlined.Search,null,Modifier.size(21.dp),tint=colors.onSurfaceVariant);Text(placeholder,Modifier.padding(start=10.dp),style=MaterialTheme.typography.bodyLarge,color=colors.onSurfaceVariant)}
        }
    }
}
