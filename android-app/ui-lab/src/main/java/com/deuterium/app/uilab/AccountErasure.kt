package com.deuterium.app.uilab

import org.json.JSONArray
import org.json.JSONObject

internal const val ErasedAccountName="已注销用户"

/** Also applied to delayed HTTP responses, so a response started before deletion
 * cannot restore a profile or an embedded private-message preview afterwards. */
internal fun redactErasedAccounts(value:Any?,refs:Set<String>) {
    if(refs.isEmpty())return
    when(value){
        is JSONObject->{
            val sender=value.optJSONObject("sender")
            if(value.has("availability")&&sender?.optString("playerRef") in refs){value.put("availability","UNAVAILABLE");value.put("content","")}
            if(value.optJSONObject("forwarded")?.optJSONObject("sender")?.optString("playerRef") in refs)value.put("content","原消息不可见")
            if(value.optString("playerRef") in refs){
                for(key in listOf("gameId","displayName","name"))if(value.has(key))value.put(key,ErasedAccountName)
                for(key in listOf("qq","bio","contactQq"))if(value.has(key))value.put(key,"")
                if(value.has("avatar"))value.put("avatar",JSONObject.NULL)
                value.put("deleted",true)
                if(value.has("registered"))value.put("registered",false)
            }
            value.keys().asSequence().toList().forEach{redactErasedAccounts(value.opt(it),refs)}
        }
        is JSONArray->for(i in 0 until value.length())redactErasedAccounts(value.opt(i),refs)
    }
}
