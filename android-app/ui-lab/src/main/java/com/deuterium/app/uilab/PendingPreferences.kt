package com.deuterium.app.uilab

import android.content.SharedPreferences
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext

/** SharedPreferences exposes memory before commit returns. Recovery requests must await the
 * same disk barrier as submissions. A failed write retries the same serialized request identity. */
internal class PendingPreferences(private val preferences:SharedPreferences) {
    private val mutex=Mutex()
    private val outstanding=linkedMapOf<String,String>()

    suspend fun save(key:String,value:String?)=mutex.withLock {
        if(value==null){outstanding.remove(key);preferences.edit().remove(key).apply()}
        else {outstanding[key]=value;flush()}
    }

    suspend fun awaitCommitted()=mutex.withLock { flush() }

    private suspend fun flush() {
        if(outstanding.isEmpty())return
        withContext(Dispatchers.IO) {
            val edit=preferences.edit()
            outstanding.forEach{(key,value)->edit.putString(key,value)}
            if(!edit.commit())throw ApiFailure("PERSISTENCE_UNAVAILABLE","请求暂时无法保存，请稍后重试")
        }
        outstanding.clear()
    }
}
