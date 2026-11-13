package com.deuterium.app.uilab

import android.content.Context
import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.*
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.platform.testTag
import com.kyant.backdrop.backdrops.layerBackdrop
import com.kyant.backdrop.backdrops.rememberLayerBackdrop
import com.kyant.backdrop.backdrops.rememberCombinedBackdrop
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.*
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.*
import kotlinx.coroutines.launch
import kotlin.math.*

private data class EmojiGroup(val name:String,val symbol:String,val items:List<String>)
private fun emojiGroup(name:String,symbol:String,items:String)=EmojiGroup(name,symbol,items.split(" "))
private val emojiGroups=listOf(
    emojiGroup("表情","☺","😀 😃 😄 😁 😆 😅 😂 🤣 🥹 😊 😇 🙂 🙃 😉 😌 😍 🥰 😘 😋 😛 😜 🤪 🤨 🧐 🤓 😎 🥳 😏 😒 😔 😞 😢 😭 😤 😠 😡 🤯 😳 🥺 😱 😨 😰 😥 🤔 🤭 🫢 🫣 🤫 😶 😐 😑 🙄 😮 😲 🥱 😴 🤤 😷 🤒 🤕 🤢 🤮 🤧 🥵 🥶 🥴 😵 🤠 🥸 😈 👻 🤖 💩"),
    emojiGroup("手势","✌","👍 👎 👊 ✊ 🤛 🤜 🤞 ✌️ 🤟 🤘 👌 🤌 🤏 🫶 👐 🙌 👏 🤝 🙏 💪 👋 🤚 🖐️ ✋ 🖖 👈 👉 👆 👇 ☝️ 🫵 ✍️ 💅 👀"),
    emojiGroup("心情","♡","❤️ 🧡 💛 💚 💙 💜 🖤 🤍 🤎 💔 ❤️‍🔥 💕 💞 💓 💗 💖 💘 💝 💯 💢 💥 💫 💦 💤 ✨ 🎉 🎊 🎈 🎁 🎀 💐 🌹"),
    emojiGroup("自然","❀","🐱 🐶 🐭 🐹 🐰 🦊 🐻 🐼 🐨 🐯 🦁 🐷 🐸 🐵 🐣 🐧 🐦 🦆 🦋 🐝 🐢 🐬 🐳 🐟 🌿 🍀 🌱 🌴 🌵 🌸 🌼 🌻 🌙 ☀️ ⭐ 🌈 ☁️ ❄️ 🔥 💧"),
    emojiGroup("美食","☕","🍎 🍊 🍋 🍉 🍇 🍓 🫐 🍒 🍑 🥭 🍍 🥥 🥑 🍅 🌽 🥕 🍞 🥐 🥨 🧀 🍔 🍟 🍕 🌮 🍜 🍲 🍣 🍙 🍚 🥟 🍰 🎂 🍦 🍪 🍫 🍬 ☕ 🫖 🧋 🥤"),
    emojiGroup("活动","⚑","🎮 🕹️ 🎲 ♟️ 🎯 ⚽ 🏀 🏈 🎾 🏓 🏸 🏆 🥇 🎸 🎹 🎵 🎶 🎧 🎤 🎬 📷 📱 💻 💡 📚 ✏️ 🏡 🚗 🚲 ✈️ 🚀 ⛏️ 💎 🔑 🛠️ ✅ ❌ ❓ ❗ 🔔")
)

@Composable
internal fun ChatExpressionPanel(state:LabState,onChoose:()->Unit,onView:(ChatViewerRequest)->Unit,
    value:TextFieldValue,onValue:(TextFieldValue)->Unit,onSendSticker:suspend (ChatMedia)->Unit){
    val service=state.chatMedia ?: return
    val colors=MaterialTheme.colorScheme
    val ink=colors.onSurface.copy(alpha=.82f)
    val dark = colors.background.red < .5f
    val selectedTab = if (dark) androidx.compose.ui.graphics.Color(0xFF536F93).copy(alpha = .15f) else colors.onSurface.copy(alpha = .035f)
    val motion=LocalMotion.current
    val scope=rememberCoroutineScope()
    val context=LocalContext.current
    val prefs=remember{context.getSharedPreferences("chat_expression_ui",Context.MODE_PRIVATE)}
    var recent by remember{mutableStateOf(prefs.getString("recent","").orEmpty().split('|').filter{e->emojiGroups.any{e in it.items}}.take(24))}
    var tab by rememberSaveable{mutableIntStateOf(0)}
    var category by rememberSaveable{mutableIntStateOf(0)}
    var managing by rememberSaveable{mutableStateOf(false)}
    val density = LocalDensity.current
    val headerHeight = with(density) { 44.dp.toPx() }
    val footerHeight = with(density) { 56.dp.toPx() }
    val chrome = remember(density, motion) { ExpressionPanelScroll(scope, headerHeight, footerHeight, motion) }
    val gridBackdrop = rememberLayerBackdrop()
    val surfaceBackdrop = LocalExpressionSurface.current
    val controlsBackdrop = if (surfaceBackdrop != null) rememberCombinedBackdrop(surfaceBackdrop, gridBackdrop) else null
    val headerSpace = with(density) { (headerHeight - chrome.headerHidden).toDp() }
    LaunchedEffect(tab, category, managing) { chrome.reveal() }
    fun insert(emoji:String){
        val start=min(value.selection.start,value.selection.end);val end=max(value.selection.start,value.selection.end)
        val text=value.text.replaceRange(start,end,emoji)
        if(text.length>256)return
        onValue(TextFieldValue(text,TextRange(start+emoji.length)))
        recent=(listOf(emoji)+recent.filter{it!=emoji}).take(24)
        prefs.edit().putString("recent",recent.joinToString("|")).apply()
    }
    fun backspace(){
        val start=min(value.selection.start,value.selection.end);val end=max(value.selection.start,value.selection.end)
        if(start!=end)onValue(TextFieldValue(value.text.removeRange(start,end),TextRange(start)))
        else if(start>0){val breaker=android.icu.text.BreakIterator.getCharacterInstance();breaker.setText(value.text);val before=breaker.preceding(start).coerceAtLeast(0);onValue(TextFieldValue(value.text.removeRange(before,start),TextRange(before)))}
    }
    Box(Modifier.fillMaxSize().clipToBounds().nestedScroll(chrome)){
        AnimatedContent(tab,Modifier.fillMaxSize().layerBackdrop(gridBackdrop),transitionSpec={
            if(motion)(fadeIn(tween(150))+slideInHorizontally(tween(220)){if(targetState>initialState)it/8 else -it/8}) togetherWith
                (fadeOut(tween(100))+slideOutHorizontally(tween(220)){if(targetState>initialState)-it/8 else it/8})
            else EnterTransition.None togetherWith ExitTransition.None
        },label="expression pages"){page->
            if(page==0)Column(Modifier.fillMaxSize()){
                Box(Modifier.fillMaxWidth().height(headerSpace).clipToBounds()
                    .testTag("expression-header").semantics {
                        stateDescription = if (chrome.headerHidden >= headerHeight - 1f) "已收起" else "已展开"
                    }) {
                Row(Modifier.fillMaxWidth().requiredHeight(44.dp).graphicsLayer { translationY = -chrome.headerHidden }.horizontalScroll(rememberScrollState()).padding(horizontal=6.dp),verticalAlignment=Alignment.CenterVertically){
                    GlassIconButton({category=-1},Modifier.size(44.dp), surface = if (category == -1) colors.onSurface.copy(alpha = .06f) else androidx.compose.ui.graphics.Color.Unspecified){
                        Icon(Icons.Outlined.History,"最近使用",Modifier.size(20.dp),tint=ink)
                    }
                    emojiGroups.forEachIndexed{index,group->
                        Box(Modifier.size(44.dp).clip(CircleShape).background(if(category==index)colors.onSurface.copy(alpha = .06f) else androidx.compose.ui.graphics.Color.Transparent)
                            .selectable(category==index,role=Role.Tab,onClick={category=index}).semantics{contentDescription=group.name},contentAlignment=Alignment.Center){
                            Text(group.symbol,fontSize=23.sp,color=ink)
                        }
                    }
                }
                }
                key(category){
                    val items=if(category==-1)recent else emojiGroups[category.coerceIn(emojiGroups.indices)].items
                    if(items.isEmpty())Box(Modifier.weight(1f).fillMaxWidth(),contentAlignment=Alignment.Center){Text("用过的表情会出现在这里",style=MaterialTheme.typography.bodySmall,color=ink)}
                    else ExpressionGrid(GridCells.Adaptive(44.dp), 44.dp, Modifier.weight(1f),
                        horizontalPadding = 6.dp, label = "表情网格") { rowHeight ->
                        items(items, key = { it }) { emoji ->
                            val fontSize = with(LocalDensity.current) {
                                (rowHeight - 4.dp).coerceAtLeast(1.dp).toSp().value.coerceAtMost(28f).sp
                            }
                            Box(Modifier.height(rowHeight).clip(RoundedCornerShape(10.dp))
                                .clickable(role = Role.Button) { insert(emoji) }, contentAlignment = Alignment.Center) {
                                Text(emoji, fontSize = fontSize)
                            }
                        }
                    }
                }
            }else Column(Modifier.fillMaxSize()){
                Box(Modifier.fillMaxWidth().height(headerSpace).clipToBounds()) {
                Row(Modifier.fillMaxWidth().requiredHeight(44.dp).graphicsLayer { translationY = -chrome.headerHidden }.padding(horizontal=16.dp),verticalAlignment=Alignment.CenterVertically){
                    Icon(Icons.Outlined.StarBorder,null,Modifier.size(19.dp),tint=ink)
                    Text("我的贴纸",Modifier.weight(1f).padding(start=8.dp),style=MaterialTheme.typography.labelLarge,color=ink)
                    PlainButton({managing=!managing}){Text(if(managing)"完成" else "管理")}
                }
                }
                ExpressionGrid(GridCells.Adaptive(80.dp), 80.dp, Modifier.weight(1f),
                    horizontalPadding = 8.dp, rowGap = 2.dp, label = "贴纸网格") { rowHeight ->
                    items(service.stickers,key={it.id}){media->Box(Modifier.height(rowHeight),contentAlignment=Alignment.Center){
                        ChatMediaTile(service,media,panel=true,onOpen={request->if(managing)onView(request)else scope.launch{
                            runCatching{onSendSticker(media)}.onFailure{state.storageMessage=it.message}
                        }},onLong={onView(ChatViewerRequest(media,null,null){null})})
                        if(managing)GlassIconButton({scope.launch{runCatching{service.deleteSticker(media)}.onFailure{state.storageMessage=it.message}}},Modifier.align(Alignment.TopEnd).size(36.dp)){
                            Icon(Icons.Outlined.Close,"删除贴纸",Modifier.size(22.dp).background(colors.surface.copy(alpha=.68f),CircleShape),tint=colors.error)
                        }
                    }}
                    item{Box(Modifier.height(rowHeight).padding(5.dp).clip(RoundedCornerShape(18.dp)).background(colors.surfaceVariant.copy(alpha=.55f)).clickable(enabled=service.stickers.size<10,role=Role.Button,onClick=onChoose),contentAlignment=Alignment.Center){Icon(Icons.Outlined.Add,"添加贴纸",Modifier.size(26.dp),tint=ink)}}
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Text(if(service.stickers.size>=10)"贴纸已满，删除已有贴纸后可继续添加" else "添加图片或 GIF · 长按预览",Modifier.padding(horizontal=8.dp,vertical=5.dp),style=MaterialTheme.typography.labelSmall,color=ink)
                    }
                }
                service.pending.filter{it.importing}.forEach{job->Row(Modifier.padding(horizontal=12.dp),verticalAlignment=Alignment.CenterVertically){
                    Text(job.error ?: "正在添加…",Modifier.weight(1f),style=MaterialTheme.typography.labelSmall,color=ink)
                    if(job.status=="failed"){PlainButton({service.retry(job)}){Text("重试")};PlainButton({scope.launch{service.dismiss(job)}}){Text("移除")}}
                }}
                if(service.stickerError!=null)Row(Modifier.padding(horizontal=12.dp),verticalAlignment=Alignment.CenterVertically){Text("贴纸暂时无法同步",Modifier.weight(1f),style=MaterialTheme.typography.labelSmall,color=ink);PlainButton({scope.launch{service.refreshStickers()}}){Text("重试")}}
                // This message stays within the scrolling content, clear of the floating controls.
            }
        }
        Row(Modifier.align(Alignment.BottomCenter).fillMaxWidth().height(52.dp)
            .graphicsLayer { translationY = chrome.footerHidden }
            .testTag("expression-footer").padding(horizontal=8.dp),verticalAlignment=Alignment.CenterVertically){
            Spacer(Modifier.width(44.dp))
            Spacer(Modifier.weight(1f))
            ExpressionGlassSurface(controlsBackdrop, Modifier.widthIn(max=232.dp).weight(4f).height(36.dp)) {
            BoxWithConstraints(Modifier.fillMaxSize()) {
                val position by animateFloatAsState(tab.toFloat(),if(motion)tween(230, easing=FastOutSlowInEasing)else snap(),label="expression tab")
                Box(Modifier.offset(x=(maxWidth/2)*position).width(maxWidth/2).fillMaxHeight().padding(3.dp).background(selectedTab,CircleShape))
                Row(Modifier.fillMaxSize()){
                    listOf("表情符号",if(service.stickersLoaded)"贴纸 ${service.stickers.size}/10" else "贴纸").forEachIndexed{index,label->
                        Box(Modifier.weight(1f).fillMaxHeight().selectable(tab==index,role=Role.Tab,onClick={tab=index}),contentAlignment=Alignment.Center){Text(label,fontSize=14.sp,color=colors.onSurface.copy(alpha=if(tab==index).95f else .70f))}
                    }
                }
            }
            }
            Spacer(Modifier.weight(1f))
            ExpressionGlassSurface(controlsBackdrop, Modifier.size(36.dp)) {
            if(tab==0)IconButton(::backspace,Modifier.size(36.dp)){Icon(Icons.Outlined.Backspace,"删除表情",Modifier.size(21.dp),tint=ink)}
            else IconButton(onChoose,Modifier.size(36.dp),enabled=service.stickers.size<10){Icon(Icons.Outlined.AddPhotoAlternate,"导入贴纸",Modifier.size(21.dp),tint=ink)}
            }
        }
    }
}

@Composable
private fun ExpressionGrid(
    columns: GridCells,
    preferredRowHeight: Dp,
    modifier: Modifier,
    horizontalPadding: Dp,
    label: String,
    rowGap: Dp = 0.dp,
    content: LazyGridScope.(Dp) -> Unit
) {
    LazyVerticalGrid(
        columns, modifier.fillMaxWidth().semantics { contentDescription = label },
        contentPadding = PaddingValues(start = horizontalPadding, end = horizontalPadding,
            top = 2.dp, bottom = 56.dp),
        verticalArrangement = Arrangement.spacedBy(rowGap)
    ) { content(preferredRowHeight) }
}
