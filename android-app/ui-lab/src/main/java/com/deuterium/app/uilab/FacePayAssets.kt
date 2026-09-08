package com.deuterium.app.uilab

import android.content.Context
import android.media.AudioAttributes
import android.media.SoundPool
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathFillType
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import org.json.JSONObject

data class FacePayLayer(val color:Color,val path:Path)
data class FacePayMovie(val viewport:Float,val frames:List<List<FacePayLayer>>,val atlas:ImageBitmap)

/** Cached transparent reference frames and fitted paths contain no reference backdrop. */
object FacePayAssets {
    private val mutex=Mutex()
    @Volatile var movie:FacePayMovie?=null
        private set
    suspend fun load(context:Context):FacePayMovie=mutex.withLock {
        movie ?: withContext(Dispatchers.Default) {
            val root=JSONObject(context.resources.openRawResource(R.raw.face_pay_frames).bufferedReader().use { it.readText() })
            val frames=root.getJSONArray("frames")
            FacePayMovie(root.getInt("viewport").toFloat(),List(frames.length()){frameIndex->
                val source=frames.getJSONArray(frameIndex)
                List(source.length()){layerIndex->
                    val layer=source.getJSONObject(layerIndex);val rgb=layer.getJSONArray("c")
                    val path=Path().apply { fillType=PathFillType.EvenOdd }
                    layer.optJSONArray("e")?.let { ellipses->
                        for(i in 0 until ellipses.length()) {
                            val e=ellipses.getJSONArray(i);val cx=e.getDouble(0).toFloat();val cy=e.getDouble(1).toFloat()
                            val rx=e.getDouble(2).toFloat();val ry=e.getDouble(3).toFloat()
                            path.addOval(androidx.compose.ui.geometry.Rect(cx-rx,cy-ry,cx+rx,cy+ry))
                        }
                    }
                    val contours=layer.getJSONArray("p")
                    for(contourIndex in 0 until contours.length()) {
                        val contour=contours.getJSONArray(contourIndex);val count=contour.length()/2
                        fun x(i:Int)=contour.getDouble((i%count)*2).toFloat()
                        fun y(i:Int)=contour.getDouble((i%count)*2+1).toFloat()
                        if(layer.optBoolean("s",true)) {
                            path.moveTo((x(count-1)+x(0))*.5f,(y(count-1)+y(0))*.5f)
                            for(i in 0 until count)path.quadraticTo(x(i),y(i),(x(i)+x(i+1))*.5f,(y(i)+y(i+1))*.5f)
                        } else {
                            path.moveTo(x(0),y(0));for(i in 1 until count)path.lineTo(x(i),y(i))
                        }
                        path.close()
                    }
                    FacePayLayer(Color(rgb.getInt(0),rgb.getInt(1),rgb.getInt(2),(layer.getDouble("a")*255).toInt()),path)
                }
            },android.graphics.BitmapFactory.decodeResource(context.resources,R.drawable.face_pay_atlas).asImageBitmap())
        }.also { movie=it }
    }
}

object PaymentSound {
    private var pool:SoundPool?=null
    private var sample=0
    @Volatile private var ready=false
    @Synchronized fun prepare(context:Context) {
        if(pool!=null)return
        val sound=SoundPool.Builder().setMaxStreams(1).setAudioAttributes(AudioAttributes.Builder()
            .setUsage(AudioAttributes.USAGE_ASSISTANCE_SONIFICATION).setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION).build()).build()
        pool=sound
        sound.setOnLoadCompleteListener { _,_,status->ready=status==0 }
        sample=sound.load(context.applicationContext,R.raw.face_pay_success,1)
    }
    fun confirm(){
        if(ready){val stream=pool?.play(sample,1f,1f,1,0,1f);android.util.Log.d("DeuteriumPayment","confirmation_sound stream=$stream")}
    }
}
