package com.deuterium.app.uilab

import android.app.Activity
import android.app.Instrumentation
import android.graphics.Bitmap
import android.graphics.Rect
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.os.Build
import android.view.FrameMetrics
import android.view.Window
import android.os.HandlerThread
import org.json.JSONArray
import org.json.JSONObject
import android.view.PixelCopy
import android.view.accessibility.AccessibilityNodeInfo
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

/** Platform APIs only, so the same capture tool can inspect the old and new R8 target APKs. */
object OverdrawPageCapture {
    fun run(test:Instrumentation,label:String,benchmark:Boolean=false) {
        require(label in setOf("before","after"))
        val output=checkNotNull(test.targetContext.getExternalFilesDir(null)).resolve("qa-overdraw-v210/${if(benchmark)"metrics" else "actual"}-$label").apply{mkdirs()}
        var host:Activity?=null
        try {
            fun shell(command:String){test.uiAutomation.executeShellCommand(command).use{pipe->java.io.FileInputStream(pipe.fileDescriptor).use{it.readBytes()}}}
            val pkg=test.targetContext.packageName
            val component=checkNotNull(test.targetContext.packageManager.getLaunchIntentForPackage(pkg)?.component).flattenToString()
            require(component.matches(Regex("[A-Za-z0-9_./]+")))
            val monitor=test.addMonitor("$pkg.DeuteriumActivity",null,false)
            try {
                shell("am start -W -a android.intent.action.MAIN -c android.intent.category.LAUNCHER -p $pkg -n $component -f 0x10008000")
                host=test.waitForMonitorWithTimeout(monitor,15_000)
                checkNotNull(host){"App activity did not start"}
            }finally{test.removeMonitor(monitor)}
            fun find(node:AccessibilityNodeInfo?,label:String,bottom:Boolean=true):AccessibilityNodeInfo? {
                if(node==null)return null
                if(node.text?.toString()==label){
                    val bounds=Rect().also{node.getBoundsInScreen(it)}
                    if(!bounds.isEmpty&&(bounds.centerY()>host!!.window.decorView.height*.8f)==bottom)return node
                }
                for(i in 0 until node.childCount)find(node.getChild(i),label,bottom)?.let{return it}
                return null
            }
            val pages=if(benchmark)listOf("信息" to "info") else listOf("商城" to "shop","市场" to "market","我的" to "profile","信息" to "info")
            for((title,name) in pages) {
                if(Build.VERSION.SDK_INT>=33)test.uiAutomation.clearCache()
                var tab:AccessibilityNodeInfo?=null
                repeat(80){if(tab==null){tab=find(test.uiAutomation.rootInActiveWindow,title);if(tab==null)Thread.sleep(100)}}
                checkNotNull(tab){"Missing navigation tab: $title"}
                val bounds=Rect().also{tab!!.getBoundsInScreen(it)}
                var parent=tab!!.parent
                while(parent!=null){
                    val area=Rect().also{parent!!.getBoundsInScreen(it)}
                    if(area.width()>bounds.width()&&area.width()<host!!.window.decorView.width/2&&area.contains(bounds)){bounds.set(area);break}
                    parent=parent.parent
                }
                shell("input touchscreen tap ${bounds.centerX()} ${bounds.centerY()}")
                test.waitForIdleSync();Thread.sleep(900)
                val view=host!!.window.decorView;val bitmap=Bitmap.createBitmap(view.width,view.height,Bitmap.Config.ARGB_8888)
                val copied=CountDownLatch(1);val code=AtomicInteger(-1)
                PixelCopy.request(host!!.window,bitmap,{code.set(it);copied.countDown()},Handler(Looper.getMainLooper()))
                check(copied.await(5,TimeUnit.SECONDS)&&code.get()==PixelCopy.SUCCESS)
                output.resolve("$name.png").outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)}
                bitmap.recycle()
                if(Build.VERSION.SDK_INT>=33)test.uiAutomation.clearCache()
                check(find(test.uiAutomation.rootInActiveWindow,title,bottom=false)!=null){"Navigation did not reach $title"}
                test.sendStatus(1,Bundle().apply{putString("stream","Captured $name\n")})
            }
            if(benchmark) {
                val window=host!!.window;val view=window.decorView
                fun swipe(up:Boolean) {
                    val x=(view.width*.83f).toInt()
                    val from=view.height*(if(up).75f else .35f);val to=view.height*(if(up).35f else .75f)
                    shell("input touchscreen swipe $x ${from.toInt()} $x ${to.toInt()} 480")
                    SystemClock.sleep(150)
                }
                swipe(true);swipe(false)
                val frames=java.util.Collections.synchronizedList(mutableListOf<Pair<Long,Long>>())
                val thread=HandlerThread("overdraw-frame-metrics").apply{start()}
                val listener=Window.OnFrameMetricsAvailableListener{_,metrics,_->frames.add(metrics.getMetric(FrameMetrics.TOTAL_DURATION) to metrics.getMetric(FrameMetrics.GPU_DURATION))}
                test.runOnMainSync{window.addOnFrameMetricsAvailableListener(listener,Handler(thread.looper))}
                try{repeat(4){swipe(true);swipe(false)}}finally{test.runOnMainSync{window.removeOnFrameMetricsAvailableListener(listener)};thread.quitSafely();thread.join(1000)}
                val samples=frames.toList();check(samples.size>=60){"Insufficient frame samples: ${samples.size}"}
                val gpu=samples.map{it.second}.filter{it>0}.sorted();val total=samples.map{it.first}.sorted()
                fun percentile(values:List<Long>,p:Double)=if(values.isEmpty())-1.0 else values[((values.size-1)*p).toInt()]/1_000_000.0
                val json=JSONObject().put("frames",samples.size).put("gpuSamples",gpu.size)
                    .put("gpuMedianMs",percentile(gpu,.5)).put("gpuP95Ms",percentile(gpu,.95))
                    .put("totalMedianMs",percentile(total,.5)).put("totalP95Ms",percentile(total,.95))
                    .put("totalNs",JSONArray(samples.map{it.first})).put("gpuNs",JSONArray(samples.map{it.second}))
                output.resolve("frames.json").writeText(json.toString(2))
                test.sendStatus(1,Bundle().apply{putString("stream","Measured ${samples.size} frames\n")})
            }
            test.finish(-1,Bundle().apply{putString("overdrawPageCapture","PASS")})
        }catch(failure:Throwable){test.finish(1,Bundle().apply{putString("failure",failure.stackTraceToString())})}
        // Instrumentation completion can close the task; the host reopens the user's App afterwards.
    }
}
