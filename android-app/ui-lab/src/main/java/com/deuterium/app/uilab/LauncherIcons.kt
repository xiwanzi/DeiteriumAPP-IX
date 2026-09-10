package com.deuterium.app.uilab

import android.content.ComponentName
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext

/** Device-wide selection. Only compiled aliases are switchable; no downloaded code or icon path. */
class LauncherIcons internal constructor(
    context:Context,
    private val api:BackendApi=BackendApi.get(context),
    preferencesName:String="launcher-icons",
    private val batchSwitch:Boolean=Build.VERSION.SDK_INT>=33,
    private val fetchConfiguration:suspend ()->org.json.JSONObject={api.request("GET","/app/launcher-icon",authenticated=false)},
) {
    private val app=context.applicationContext
    private val pm=app.packageManager
    private val prefs=app.getSharedPreferences(preferencesName,Context.MODE_PRIVATE)
    private val mutex=Mutex()
    private val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
    private var initialized=false
    var current by mutableStateOf(readSelection());private set
    internal var lastError:String?=null;private set

    fun component(icon:LauncherIconChoice)=ComponentName(app.packageName,app.packageName+icon.alias)
    private fun enabled(icon:LauncherIconChoice):Boolean=when(pm.getComponentEnabledSetting(component(icon))) {
        PackageManager.COMPONENT_ENABLED_STATE_ENABLED->true
        PackageManager.COMPONENT_ENABLED_STATE_DEFAULT->icon==LauncherIconChoice.Default
        else->false
    }
    private fun readSelection():LauncherIconChoice {
        val saved=LauncherIconChoice.fromId(prefs.getString("iconId","").orEmpty())
        return saved?.takeIf{enabled(it)} ?: LauncherIconChoice.entries.firstOrNull{enabled(it)} ?: LauncherIconChoice.Default
    }

    private fun selectOnDevice(target:LauncherIconChoice) {
        val choices=LauncherIconChoice.entries
        if(choices.all{enabled(it)==(it==target)})return
        val flags=PackageManager.DONT_KILL_APP or if(Build.VERSION.SDK_INT>=30)PackageManager.SYNCHRONOUS else 0
        if(Build.VERSION.SDK_INT>=33 && batchSwitch) {
            pm.setComponentEnabledSettings(choices.map{icon->PackageManager.ComponentEnabledSetting(component(icon),
                if(icon==target)PackageManager.COMPONENT_ENABLED_STATE_ENABLED else PackageManager.COMPONENT_ENABLED_STATE_DISABLED,flags)})
        } else {
            // Enable first: even if the process is interrupted, an entry remains available.
            pm.setComponentEnabledSetting(component(target),PackageManager.COMPONENT_ENABLED_STATE_ENABLED,flags)
            choices.filter{it!=target}.forEach{pm.setComponentEnabledSetting(component(it),PackageManager.COMPONENT_ENABLED_STATE_DISABLED,flags)}
        }
        check(choices.all{enabled(it)==(it==target)}) { "Launcher entry did not change" }
    }

    private suspend fun restoreLocked() {
        if(initialized)return
        val choice=withContext(Dispatchers.IO) {
            val saved=LauncherIconChoice.fromId(prefs.getString("iconId","").orEmpty())
            val target=saved ?: readSelection()
            selectOnDevice(target)
            target
        }
        withContext(Dispatchers.Main.immediate){current=choice}
        initialized=true
    }

    suspend fun restore() {
        try { mutex.withLock{restoreLocked()} }
        catch(e:CancellationException){throw e}
        catch(e:Exception){lastError=e.message}
    }

    /** Serialized fetches keep queued change hints from racing an older response. */
    suspend fun sync() {
        try {
            mutex.withLock {
                restoreLocked()
                val config=LauncherIconConfiguration.parse(fetchConfiguration(),BuildConfig.VERSION_CODE)
                applyLocked(config)
                lastError=null
            }
        } catch(e:CancellationException){throw e}
        catch(e:Exception){lastError=e.message}
    }

    fun requestSync(){scope.launch{sync()}}

    internal suspend fun applyConfiguration(config:LauncherIconConfiguration)=mutex.withLock {
        restoreLocked()
        applyLocked(config)
    }

    private suspend fun applyLocked(config:LauncherIconConfiguration):Boolean {
        val sameOrigin=prefs.getString("origin",null)==api.baseUrl
        val oldVersion=if(sameOrigin)prefs.getLong("version",0) else 0
        if(config.version<oldVersion)return false
        if(config.version==oldVersion&&prefs.getString("iconId",null)!=config.icon.id)return false
        try {
            withContext(Dispatchers.IO) {
                selectOnDevice(config.icon)
                if(!sameOrigin||config.version!=oldVersion||prefs.getString("iconId",null)!=config.icon.id) {
                    check(prefs.edit().putString("origin",api.baseUrl).putString("iconId",config.icon.id).putLong("version",config.version).commit()) { "Cannot persist launcher icon selection" }
                }
            }
        } finally {
            // Lifecycle cancellation can arrive after PackageManager changed the
            // entry. Keep the in-app preview consistent with that actual state.
            withContext(NonCancellable+Dispatchers.Main.immediate){current=readSelection()}
        }
        return true
    }

    companion object {
        @Volatile private var instance:LauncherIcons?=null
        fun get(context:Context):LauncherIcons=instance ?: synchronized(this){instance ?: LauncherIcons(context.applicationContext).also{instance=it}}
    }
}
