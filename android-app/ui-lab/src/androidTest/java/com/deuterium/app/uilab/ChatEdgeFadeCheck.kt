package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.rememberGraphicsLayer
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInRoot
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp
import kotlin.math.abs

/** Test the wallpaper restoration itself, not merely symmetry of two flat veils. */
internal object ChatEdgeFadeCheck {
    fun run(test: Instrumentation) {
        check(test.targetContext.packageName.endsWith(".chatmediaqa"))
        val result=Bundle()
        val activity=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
        val mode=mutableIntStateOf(1)
        val phase=mutableIntStateOf(0)
        var seen=-1
        var rootTop=0f
        var height=0
        var dock=0f
        var header=0f
        try {
            test.runOnMainSync { activity.setContent {
                LabTheme(mode.intValue,false) {
                    // Contrasting theme backgrounds expose the old black/white replacement bug.
                    val base=if(mode.intValue==1)Color(0xFFBED5A5) else Color(0xFF193349)
                    MaterialTheme(colorScheme=MaterialTheme.colorScheme.copy(background=base)) {
                        val density=LocalDensity.current
                        val atmosphere=rememberGraphicsLayer()
                        val backdrop=rememberGraphicsLayer()
                        val chrome=WindowInsets.statusBars.asPaddingValues().calculateTopPadding()+48.dp
                        header=with(density){(chrome+24.dp).toPx()}
                        SideEffect { seen=mode.intValue*10+phase.intValue }
                        CompositionLocalProvider(LocalGlassBackdrop provides atmosphere) {
                            Box(Modifier.fillMaxSize().background(base).onGloballyPositioned {
                                rootTop=it.positionInRoot().y;height=it.size.height
                            }) {
                                GlassScene(atmosphere,backdrop,Modifier.fillMaxSize()) {
                                    if(phase.intValue!=0) ChatGlassLayout(body={inset->
                                        dock=with(density){inset.toPx()}
                                        Box(Modifier.fillMaxSize().then(if(phase.intValue==2)
                                            Modifier.background(if(mode.intValue==1)Color.Black else Color.White) else Modifier))
                                    },composer={Box(Modifier.navigationBarsPadding()) {
                                        MessageComposer(TextFieldValue(),{},{},"输入消息")
                                    }})
                                }
                                if(phase.intValue!=0)ChatTopFade(Modifier.fillMaxWidth().height(chrome+24.dp))
                            }
                        }
                    }
                }
            } }
            val d=test.targetContext.resources.displayMetrics.density
            fun shot(theme:Int,p:Int):Bitmap {
                test.runOnMainSync { mode.intValue=theme;phase.intValue=p }
                repeat(50){if(seen!=theme*10+p||height==0)Thread.sleep(60)}
                test.waitForIdleSync();Thread.sleep(650)
                return test.uiAutomation.takeScreenshot()!!.also{image->
                    test.targetContext.filesDir.resolve("chat-wallpaper-$theme-$p.png").outputStream()
                        .use{image.compress(Bitmap.CompressFormat.PNG,100,it)}
                }
            }
            fun rgb(image:Bitmap,x:Int,y:Float):List<Int> {
                val c=image.getPixel(x,(rootTop+y).toInt().coerceIn(0,image.height-1))
                return listOf(android.graphics.Color.red(c),android.graphics.Color.green(c),android.graphics.Color.blue(c))
            }
            for(theme in listOf(1,2)) {
                val base=shot(theme,0)
                val empty=shot(theme,1)
                val foreground=shot(theme,2)
                val start=height-dock
                val protection=1f-(1f-207f/255f)*(1f-.85f)
                val points=listOf(
                    Triple(base.width/3,header-24*d,protection),
                    Triple(base.width/3,header-16*d,protection*157f/207f),
                    Triple(base.width/3,header-8*d,protection*85f/207f),
                    Triple((2*d).toInt(),start+16*d,85f/255f),
                    Triple((2*d).toInt(),start+32*d,157f/255f),
                    Triple((2*d).toInt(),start+48*d,207f/255f))
                val source=if(theme==1)0 else 255
                var backgroundError=0
                var blendError=0f
                for((x,y,alpha) in points) {
                    val b=rgb(base,x,y);val e=rgb(empty,x,y);val a=rgb(foreground,x,y)
                    for(c in 0..2) {
                        backgroundError=maxOf(backgroundError,abs(b[c]-e[c]))
                        blendError=maxOf(blendError,abs(a[c]-(source*(1-alpha)+b[c]*alpha)))
                    }
                }
                check(backgroundError<=3){"Wallpaper changed by fade: theme=$theme error=$backgroundError"}
                check(blendError<=5){"Fade does not restore the sampled background: theme=$theme error=$blendError"}
                val below=rgb(foreground,base.width/3,header+8*d)
                check(below.all{abs(it-source)<=2}){"Top fade extends beyond toolbar: $below"}
                base.recycle();empty.recycle();foreground.recycle()
                result.putString("theme$theme","PASS original wallpaper retained, background error=$backgroundError, blend error=$blendError, protected toolbar and short fade tail")
            }
            test.finish(-1,result)
        }catch(failure:Throwable){result.putString("failure",failure.stackTraceToString());test.finish(1,result)}
    }
}
