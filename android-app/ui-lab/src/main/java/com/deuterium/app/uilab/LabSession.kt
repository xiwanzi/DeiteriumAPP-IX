package com.deuterium.app.uilab

import androidx.lifecycle.ViewModel
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import androidx.compose.runtime.snapshotFlow
import androidx.compose.runtime.*

/** Retains the authenticated server session across activity configuration changes. */
class LabSession : ViewModel() {
    var launchFinished by mutableStateOf(false)
    private var loadedUser by mutableStateOf<String?>(null)
    fun readyFor(name:String)=loadedUser==name
    private var current: LabState? = null
    private var scope: CoroutineScope? = null
    @OptIn(FlowPreview::class)
    fun get(name:String,followed:Set<String>,saveFollowed:(Set<String>)->Unit,notify:(DemoNotice)->Unit,context:android.content.Context):LabState {
        current?.takeIf { it.userName==name }?.let { return it }
        clearSession()
        val newScope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        scope=newScope
        val state=LabState(newScope,followed,saveFollowed,name,notify,BackendApi.get(context),NotificationPreferences(context,name))
        current=state;state.restoring=true
        newScope.launch {
            // Old ui-lab snapshots contain simulated money and must never enter a live session.
            state.restoring=false;loadedUser=name
            state.connect()
            while(isActive){delay(1000);state.commerce.advanceTime();state.commissions.advanceTime()}
        }
        return state
    }
    fun clearSession(){current?.close();scope?.cancel();scope=null;current=null;loadedUser=null;Players.clear();ShopCatalog.clear()}
    override fun onCleared(){clearSession()}
}
