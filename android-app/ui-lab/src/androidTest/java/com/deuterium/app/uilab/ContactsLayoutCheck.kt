package com.deuterium.app.uilab

import android.app.Instrumentation
import android.content.Intent
import android.graphics.Rect
import android.os.Bundle
import android.view.accessibility.AccessibilityNodeInfo
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel

/** Synthetic local view state only; does not send messages or create contacts on the server. */
object ContactsLayoutCheck {
    fun run(test:Instrumentation){
        val result=Bundle()
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Main.immediate)
        val previous=Players.toList()
        val opened=java.util.concurrent.atomic.AtomicReference<String>()
        try{
            val activity=test.startActivitySync(Intent(test.targetContext,MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) as MainActivity
            lateinit var state:LabState
            test.runOnMainSync{
                Players.clear();Players.addAll(listOf("Favorite","Older","Newest","NeverChat").map{PlayerProfile(it,playerRef="fixture_$it")})
                state=LabState(scope,initialFollowed=setOf("Favorite"),userName="FixtureSelf")
                state.conversation("Older").add(ChatLine(1,"Older","之前的私聊",false,remoteId="older",serverAt=100))
                state.conversation("Newest").add(ChatLine(2,"你","最新发出的消息",true,remoteId="newest",serverAt=200))
                state.conversation("NeverChat") // An empty thread must not enter the home list.
                activity.setContent{LabTheme(1,false,false){CompositionLocalProvider(androidx.compose.material3.LocalContentColor provides MaterialTheme.colorScheme.onSurface){Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)){InfoPage(state,32.dp,""){} }}}}
            }
            fun find(node:AccessibilityNodeInfo?,testNode:(AccessibilityNodeInfo)->Boolean):AccessibilityNodeInfo?{
                if(node==null)return null
                if(testNode(node))return node
                for(index in 0 until node.childCount)find(node.getChild(index),testNode)?.let{return it}
                return null
            }
            fun named(name:String)=find(test.uiAutomation.rootInActiveWindow){it.text?.toString()?.lineSequence()?.firstOrNull()==name}
            fun waitFor(block:()->Boolean){repeat(60){if(block())return;Thread.sleep(100)};error("Contact UI did not reach expected state")}
            fun top(name:String)=Rect().also{checkNotNull(named(name)){"Missing $name"}.getBoundsInScreen(it)}.top
            waitFor{named("Favorite")!=null&&named("Newest")!=null&&named("Older")!=null}
            check(named("NeverChat")==null){"Unmessaged player appeared on home list"}
            check(top("Favorite")<top("Newest")&&top("Newest")<top("Older")){"Favorite/recency order is wrong"}
            test.runOnMainSync{state.conversation("Older").add(ChatLine(3,"Older","刚刚收到的消息",false,remoteId="incoming",serverAt=300))}
            waitFor{top("Older")<top("Newest")}
            check(top("Favorite")<top("Older"))
            test.runOnMainSync{activity.setContent{LabTheme(1,false,false){CompositionLocalProvider(androidx.compose.material3.LocalContentColor provides MaterialTheme.colorScheme.onSurface){SearchPage("Info",state,{},{opened.set(it)},active=false)}}}}
            waitFor{find(test.uiAutomation.rootInActiveWindow){it.isEditable}!=null}
            val input=checkNotNull(find(test.uiAutomation.rootInActiveWindow){it.isEditable})
            check(input.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT,Bundle().apply{putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE,"NeverChat")}))
            fun resultCard()=find(test.uiAutomation.rootInActiveWindow){node->node.isClickable&&!node.isEditable&&find(node){it.text?.contains("NeverChat")==true}!=null&&find(node){it.text?.contains("QQ")==true}!=null}
            waitFor{resultCard()!=null}
            check(resultCard()!!.performAction(AccessibilityNodeInfo.ACTION_CLICK))
            waitFor{opened.get()=="player:NeverChat"}
            check(recentContacts(Players,state.followed.toSet(),state.directChats).none{it.person.name=="NeverChat"})
            result.putString("history_filter_favorites_and_live_recency","PASS")
            result.putString("search_unmessaged_player","PASS")
            test.finish(-1,result)
        }catch(error:Throwable){
            fun dump(node:AccessibilityNodeInfo?,depth:Int=0):String{if(node==null||depth>25)return "";return " ".repeat(depth)+"text=${node.text} editable=${node.isEditable} clickable=${node.isClickable}\n"+(0 until node.childCount).joinToString(""){dump(node.getChild(it),depth+1)}}
            test.targetContext.filesDir.resolve("qa-contacts-failure.txt").writeText(dump(test.uiAutomation.rootInActiveWindow))
            result.putString("error",error.stackTraceToString());test.finish(1,result)
        }
        finally{scope.cancel();test.runOnMainSync{Players.clear();Players.addAll(previous)}}
    }
}
