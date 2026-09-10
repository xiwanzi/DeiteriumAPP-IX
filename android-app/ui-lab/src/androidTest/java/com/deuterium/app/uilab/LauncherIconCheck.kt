package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.graphics.Bitmap
import android.os.Bundle
import android.os.Process
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.runBlocking
import org.json.JSONObject
import java.io.IOException

/** Executed from the separate instrumentation APK, with no player or production writes. */
object LauncherIconCheck {
    fun select(test:Instrumentation,id:String) {
        val result=Bundle()
        try {
            val context=test.targetContext
            val choice=checkNotNull(LauncherIconChoice.fromId(id))
            val prefs=context.getSharedPreferences("launcher-icons",Context.MODE_PRIVATE)
            val icons=LauncherIcons(context,BackendApi(context,"https://127.0.0.1:1"))
            runBlocking{icons.applyConfiguration(LauncherIconConfiguration(choice,prefs.getLong("version",0)+1))}
            result.putString("selected",choice.id);result.putString("result","PASS")
        } catch(e:Throwable){result.putString("result","FAIL");result.putString("error",e.stackTraceToString())}
        test.finish(if(result.getString("result")=="PASS")-1 else 1,result)
    }

    fun run(test:Instrumentation) {
        val result=Bundle()
        val context=test.targetContext
        val pm=context.packageManager
        val prefName="launcher-icons-native-qa"
        val instanceField=LauncherIcons::class.java.getDeclaredField("instance").apply{isAccessible=true}
        val previous=instanceField.get(null)
        var host:DeuteriumActivity?=null
        var icons:LauncherIcons?=null
        try {
            context.getSharedPreferences(prefName,Context.MODE_PRIVATE).edit().clear().commit()
            val api=BackendApi(context,"https://127.0.0.1:1")
            var remote=JSONObject().put("iconId","default").put("version",1).put("minAppVersionCode",20800)
            var unavailable=false
            val controller=LauncherIcons(context,api,prefName,fetchConfiguration={if(unavailable)throw IOException("isolated offline fixture");JSONObject(remote.toString())})
            icons=controller
            instanceField.set(null,controller)
            fun assertEntry(choice:LauncherIconChoice) {
                val entries=pm.queryIntentActivities(Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LAUNCHER).setPackage(context.packageName),0)
                check(entries.size==1 && entries.single().activityInfo.name==context.packageName+choice.alias){"Wrong launcher entries: ${entries.map{it.activityInfo.name}}"}
                val target=pm.getActivityInfo(ComponentName(context.packageName,context.packageName+".DeuteriumActivity"),0)
                check(target.enabled){"Real activity was disabled"}
                check(pm.getLaunchIntentForPackage(context.packageName)?.component==controller.component(choice)){"Package launch entry mismatch"}
            }
            runBlocking{controller.sync()};assertEntry(LauncherIconChoice.Default)
            result.putString("default_and_legacy_component","PASS")
            host=test.startActivitySync(Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LAUNCHER).setComponent(controller.component(LauncherIconChoice.Default)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            var theme by mutableIntStateOf(1)
            val activity=host
            test.runOnMainSync{activity.setContent{LabTheme(theme,false,false){Column(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background).padding(36.dp),verticalArrangement=Arrangement.Center,horizontalAlignment=Alignment.CenterHorizontally){
                Text("应用图标 · 2.0.8",style=MaterialTheme.typography.headlineSmall)
                Spacer(Modifier.height(32.dp));AppLauncherIcon(Modifier.size(180.dp));Spacer(Modifier.height(28.dp))
                Text(controller.current.id,color=MaterialTheme.colorScheme.onSurfaceVariant)
                Spacer(Modifier.height(24.dp));Row(horizontalArrangement=Arrangement.spacedBy(22.dp)){AppLauncherIcon(Modifier.size(48.dp));AppLauncherIcon(Modifier.size(32.dp))}
            }}}}
            fun snapshot(name:String) { test.waitForIdleSync();Thread.sleep(350);test.uiAutomation.takeScreenshot().let{bitmap->context.filesDir.resolve("launcher-$name.png").outputStream().use{bitmap.compress(Bitmap.CompressFormat.PNG,100,it)};bitmap.recycle()} }
            snapshot("default-light")
            val pid=Process.myPid()
            remote=remote.put("iconId","anniversary_911").put("version",2)
            runBlocking{controller.sync()};assertEntry(LauncherIconChoice.Anniversary)
            check(Process.myPid()==pid&&!activity.isDestroyed){"Switch disrupted the activity"}
            snapshot("anniversary-light")
            test.runOnMainSync{theme=2};snapshot("anniversary-dark")
            result.putString("batch_roundtrip_and_foreground_survival","PASS")

            runBlocking{check(!controller.applyConfiguration(LauncherIconConfiguration(LauncherIconChoice.Default,1)))}
            runBlocking{check(!controller.applyConfiguration(LauncherIconConfiguration(LauncherIconChoice.Default,2)))}
            assertEntry(LauncherIconChoice.Anniversary)
            unavailable=true;runBlocking{controller.sync()};assertEntry(LauncherIconChoice.Anniversary)
            unavailable=false;remote.put("iconId","unknown").put("version",3);runBlocking{controller.sync()};assertEntry(LauncherIconChoice.Anniversary)
            result.putString("offline_unknown_and_stale_do_not_revert","PASS")

            val restored=LauncherIcons(context,api,prefName)
            runBlocking{restored.restore()};check(restored.current==LauncherIconChoice.Anniversary);assertEntry(LauncherIconChoice.Anniversary)
            result.putString("persisted_selection","PASS")

            val legacy=LauncherIcons(context,api,prefName,batchSwitch=false)
            runBlocking{legacy.applyConfiguration(LauncherIconConfiguration(LauncherIconChoice.Default,4))};assertEntry(LauncherIconChoice.Default)
            runBlocking{legacy.applyConfiguration(LauncherIconConfiguration(LauncherIconChoice.Anniversary,5))};assertEntry(LauncherIconChoice.Anniversary)
            result.putString("individual_api_roundtrip","PASS")

            remote.put("iconId","anniversary_911").put("version",5)
            test.runOnMainSync{activity.finish()};test.waitForIdleSync()
            host=test.startActivitySync(Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LAUNCHER).setComponent(controller.component(LauncherIconChoice.Anniversary)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            check(!host.isFinishing);assertEntry(LauncherIconChoice.Anniversary)
            result.putString("anniversary_entry_opens_same_activity","PASS")
            remote.put("iconId","default").put("version",6);runBlocking{controller.sync()};assertEntry(LauncherIconChoice.Default)
            result.putString("final_default","PASS")
            result.putString("result","PASS")
        } catch(e:Throwable) {
            result.putString("result","FAIL");result.putString("error",e.stackTraceToString())
        } finally {
            runCatching{runBlocking{icons?.applyConfiguration(LauncherIconConfiguration(LauncherIconChoice.Default,100))}}
            test.runOnMainSync{host?.finish()}
            instanceField.set(null,previous)
            context.filesDir.resolve("launcher-check.txt").writeText(result.keySet().joinToString("\n"){"$it=${result.get(it)}"})
            test.finish(if(result.getString("result")=="PASS")-1 else 1,result)
        }
    }
}
