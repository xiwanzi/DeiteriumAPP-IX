package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.os.Bundle
import android.os.SystemClock
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveableStateHolder
import kotlinx.coroutines.CompletableDeferred
import java.util.concurrent.atomic.AtomicInteger

/** Real composable timing with controlled responses, never a real financial API. */
object PaymentTimingCheck {
    fun run(test:Instrumentation){
        val report=Bundle()
        try{
            val activity=test.startActivitySync(Intent(test.targetContext,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            fun contains(node:AccessibilityNodeInfo?,text:String):Boolean{
                if(node==null)return false
                if(node.text?.contains(text)==true)return true
                return (0 until node.childCount).any{contains(node.getChild(it),text)}
            }
            fun shows(text:String)=contains(test.uiAutomation.rootInActiveWindow,text)
            fun waitFor(message:String,timeout:Long=6000,condition:()->Boolean){
                val until=SystemClock.elapsedRealtime()+timeout
                while(SystemClock.elapsedRealtime()<until){if(condition())return;Thread.sleep(30)}
                error(message)
            }
            for(kind in listOf("fast","slow","failure","recovered")){
                val calls=AtomicInteger()
                val visible=mutableStateOf(true)
                val recovered=mutableStateOf<Boolean?>(null)
                val response=CompletableDeferred<Boolean>()
                if(kind=="fast")response.complete(true)
                if(kind=="failure")response.complete(false)
                val started=SystemClock.elapsedRealtime()
                val title="QA payment $kind"
                test.runOnMainSync{activity.setContent{
                    key(kind){LabTheme(1,kind!="slow",false){Surface(color=MaterialTheme.colorScheme.background){
                        val holder=rememberSaveableStateHolder()
                        if(visible.value)holder.SaveableStateProvider("payment"){
                            PaymentExperience(100,"本机时序夹具",title,onCommit={calls.incrementAndGet();response.await()},embedded=true,confirmed=recovered.value,onClose={})
                        }
                    }}}
                }}
                waitFor("Request did not start during scan",900){calls.get()==1&&shows("请看向屏幕")}
                check(!shows(title)){"Success appeared before the scan finished"}
                if(kind=="fast"){
                    test.runOnMainSync{visible.value=false};test.waitForIdleSync()
                    test.runOnMainSync{visible.value=true}
                }
                if(kind=="slow"){
                    waitFor("Slow response did not stay pending"){shows("正在确认")}
                    check(!shows(title));response.complete(true)
                }
                if(kind=="recovered"){
                    test.runOnMainSync{recovered.value=true}
                    response.complete(false)
                }
                if(kind=="failure")waitFor("Failed request did not display failure"){shows("支付未完成")}
                else waitFor("Confirmed response did not display success"){shows(title)}
                check(SystemClock.elapsedRealtime()-started>=950){"The minimum scan was skipped"}
                check(calls.get()==1){"Recomposition or restored state repeated submission"}
                if(kind=="failure")check(!shows(title))
                report.putString(kind,"PASS")
            }
            test.finish(-1,report)
        }catch(error:Throwable){report.putString("error",error.stackTraceToString());test.finish(1,report)}
    }
}
