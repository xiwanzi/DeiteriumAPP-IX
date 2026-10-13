package com.deuterium.app.uilab

import kotlinx.coroutines.*

/** UI-thread event coalescing. Timers exist only while a requested refresh fails. */
internal class EventRefresh(private val scope:CoroutineScope,private val stopped:()->Boolean) {
    private val jobs=mutableMapOf<String,Job>()
    private val versions=mutableMapOf<String,Long>()
    fun request(key:String,action:suspend ()->Unit){
        if(stopped())return
        versions[key]=(versions[key] ?: 0)+1
        if(jobs[key]?.isActive==true)return
        val job=scope.launch(start=CoroutineStart.LAZY){
            var attempts=0
            try{
                while(isActive&&!stopped()){
                    val version=versions[key]
                    try{action();attempts=0}
                    catch(e:CancellationException){throw e}
                    catch(_:Exception){attempts++;delay((1000L shl attempts.coerceAtMost(6)).coerceAtMost(60_000));continue}
                    if(version==versions[key])break
                }
            }finally{jobs.remove(key);versions.remove(key)}
        }
        jobs[key]=job;job.start()
    }
    fun close(){jobs.values.toList().forEach{it.cancel()};jobs.clear();versions.clear()}
}
