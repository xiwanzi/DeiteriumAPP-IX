package com.deuterium.app.uilab

import android.app.ActivityManager
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent

/** Handles upgrades initiated by older App versions as well as our own installer. */
class InstallCompletionReceiver : BroadcastReceiver() {
    override fun onReceive(context:Context,intent:Intent) {
        if(intent.action==Intent.ACTION_MY_PACKAGE_REPLACED&&!freshUiStarted)closeAppTasks(context)
    }
    companion object {
        // Package replacement starts a new process. If the user already chose
        // Open, a delayed replacement broadcast must not close that fresh UI.
        private var freshUiStarted=false
        fun onUiStarted(){freshUiStarted=true}
        fun closeAppTasks(context:Context) {
            context.getSystemService(ActivityManager::class.java).appTasks
                .filter{it.taskInfo.baseIntent.component?.packageName==context.packageName}
                .forEach{it.finishAndRemoveTask()}
        }
    }
}
