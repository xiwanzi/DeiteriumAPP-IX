package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.view.PixelCopy
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithCache
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.*
import androidx.compose.ui.graphics.layer.CompositingStrategy
import androidx.compose.ui.graphics.layer.drawLayer
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import org.json.JSONArray
import org.json.JSONObject

/** Rendering fixtures only: no account, preferences, catalog or messaging state is changed. */
object OverdrawDiagnostics {
    fun run(test:Instrumentation,experiments:Boolean=false) {
        val result=Bundle();var host:DeuteriumActivity?=null
        val output=checkNotNull(test.targetContext.getExternalFilesDir(null)).resolve("qa-overdraw-v210").apply{mkdirs()}
        try {
            val launch=checkNotNull(test.targetContext.packageManager.getLaunchIntentForPackage(test.targetContext.packageName))
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK)
            test.sendStatus(1,Bundle().apply{putString("stream","Starting render fixture through launcher\n")})
            val monitor=test.addMonitor(DeuteriumActivity::class.java.name,null,false)
            try {
                val component=checkNotNull(launch.component).flattenToString();require(component.matches(Regex("[A-Za-z0-9_./]+")))
                test.uiAutomation.executeShellCommand("am start -W -a android.intent.action.MAIN -c android.intent.category.LAUNCHER -p ${test.targetContext.packageName} -n $component -f 0x10008000")
                    .use{pipe->java.io.FileInputStream(pipe.fileDescriptor).use{it.readBytes()}}
                host=test.waitForMonitorWithTimeout(monitor,15_000) as? DeuteriumActivity
                checkNotNull(host){"Launcher did not create the render fixture activity within 15 seconds"}
            }finally{test.removeMonitor(monitor)}
            test.sendStatus(1,Bundle().apply{putString("stream","Render fixture activity created\n")})
            val comparisons=JSONArray()
            for(theme in listOf(1,2))for(scene in listOf("background","glass")) {
                var reference:IntArray?=null
                for(mode in if(experiments)listOf("original","cached","shader","separate") else listOf("original","production")) {
                    test.runOnMainSync{host!!.setContent{LabTheme(theme,false){
                        val backdrop=rememberGraphicsLayer();val atmosphere=rememberGraphicsLayer()
                        CompositionLocalProvider(LocalContentColor provides MaterialTheme.colorScheme.onSurface) {
                            Box(Modifier.fillMaxSize().then(if(mode=="production")Modifier else Modifier.background(MaterialTheme.colorScheme.background))) {
                                val foreground:@Composable BoxScope.()->Unit={
                                    if(scene=="glass")Column(Modifier.padding(top=190.dp,start=24.dp,end=24.dp),verticalArrangement=Arrangement.spacedBy(16.dp)) {
                                        repeat(4){index->Surface(color=if(index%2==0)Color(0xFF007AFF) else Color(0xFFE2C599),modifier=Modifier.fillMaxWidth().height(80.dp)) {Text("固定对照 $index",Modifier.padding(20.dp))}}
                                    }
                                }
                                if(mode=="production")GlassScene(atmosphere,backdrop,Modifier.fillMaxSize(),foreground)
                                else if(mode=="separate")DiagnosticSeparateScene(backdrop,foreground)
                                else Box(Modifier.fillMaxSize().recordGlassBackdrop(backdrop)){DiagnosticAtmosphere(mode,Modifier.recordGlassBackdrop(atmosphere));foreground()}
                                if(scene=="glass") {
                                    GradientGlassHeader(backdrop,Modifier.fillMaxWidth().height(160.dp))
                                    LiquidGlass(Modifier.align(Alignment.Center).padding(horizontal=24.dp).fillMaxWidth().height(140.dp),backdrop=backdrop){Text("保留原玻璃",Modifier.align(Alignment.Center))}
                                    if(!experiments)LiquidGlass(Modifier.align(Alignment.BottomCenter).padding(24.dp).fillMaxWidth().height(90.dp),backdrop=atmosphere){Text("原背景采样",Modifier.align(Alignment.Center))}
                                }
                            }
                        }
                    }}}
                    test.waitForIdleSync();Thread.sleep(300)
                    val view=host!!.window.decorView
                    val bitmap=Bitmap.createBitmap(view.width,view.height,Bitmap.Config.ARGB_8888)
                    val copied=CountDownLatch(1);val status=AtomicInteger(-1)
                    PixelCopy.request(host!!.window,bitmap,{status.set(it);copied.countDown()},Handler(Looper.getMainLooper()))
                    check(copied.await(5,TimeUnit.SECONDS)&&status.get()==PixelCopy.SUCCESS)
                    val pixels=IntArray(bitmap.width*bitmap.height);bitmap.getPixels(pixels,0,bitmap.width,0,0,bitmap.width,bitmap.height)
                    check(pixels.any{it!=pixels[0]}){"A uniform frame cannot serve as rendering evidence"}
                    output.resolve("$scene-t$theme-$mode.png").outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)}
                    if(reference==null)reference=pixels else {
                        val baseline=reference!!;var different=0;var maxDelta=0
                        for(i in pixels.indices)if(pixels[i]!=baseline[i]){different++;for(channel in 0..3){val shift=channel*8;maxDelta=maxOf(maxDelta,kotlin.math.abs(((pixels[i] ushr shift)and 255)-((baseline[i] ushr shift)and 255)))}}
                        comparisons.put(JSONObject().put("theme",theme).put("scene",scene).put("mode",mode).put("differentPixels",different).put("maxChannelDelta",maxDelta))
                        if(!experiments)check(different==0){"$scene theme=$theme differs in $different pixels"}
                    }
                    bitmap.recycle()
                }
            }
            output.resolve("comparison.json").writeText(comparisons.toString(2))
            result.putString("comparison",comparisons.toString());test.finish(-1,result)
        }catch(failure:Throwable){result.putString("failure",failure.stackTraceToString());test.finish(1,result)}
        finally{host?.let{activity->test.runOnMainSync{activity.finish()}}}
    }
}

@Composable
private fun DiagnosticSeparateScene(backdrop:androidx.compose.ui.graphics.layer.GraphicsLayer,content:@Composable BoxScope.()->Unit) {
    val base=MaterialTheme.colorScheme.background;val accent=MaterialTheme.colorScheme.primary;val dark=base.red<.5f
    val original=rememberGraphicsLayer();val cached=rememberGraphicsLayer();val foreground=rememberGraphicsLayer()
    Box(Modifier.fillMaxSize().drawWithCache {
        val first=Brush.radialGradient(listOf(accent.copy(alpha=if(dark).03f else .025f),Color.Transparent),center=Offset(size.width*.95f,size.height*.15f),radius=size.width*.95f)
        val second=Brush.radialGradient(listOf(Color(0xFFE2C599).copy(alpha=if(dark).02f else .03f),Color.Transparent),center=Offset(size.width*.04f,size.height*.63f),radius=size.width*.9f)
        val third=Brush.radialGradient(listOf(Color(0xFF98A8DC).copy(alpha=.025f),Color.Transparent),center=Offset(size.width,size.height*.95f),radius=size.width*.9f)
        val dimensions=IntSize(size.width.toInt(),size.height.toInt())
        original.record(this,layoutDirection,dimensions){drawRect(base);drawRect(first);drawRect(second);drawRect(third)}
        cached.compositingStrategy=CompositingStrategy.Offscreen
        cached.record(this,layoutDirection,dimensions){drawLayer(original)}
        onDrawWithContent {
            foreground.record{this@onDrawWithContent.drawContent()}
            backdrop.record{drawLayer(original);drawLayer(foreground)}
            drawLayer(cached);drawLayer(foreground)
        }
    },content=content)
}

@Composable
private fun DiagnosticAtmosphere(mode:String,modifier:Modifier=Modifier) {
    val base=MaterialTheme.colorScheme.background;val accent=MaterialTheme.colorScheme.primary;val dark=base.red<.5f
    val cached=rememberGraphicsLayer()
    Spacer(modifier.fillMaxSize().drawWithCache {
        val first=Brush.radialGradient(listOf(accent.copy(alpha=if(dark).03f else .025f),Color.Transparent),center=Offset(size.width*.95f,size.height*.15f),radius=size.width*.95f)
        val second=Brush.radialGradient(listOf(Color(0xFFE2C599).copy(alpha=if(dark).02f else .03f),Color.Transparent),center=Offset(size.width*.04f,size.height*.63f),radius=size.width*.9f)
        val third=Brush.radialGradient(listOf(Color(0xFF98A8DC).copy(alpha=.025f),Color.Transparent),center=Offset(size.width,size.height*.95f),radius=size.width*.9f)
        if(mode=="cached") {
            cached.compositingStrategy=CompositingStrategy.Offscreen
            cached.record(this,layoutDirection,IntSize(size.width.toInt(),size.height.toInt())){drawRect(base);drawRect(first);drawRect(second);drawRect(third)}
            onDrawBehind{drawLayer(cached)}
        }else if(mode=="shader") {
            var shader:android.graphics.Shader=android.graphics.LinearGradient(0f,0f,1f,0f,base.toArgb(),base.toArgb(),android.graphics.Shader.TileMode.CLAMP)
            for(brush in listOf(first,second,third))shader=android.graphics.ComposeShader(shader,(brush as ShaderBrush).createShader(size),android.graphics.PorterDuff.Mode.SRC_OVER)
            val combined=ShaderBrush(shader)
            onDrawBehind{drawRect(combined)}
        }else onDrawBehind{drawRect(base);drawRect(first);drawRect(second);drawRect(third)}
    })
}
