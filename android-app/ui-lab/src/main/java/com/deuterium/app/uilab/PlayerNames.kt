package com.deuterium.app.uilab

import org.json.JSONArray
import org.json.JSONObject

internal data class PlayerNameChange(val sequence:Long,val uuid:String,val playerRef:String,val gameId:String) {
    fun json()=JSONObject().put("sequence",sequence).put("uuid",uuid).put("playerRef",playerRef).put("gameId",gameId)
    companion object {
        fun parse(value:JSONObject):PlayerNameChange {
            val result=PlayerNameChange(value.getLong("sequence"),value.getString("uuid"),value.getString("playerRef"),value.getString("gameId"))
            require(result.sequence>0&&result.uuid.matches(Regex("[0-9a-f-]{36}"))&&result.playerRef.isNotBlank()&&result.gameId.matches(Regex("[A-Za-z0-9_]{1,32}")))
            return result
        }
    }
}
internal fun mergePlayerNames(previous:Map<String,PlayerNameChange>,changes:List<PlayerNameChange>):Map<String,PlayerNameChange> = previous.toMutableMap().apply {
    changes.forEach { change -> if((get(change.playerRef)?.sequence ?: -1)<change.sequence){put(change.playerRef,change);put(change.uuid,change)} }
}
private val archivedNameFields=setOf("snapshot","snapshotJson","raw","signedPayload","body","content","settings","configuration")
internal fun applyCurrentPlayerNames(value:Any?,names:Map<String,PlayerNameChange>) {
    if(names.isEmpty())return
    when(value){
        is JSONObject -> {
            val ref=listOf("playerRef","senderRef","playerUuid","serverUuid","profileUuid","uuid").map{value.optString(it)}.firstOrNull{it in names}
            names[ref]?.let{current->
                if(!value.optBoolean("deleted")){
                    listOf("gameId","senderName","playerName").filter(value::has).forEach{value.put(it,current.gameId)}
                    if(value.optString("kind")=="PLAYER"){value.put("gameId",current.gameId);if(value.has("displayName"))value.put("displayName",current.gameId)}
                }
            }
            names[value.optString("_cachePeerRef")]?.let{value.put("_cachePeerName",it.gameId)}
            value.keys().asSequence().toList().filterNot{it in archivedNameFields}.forEach{applyCurrentPlayerNames(value.opt(it),names)}
        }
        is JSONArray -> for(i in 0 until value.length())applyCurrentPlayerNames(value.opt(i),names)
    }
}
