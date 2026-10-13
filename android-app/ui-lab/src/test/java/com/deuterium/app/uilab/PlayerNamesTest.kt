package com.deuterium.app.uilab

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class PlayerNamesTest {
    private val uuid="11111111-1111-4111-8111-111111111111"
    @Test fun lateNameCannotReplaceNewerIdentityOrRewriteTextAndNickname(){
        val entry=PlayerNameChange(5,uuid,"player_a","NewName")
        val names=mergePlayerNames(mergePlayerNames(emptyMap(),listOf(entry)),listOf(entry.copy(sequence=3,gameId="OldName")))
        val value=JSONObject("""{"amount":"9007199254740993.25","playerRef":"player_a","gameId":"OldName","displayName":"中文昵称","content":"OldName 原文","snapshot":{"playerRef":"player_a","gameId":"OldName"}}""")
        applyCurrentPlayerNames(value,names)
        assertEquals("NewName",value.getString("gameId"));assertEquals("中文昵称",value.getString("displayName"));assertEquals("OldName 原文",value.getString("content"))
        assertEquals("OldName",value.getJSONObject("snapshot").getString("gameId"));assertEquals("9007199254740993.25",value.getString("amount"))
    }
    @Test fun loadedChatAndStableRouteFollowUuidWithoutLosingDraftOrHistory(){
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Unconfined);val state=LabState(scope,userName="Self")
        try{
            Players.clear();Players.add(PlayerProfile("OldName",playerRef="player_a",displayName="自定义昵称"))
            val route=state.stablePlayerRoute("dm:OldName");assertEquals("dm:player_a",route)
            val messages=state.conversation("OldName");messages.add(ChatLine(1,"OldName","原文不变",false,remoteId="message",serverAt=123,playerRef="player_a"))
            state.followed.add("OldName");state.chatDraft=androidx.compose.ui.text.input.TextFieldValue("正在编辑")
            state.applyPlayerNames(mergePlayerNames(emptyMap(),listOf(PlayerNameChange(1,uuid,"player_a","NewName"))))
            assertSame(messages,state.conversation("NewName"));assertEquals("NewName",messages.single().name);assertEquals("message",messages.single().remoteId)
            assertEquals("原文不变",messages.single().text);assertEquals("正在编辑",state.chatDraft.text);assertEquals("NewName",state.routeGameID(route.substringAfter(':')))
            assertEquals("自定义昵称",Players.single().displayName);assertTrue("NewName" in state.followed)
        }finally{scope.cancel();Players.clear()}
    }
}
