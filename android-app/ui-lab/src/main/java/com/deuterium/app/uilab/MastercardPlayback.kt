package com.deuterium.app.uilab

import android.media.MediaPlayer
import android.os.SystemClock
import androidx.compose.foundation.Canvas
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.*
import androidx.compose.ui.graphics.drawscope.withTransform
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.geometry.Offset
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.ui.platform.LocalContext
import androidx.core.graphics.PathParser
import kotlinx.coroutines.*
import org.json.JSONArray
import org.json.JSONObject

private data class SonicPath(val path:Path,val color:Color)
private data class SonicFrames(val frames:List<List<SonicPath>>,val frameMs:Float,val delayMs:Long)

/** Plays the SDK opening and original audio, then draws this prototype's custom confirmation checkmark. */
@Composable
fun MastercardPlayback(modifier:Modifier,play:Boolean=true,onComplete:()->Unit) {
    val context=LocalContext.current
    val motion=LocalMotion.current
    val complete by rememberUpdatedState(onComplete)
    var frame by remember { mutableIntStateOf(0) }
    var tail by remember { mutableFloatStateOf(0f) }
    var player by remember { mutableStateOf<MediaPlayer?>(null) }
    val frames by produceState<SonicFrames?>(null) {
        value=withContext(Dispatchers.Default) {
            val source=JSONArray(context.resources.openRawResource(R.raw.mastercard_frames).bufferedReader().use { it.readText() })
            val config=JSONObject(context.resources.openRawResource(R.raw.mastercard_timing).bufferedReader().use { it.readText() })
            val pathTag=Regex("<path\\s+[^>]+>")
            val attribute=Regex("([\\w-]+)=\"([^\"]*)\"")
            SonicFrames(List(source.length()){index ->
                pathTag.findAll(source.getString(index)).map { tag ->
                    val attrs=attribute.findAll(tag.value).associate { it.groupValues[1] to it.groupValues[2] }
                    val opacity=attrs["style"]?.substringAfter("opacity:")?.substringBefore(';')?.toFloatOrNull() ?: 1f
                    SonicPath(PathParser.createPathFromPathData(attrs.getValue("d"))!!.asComposePath(),Color(android.graphics.Color.parseColor(attrs.getValue("fill"))).copy(alpha=opacity))
                }.toList()
            },config.getJSONObject("frameTime").getDouble("Frame_1").toFloat(),config.getLong("soundDelayInMillis"))
        }
    }
    DisposableEffect(Unit) { onDispose { player?.release();player=null } }
    LaunchedEffect(frames,play) {
        val data=frames ?: return@LaunchedEffect
        if(!play){frame=19;tail=1f;return@LaunchedEffect}
        val audio=MediaPlayer.create(context,R.raw.mastercard_checkout)
        player=audio
        val audioFinished=CompletableDeferred<Unit>()
        audio?.setOnCompletionListener { audioFinished.complete(Unit) }
        audio?.setOnErrorListener { _,_,_->audioFinished.complete(Unit);true }
        val started=SystemClock.elapsedRealtime()
        val soundJob=launch { delay(data.delayMs);if(audio!=null){audio.start();withTimeoutOrNull(audio.duration+1500L){audioFinished.await()}} }
        val duration=(data.frames.size*data.frameMs).toLong()
        do {
            val elapsed=SystemClock.elapsedRealtime()-started
            frame=if(motion)(elapsed/data.frameMs).toInt().coerceIn(0,19) else 19
            tail=if(motion)((elapsed-19*data.frameMs)/650f).coerceIn(0f,1f) else 1f
            withFrameNanos { }
        }while(SystemClock.elapsedRealtime()-started<duration)
        soundJob.join();complete()
    }
    Canvas(modifier) {
        val paths=frames?.frames?.getOrNull(frame) ?: return@Canvas
        val scale=minOf(size.width,size.height)/1500f
        withTransform({translate((size.width-1500*scale)/2,(size.height-1500*scale)/2);scale(scale,scale,pivot=androidx.compose.ui.geometry.Offset.Zero)}) {
            val t=FastOutSlowInEasing.transform(tail)
            if(t<.32f)paths.forEach { drawPath(it.path,it.color.copy(alpha=it.color.alpha*(1-t/.32f))) }
            if(t>0f) {
                val c=Offset(750f,750f);val green=Color(0xFF30B958)
                if(t<.65f)drawCircle(lerp(Color(0xFFFF5F00),green,(t*2).coerceAtMost(1f)),95f*(1-t/.65f).coerceAtLeast(0f),c)
                drawCircle(green.copy(alpha=.22f*(1-t)),120f+190f*t,c,style=Stroke(10f*(1-t)+1f))
                val a=c+Offset(-180f,0f);val b=c+Offset(-55f,125f);val end=c+Offset(205f,-145f)
                val progress=((t-.12f)/.75f).coerceIn(0f,1f)
                if(progress>0f){val check=Path().apply{moveTo(a.x,a.y);if(progress<.35f){val point=a+(b-a)*(progress/.35f);lineTo(point.x,point.y)}else{lineTo(b.x,b.y);val point=b+(end-b)*((progress-.35f)/.65f);lineTo(point.x,point.y)}};drawPath(check,green,style=Stroke(42f,cap=StrokeCap.Round,join=StrokeJoin.Round))}
            }
        }
    }
}
