package com.deuterium.app.uilab

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.repeatOnLifecycle
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

@Composable
internal fun OnlinePlayersSheet(state:LabState,onClose:()->Unit,onMessage:(String)->Unit){
    val lifecycle=LocalLifecycleOwner.current
    val scope=rememberCoroutineScope()
    LaunchedEffect(state,lifecycle){
        lifecycle.lifecycle.repeatOnLifecycle(Lifecycle.State.STARTED){
            while(isActive){state.refreshOnlinePlayers();delay(5000)}
        }
    }
    IosSheet(onClose){
        Column(Modifier.padding(horizontal=24.dp).padding(bottom=24.dp)){
            Row(Modifier.fillMaxWidth(),verticalAlignment=Alignment.CenterVertically){
                Text("在线玩家",Modifier.weight(1f),style=MaterialTheme.typography.titleLarge)
                if(state.onlinePlayersRefreshing)CircularProgressIndicator(Modifier.size(20.dp),strokeWidth=2.dp)
            }
            when {
                state.onlinePlayersError!=null->Column(Modifier.padding(top=18.dp)){
                    Text(state.onlinePlayersError!!,color=MaterialTheme.colorScheme.onSurfaceVariant)
                    TextButton(onClick={scope.launch{state.refreshOnlinePlayers()}},enabled=!state.onlinePlayersRefreshing){Text("重试")}
                }
                !state.onlinePlayersKnown->Text("正在读取在线玩家…",Modifier.padding(vertical=18.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)
                state.onlinePlayers.isEmpty()->Text("当前没有玩家在线",Modifier.padding(vertical=18.dp),color=MaterialTheme.colorScheme.onSurfaceVariant)
                else->LazyColumn(Modifier.heightIn(max=420.dp).padding(top=10.dp)){
                    items(state.onlinePlayers,key={it.playerRef}){person->
                        Row(Modifier.fillMaxWidth().padding(vertical=8.dp),verticalAlignment=Alignment.CenterVertically){
                            PlayerAvatar(person.name)
                            Text(person.name,Modifier.weight(1f).padding(start=12.dp))
                            if(person.name==state.userName)Text("我",color=MaterialTheme.colorScheme.onSurfaceVariant)
                            else {
                                if(person.registered)PlainButton({onClose();onMessage(person.name)}){Text("私聊")}
                                PlainButton({onClose();state.mentionPlayer(person.name)}){Text("@ 提及")}
                            }
                        }
                    }
                }
            }
        }
    }
}
