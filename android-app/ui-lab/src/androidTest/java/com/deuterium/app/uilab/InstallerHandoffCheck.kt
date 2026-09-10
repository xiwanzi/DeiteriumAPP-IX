package com.deuterium.app.uilab

import android.app.ActivityManager
import android.app.Instrumentation
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Bundle
import androidx.lifecycle.ViewModelProvider
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okio.Buffer
import java.security.MessageDigest

/** Installs only the explicitly supplied local QA APK; no production update is published. */
object InstallerHandoffCheck {
    fun run(test:Instrumentation) {
        val result=Bundle();val server=MockWebServer()
        try {
            val context=test.targetContext
            val file=context.filesDir.resolve("qa-next-update.apk")
            check(file.isFile){"Missing isolated next-version APK"}
            val info=checkNotNull(context.packageManager.getPackageArchiveInfo(file.absolutePath,PackageManager.GET_SIGNING_CERTIFICATES))
            val installed=context.packageManager.getPackageInfo(context.packageName,0)
            check(info.packageName==context.packageName&&info.longVersionCode>installed.longVersionCode)
            check(context.packageManager.canRequestPackageInstalls()){"Emulator must explicitly allow this test's installation source"}
            server.enqueue(MockResponse().setHeader("Content-Type","application/vnd.android.package-archive").setBody(Buffer().write(file.readBytes())));server.start()
            val activity=test.startActivitySync(Intent(context,DeuteriumActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as DeuteriumActivity
            lateinit var updates:AppUpdates
            test.runOnMainSync{updates=ViewModelProvider(activity)[AppUpdates::class.java]}
            fun waitFor(label:String,condition:()->Boolean){repeat(250){if(condition())return;Thread.sleep(100)};error(label)}
            waitFor("Initial update check did not finish"){!updates.checking}
            val sha=MessageDigest.getInstance("SHA-256").digest(file.readBytes()).joinToString(""){"%02x".format(it)}
            val item=AppRelease("qa-install-handoff-${info.longVersionCode}",UpdateKind.Apk,info.versionName ?: "QA",info.longVersionCode,0,"仅模拟器安装接力验证",server.url("/next.apk").toString(),file.length(),sha)
            test.runOnMainSync{updates.start(item)}
            waitFor("Local verified update did not become installable: ${updates.message}"){updates.stage==UpdateStage.Ready||updates.stage==UpdateStage.Failed}
            check(updates.stage==UpdateStage.Ready){updates.message}
            val taskId=activity.taskId
            test.runOnMainSync{InstallCompletionReceiver().onReceive(context,Intent(Intent.ACTION_MY_PACKAGE_REPLACED))}
            check(!activity.isFinishing){"Delayed replacement notification closed newly opened UI"}
            result.putString("late_replacement_notification_preserves_fresh_ui","PASS")
            test.runOnMainSync{updates.install()}
            waitFor("Old app task remained after installer handoff"){activity.isFinishing||activity.isDestroyed}
            waitFor("Old app task remained in recents"){context.getSystemService(ActivityManager::class.java).appTasks.none{it.taskInfo.taskId==taskId}}
            val ready=context.getSharedPreferences("app-updates-v2",0).getBoolean("ready",false)
            check(ready){"Installation handoff discarded the verified update"}
            fun installerReady(node:android.view.accessibility.AccessibilityNodeInfo?):Boolean {
                if(node==null)return false
                if(node.packageName?.contains("packageinstaller")==true&&node.text?.toString()?.lowercase() in setOf("update","install","更新","安装"))return true
                return (0 until node.childCount).any{installerReady(node.getChild(it))}
            }
            // Let the real installer finish copying the granted URI before
            // instrumentation ends (which force-stops its target process).
            waitFor("System installer did not retain an installable APK"){installerReady(test.uiAutomation.rootInActiveWindow)}
            result.putString("verified_apk_handoff_removes_only_app_task","PASS")
            result.putString("verified_update_remains_available_to_installer","PASS")
            result.putString("system_installer_confirmation_remains_visible","PASS")
            result.putLong("fromVersion",installed.longVersionCode);result.putLong("toVersion",info.longVersionCode)
            test.finish(-1,result)
        }catch(error:Throwable){result.putString("error",error.stackTraceToString());test.finish(1,result)}
        finally{server.shutdown()}
    }
}
