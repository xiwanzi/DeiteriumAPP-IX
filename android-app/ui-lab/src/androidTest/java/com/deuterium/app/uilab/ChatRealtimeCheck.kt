package com.deuterium.app.uilab

import android.app.Instrumentation
import android.os.Bundle
import kotlinx.coroutines.*
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okhttp3.mockwebserver.*
import org.json.JSONArray
import org.json.JSONObject
import java.time.Instant
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.atomic.AtomicInteger

/** Local-only recovery checks in the isolated chat QA application. */
object ChatRealtimeCheck {
    fun run(test:Instrumentation){
        val result=Bundle();val context=test.targetContext
        check(context.packageName.endsWith(".chatmediaqa"))
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val server=MockWebServer();var state:LabState?=null
        try {
            val sockets=CopyOnWriteArrayList<WebSocket>()
            val public=CopyOnWriteArrayList<JSONObject>()
            val direct=CopyOnWriteArrayList<JSONObject>()
            val publicReads=AtomicInteger();val privateReads=AtomicInteger()
            val user=JSONObject().put("userId","realtime-qa").put("playerRef","player_realtime_qa").put("gameId","RealtimeQA").put("gameVerified",true)
            val bob=JSONObject().put("playerRef","player_realtime_bob").put("gameId","Bob")
            fun message(id:String,private:Boolean=false)=JSONObject().put("messageId",id).put("sender",bob).put("content",id)
                .put("kind",if(private)"private_chat" else "public_chat").put("sentAt",Instant.now().toString()).apply{if(private)put("conversationId","conv_realtime")}
            fun response(data:JSONObject)=MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",data).toString())
            server.dispatcher=object:Dispatcher(){override fun dispatch(request:RecordedRequest):MockResponse{
                val path=request.requestUrl!!.encodedPath
                if(path=="/api/v1/chat/ws"){
                    check(request.getHeader("X-Deuterium-Private-Chat")=="1")
                    return MockResponse().withWebSocketUpgrade(object:WebSocketListener(){
                        override fun onOpen(socket:WebSocket,response:Response){sockets.add(socket)}
                        override fun onClosing(socket:WebSocket,code:Int,reason:String){socket.close(code,reason)}
                    })
                }
                if(path=="/api/v1/chat/messages"){
                    publicReads.incrementAndGet()
                    val offset=request.requestUrl!!.queryParameter("before")?.toIntOrNull() ?: 0
                    val all=public.reversed();val page=all.drop(offset).take(100)
                    return MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",JSONObject().put("messages",JSONArray(page)))
                        .put("page",JSONObject().put("hasMore",offset+page.size<all.size).put("nextCursor",if(offset+page.size<all.size)(offset+page.size).toString() else JSONObject.NULL)).toString())
                }
                if(path=="/api/v1/chat/conversations")return response(JSONObject().put("items",JSONArray().put(JSONObject().put("conversationId","conv_realtime").put("otherPlayer",bob).put("unreadCount",direct.size).apply{direct.lastOrNull()?.let{put("lastMessage",it)}})))
                if(path=="/api/v1/chat/conversations/conv_realtime/messages"){
                    privateReads.incrementAndGet()
                    val offset=request.requestUrl!!.queryParameter("cursor")?.toIntOrNull() ?: 0
                    val all=direct.reversed();val page=all.drop(offset).take(100)
                    return MockResponse().setHeader("Content-Type","application/json").setBody(JSONObject().put("data",JSONObject().put("items",JSONArray(page)))
                        .put("page",JSONObject().put("hasMore",offset+page.size<all.size).put("nextCursor",if(offset+page.size<all.size)(offset+page.size).toString() else JSONObject.NULL)).toString())
                }
                if(path.endsWith("/read"))return response(JSONObject().put("unreadCount",0))
                if(path=="/api/v1/account/me")return response(JSONObject().put("user",user))
                return response(JSONObject().put("items",JSONArray()).put("players",JSONArray()).put("cursor",0).put("hasMore",false))
            }}
            server.start()
            val api=BackendApi(context,server.url("/").toString())
            val remember=BackendApi::class.java.getDeclaredMethod("rememberSession",JSONObject::class.java).apply{isAccessible=true}
            test.runOnMainSync{remember.invoke(api,JSONObject().put("token","local-realtime-qa").put("user",user))}
            val app=LabState(scope,userName="RealtimeQA",api=api);state=app
            public.add(message("baseline"))
            test.runOnMainSync{LabState::class.java.getDeclaredMethod("openChat").apply{isAccessible=true}.invoke(app)}
            waitFor(test,"initial connection and history"){sockets.isNotEmpty()&&app.chat.any{it.remoteId=="baseline"}}
            fun push(value:JSONObject){check(sockets.last().send(JSONObject().put("type","chat.message").put("payload",JSONObject().put("message",value)).toString()))}
            val instantPublic=message("live-public");public.add(instantPublic);push(instantPublic)
            val instantPrivate=message("live-private",true);direct.add(instantPrivate);push(instantPrivate)
            waitFor(test,"live public and private push",2000){app.chat.any{it.remoteId=="live-public"}&&app.conversation("Bob").any{it.remoteId=="live-private"}}
            runBlocking{withContext(Dispatchers.Main){app.refreshDirect("Bob")}}
            check(app.chat.none{it.remoteId=="live-private"})
            result.putString("live_push","PASS public/private received within 2s; private isolated")

            // The socket stays apparently connected, but the server omits a push.
            public.add(message("missed-public"))
            runBlocking{withContext(Dispatchers.Main){app.refreshPublicChat()}}
            check(app.chat.any{it.remoteId=="missed-public"})
            result.putString("missed_push","PASS HTTP fallback while socket is still connected")

            // A long suspension crosses the 100-message history page boundary.
            test.runOnMainSync{app.foreground=false;app.activeConversation="Bob"}
            repeat(125){public.add(message("gap-$it"));direct.add(message("private-gap-$it",true))}
            direct.add(message("missed-private",true))
            val connections=sockets.size;val reads=publicReads.get()
            runBlocking{withContext(Dispatchers.Main){app.refreshPublicChat()}}
            check(publicReads.get()==reads)
            test.runOnMainSync{app.foreground=true;app.resumeChat()}
            waitFor(test,"resume reconnect and complete catch-up"){sockets.size>connections&&(0 until 125).all{i->app.chat.any{it.remoteId=="gap-$i"}&&app.conversation("Bob").any{it.remoteId=="private-gap-$i"}}&&app.conversation("Bob").any{it.remoteId=="missed-private"}}
            check(app.chat.map{it.remoteId}.distinct().size==app.chat.size)
            result.putString("foreground_recovery","PASS new connection, 125-message gap restored, private catch-up, no duplicates")

            val beforeFailure=sockets.size
            sockets.last().close(1001,"test disconnect")
            waitFor(test,"automatic reconnect",4000){sockets.size>beforeFailure}
            result.putString("disconnect_recovery","PASS reconnect within 4s without restarting application")
            result.putString("status","PASS")
            test.finish(-1,result)
        }catch(error:Throwable){result.putString("error",error.stackTraceToString());test.finish(1,result)}
        finally{state?.close();scope.cancel();server.shutdown()}
    }
    private fun waitFor(test:Instrumentation,label:String,timeout:Long=10000,condition:()->Boolean){
        val until=System.currentTimeMillis()+timeout
        var passed=false
        do{test.runOnMainSync{passed=condition()};if(!passed)Thread.sleep(25)}while(!passed&&System.currentTimeMillis()<until)
        check(passed){label}
    }
}
