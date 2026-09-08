package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModelProvider
import kotlinx.coroutines.runBlocking
import java.io.ByteArrayOutputStream
import java.io.File

/** Separate QA APK: real file accounting, real clear controls, no account or business mutation. */
object StorageLayoutCheck {
    fun run(test:Instrumentation,theme:Int,clear:Boolean){
        val result=Bundle()
        val context=test.targetContext
        val sentinel=File(context.filesDir,"qa-storage-preserved-record.json")
        try{
            sentinel.writeText("preserve business records")
            val photo=Bitmap.createBitmap(512,512,Bitmap.Config.ARGB_8888)
            val random=java.util.Random(20100)
            photo.setPixels(IntArray(512*512){0xff000000.toInt() or random.nextInt(0xffffff)},0,512,0,0,512,512)
            val data=ByteArrayOutputStream().also{photo.compress(Bitmap.CompressFormat.PNG,100,it)}.toByteArray();photo.recycle()
            runBlocking{repeat(4){AppImages.get(context).seed("asset:storage_layout_$it",data)}}
            File(context.cacheDir,"storage-layout-temporary.bin").writeBytes(ByteArray(512*1024))
            val download=File(context.filesDir,"updates/downloads/apk-${"c".repeat(20)}.apk").apply{parentFile!!.mkdirs()}
            download.writeBytes(ByteArray(2*1024*1024))
            val before=runBlocking{AppStorage.measure(context)}
            check(before.images>0&&before.updates>=download.length()&&before.temporary>=512*1024)
            val activity=test.startActivitySync(Intent(context,MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as MainActivity
            test.runOnMainSync{
                val updates=ViewModelProvider(activity)[AppUpdates::class.java]
                updates.initialize()
                activity.setContent{
                    LabTheme(theme,false,false){CompositionLocalProvider(LocalAppUpdates provides updates){
                        Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)){
                            StoragePage(100.dp)
                            Row(Modifier.fillMaxWidth().background(MaterialTheme.colorScheme.background).statusBarsPadding().height(50.dp).padding(horizontal=20.dp),verticalAlignment=Alignment.CenterVertically){
                                Icon(Icons.AutoMirrored.Outlined.ArrowBack,"返回",tint=MaterialTheme.colorScheme.primary)
                                Box(Modifier.weight(1f),contentAlignment=Alignment.Center){Text("存储空间",style=MaterialTheme.typography.titleMedium,color=MaterialTheme.colorScheme.onSurface)}
                                Spacer(Modifier.width(24.dp))
                            }
                        }
                    }}
                }
            }
            fun find(node:AccessibilityNodeInfo?,predicate:(String)->Boolean):AccessibilityNodeInfo?{
                if(node==null)return null
                if(predicate(node.text?.toString().orEmpty()))return node
                for(index in 0 until node.childCount)find(node.getChild(index),predicate)?.let{return it}
                return null
            }
            fun waitText(predicate:(String)->Boolean):AccessibilityNodeInfo{
                repeat(60){find(test.uiAutomation.rootInActiveWindow,predicate)?.let{return it};Thread.sleep(100)}
                error("Expected storage control was not displayed")
            }
            fun click(node:AccessibilityNodeInfo){var current:AccessibilityNodeInfo?=node;while(current!=null&&!current.isClickable)current=current.parent;check(current?.performAction(AccessibilityNodeInfo.ACTION_CLICK)==true)}
            fun screenshot(suffix:String){test.waitForIdleSync();Thread.sleep(350);test.uiAutomation.takeScreenshot()!!.let{image->File(context.filesDir,"qa-storage-$suffix.png").outputStream().use{image.compress(Bitmap.CompressFormat.PNG,100,it)};image.recycle()}}
            waitText{it=="图片缓存"};waitText{it=="清理所有缓存"};Thread.sleep(300)
            screenshot(if(theme==2)"dark" else "light")
            if(clear){
                click(waitText{it=="清理所有缓存"});waitText{it=="清理缓存？"};screenshot("confirmation")
                click(waitText{it=="清理"});waitText{it.startsWith("已释放")}
                val after=runBlocking{AppStorage.measure(context)}
                check(after.images==0L&&after.updates==0L&&after.temporary==0L){"Cache was not fully cleared: $after"}
                check(sentinel.readText()=="preserve business records")
                screenshot("cleared")
                result.putString("clear_and_preserve_records","PASS")
            }
            result.putString("storage_layout",if(theme==2)"DARK PASS" else "LIGHT PASS")
            result.putString("status","PASS");test.finish(-1,result)
        }catch(error:Throwable){result.putString("error",error.stackTraceToString());test.finish(1,result)}
        finally{sentinel.delete()}
    }
}
